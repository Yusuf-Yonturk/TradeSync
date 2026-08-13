package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"tradesync/engine/orderbook"
)

const offsetGroup = "engine-group"

func main() {
	brokers := strings.Split(envOr("KAFKA_BROKERS", "localhost:9092"), ",")
	snapshotPath := os.Getenv("SNAPSHOT_PATH")
	snapshotEvery := envDuration("SNAPSHOT_INTERVAL", 30*time.Second)

	producer, err := newKafkaProducer(brokers)
	if err != nil {
		log.Fatalf("producer olusturulamadi: %v", err)
	}
	defer producer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Defter bellekte tutulur. Acilista varsa diskteki goruntuden, yoksa orders
	// topic'inin basindan yeniden kurulur. Daha once islenmis kayitlar
	// (offset < committed) sadece deftere uygulanir, sonuclari tekrar yayinlanmaz.
	offsets := kadm.NewClient(producer)
	committed, err := fetchCommitted(ctx, offsets)
	if err != nil {
		log.Fatalf("islenmis offsetler okunamadi: %v", err)
	}

	books := make(map[string]*orderbook.OrderBook)
	// deftere uygulanmis son kaydin bir sonraki offseti
	position := make(map[int32]int64)

	if snap := usableSnapshot(snapshotPath, committed); snap != nil {
		books = restoreBooks(*snap)
		for p, next := range snap.Offsets {
			position[p] = next
		}
		log.Printf("defter goruntuden kuruldu, emir=%d offsetler=%v", len(snap.Orders), snap.Offsets)
	}

	start, err := startOffsets(ctx, offsets, position)
	if err != nil {
		log.Fatalf("topic offsetleri okunamadi: %v", err)
	}
	consumer, err := newKafkaClient(brokers, start)
	if err != nil {
		log.Fatalf("consumer olusturulamadi: %v", err)
	}
	defer consumer.Close()

	if len(committed) > 0 {
		log.Printf("defter yeniden kuruluyor, islenmis offsetler=%v", committed)
	}

	replayed := 0
	rebuilding := len(committed) > 0
	// canli islenmis ama offseti henuz kaydedilmemis kayitlar
	pending := make(map[int32]int64)
	lastSnapshot := time.Now()
	snapshotDirty := false

	log.Println("engine basliyor")

	for {
		// Kayit gelmese de dongu ara sira doner ki son durum goruntuye yazilsin.
		pollCtx, cancelPoll := context.WithTimeout(ctx, snapshotEvery)
		fetches := consumer.PollFetches(pollCtx)
		cancelPoll()

		if ctx.Err() != nil {
			log.Println("engine durdu")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			if errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.Printf("fetch hatasi topic=%s partition=%d: %v", t, p, err)
		})

		liveSeen := false

		fetches.EachRecord(func(r *kgo.Record) {
			live := isLive(committed, r.Partition, r.Offset)
			position[r.Partition] = r.Offset + 1
			snapshotDirty = true
			if live {
				pending[r.Partition] = r.Offset + 1
				liveSeen = true
			} else {
				replayed++
			}

			var incoming IncomingOrder
			if err := json.Unmarshal(r.Value, &incoming); err != nil {
				if live {
					log.Printf("order parse hatasi: %v", err)
				}
				return
			}
			if err := validateIncoming(incoming); err != nil {
				if live {
					log.Printf("gecersiz order atlandi id=%s: %v", incoming.ID, err)
				}
				return
			}

			book := getOrCreateBook(books, incoming.Symbol)
			res := process(book, incoming)
			if !live {
				return
			}
			trades := res.trades

			// Her order eklendiğinde tahtanın güncel halini kafka'ya bas
			depth := book.Snapshot(incoming.Symbol, 15)
			publishDepth(ctx, producer, depth)
			publishOrderUpdates(ctx, producer, res.updates)

			if len(trades) == 0 {
				return
			}

			outgoing := toOutgoingTrades(trades, r.Partition, r.Offset)
			publishTrades(ctx, producer, outgoing)

			log.Printf("eslesti symbol=%s trades=%d", incoming.Symbol, len(trades))
		})

		if rebuilding && liveSeen {
			rebuilding = false
			log.Printf("defter yeniden kuruldu, %d eski kayit uygulandi", replayed)
		}

		if len(pending) > 0 {
			// Sonuclar kafka'ya ulasmadan offset ilerletilmez; aksi halde cokme sonrasi kayip olur.
			if err := producer.Flush(ctx); err != nil {
				log.Printf("flush hatasi, offset ilerletilmedi: %v", err)
				continue
			}
			if err := commitProcessed(ctx, offsets, pending); err != nil {
				log.Printf("offset kayit hatasi: %v", err)
				continue
			}
			for p, next := range pending {
				committed[p] = next
			}
			clear(pending)
		}

		// Buraya gelindiyse defterdeki her seyin sonucu yayinlanmis ve kaydedilmistir.
		if snapshotPath != "" && snapshotDirty && time.Since(lastSnapshot) >= snapshotEvery {
			if err := saveSnapshot(snapshotPath, takeSnapshot(books, position)); err != nil {
				log.Printf("goruntu yazilamadi: %v", err)
			} else {
				snapshotDirty = false
			}
			lastSnapshot = time.Now()
		}
	}
}

// Diskteki goruntuyu okur; yoksa ya da kullanilamazsa nil doner.
func usableSnapshot(path string, committed map[int32]int64) *snapshot {
	if path == "" {
		return nil
	}
	snap, err := loadSnapshot(path)
	if err != nil {
		log.Printf("goruntu okunamadi, topic bastan okunacak: %v", err)
		return nil
	}
	if snap == nil {
		return nil
	}
	if !snapshotUsable(*snap, committed) {
		log.Printf("goruntu kullanilamaz (offsetler=%v, islenmis=%v), topic bastan okunacak", snap.Offsets, committed)
		return nil
	}
	return snap
}

// Kayit daha once islenmemisse true doner.
func isLive(committed map[int32]int64, partition int32, offset int64) bool {
	return offset >= committed[partition]
}

func getOrCreateBook(books map[string]*orderbook.OrderBook, symbol string) *orderbook.OrderBook {
	if b, ok := books[symbol]; ok {
		return b
	}
	b := orderbook.NewOrderBook()
	books[symbol] = b
	return b
}

func toOrder(in IncomingOrder) *orderbook.Order {
	side := orderbook.Buy
	if in.Side == "SELL" {
		side = orderbook.Sell
	}
	return &orderbook.Order{
		ID:        in.ID,
		UserID:    in.UserID,
		Symbol:    in.Symbol,
		Side:      side,
		Price:     in.Price,
		Quantity:  in.Quantity,
		Status:    orderbook.Open,
		Timestamp: in.Timestamp,

		PreventSelfTrade: in.STP,
	}
}

// Islem id'si kaynak kaydin konumundan turetilir: engine cokup ayni kaydi
// yeniden islerse ayni id cikar ve processor ikinci kez yazmaz.
func tradeID(partition int32, offset int64, i int) string {
	return fmt.Sprintf("%d-%d-%d", partition, offset, i)
}

func toOutgoingTrades(trades []orderbook.Trade, partition int32, offset int64) []OutgoingTrade {
	out := make([]OutgoingTrade, len(trades))
	for i, tr := range trades {
		out[i] = OutgoingTrade{
			ID:          tradeID(partition, offset, i),
			BuyOrderID:  tr.BuyOrderID,
			SellOrderID: tr.SellOrderID,
			BuyerID:     tr.BuyerID,
			SellerID:    tr.SellerID,
			Symbol:      tr.Symbol,
			Price:       tr.Price,
			BuyLimit:    tr.BuyLimit,
			Quantity:    tr.Quantity,
			Timestamp:   tr.Timestamp,
		}
	}
	return out
}

func publishDepth(ctx context.Context, producer *kgo.Client, snap orderbook.OrderBookSnapshot) {
	b, err := json.Marshal(snap)
	if err != nil {
		log.Printf("depth marshal hatasi: %v", err)
		return
	}
	record := &kgo.Record{
		Topic: "depth",
		Key:   []byte(snap.Symbol),
		Value: b,
	}
	producer.Produce(ctx, record, func(_ *kgo.Record, err error) {
		if err != nil {
			log.Printf("depth kafka hatasi: %v", err)
		}
	})
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return fallback
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
