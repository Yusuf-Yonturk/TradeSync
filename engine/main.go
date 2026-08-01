package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"tradesync/engine/orderbook"
)

const offsetGroup = "engine-group"

func main() {
	brokers := strings.Split(envOr("KAFKA_BROKERS", "localhost:9092"), ",")

	consumer, err := newKafkaClient(brokers)
	if err != nil {
		log.Fatalf("consumer olusturulamadi: %v", err)
	}
	defer consumer.Close()

	producer, err := newKafkaProducer(brokers)
	if err != nil {
		log.Fatalf("producer olusturulamadi: %v", err)
	}
	defer producer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Defter bellekte tutuldugu icin her acilista orders topic'i bastan okunur.
	// Daha once islenmis kayitlar (offset < committed) sadece deftere uygulanir,
	// sonuclari tekrar yayinlanmaz.
	offsets := kadm.NewClient(consumer)
	committed, err := fetchCommitted(ctx, offsets)
	if err != nil {
		log.Fatalf("islenmis offsetler okunamadi: %v", err)
	}
	if len(committed) > 0 {
		log.Printf("defter yeniden kuruluyor, islenmis offsetler=%v", committed)
	}

	// Her symbol icin ayri book
	books := make(map[string]*orderbook.OrderBook)
	replayed := 0
	rebuilding := len(committed) > 0

	log.Println("engine basliyor")

	for {
		fetches := consumer.PollFetches(ctx)

		if ctx.Err() != nil {
			log.Println("engine durdu")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			log.Printf("fetch hatasi topic=%s partition=%d: %v", t, p, err)
		})

		// bu turda canli islenen kayitlarin bir sonraki offsetleri
		processed := make(map[int32]int64)

		fetches.EachRecord(func(r *kgo.Record) {
			live := isLive(committed, r.Partition, r.Offset)
			if live {
				processed[r.Partition] = r.Offset + 1
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
			snapshot := book.Snapshot(incoming.Symbol, 15)
			publishDepth(ctx, producer, snapshot)
			publishOrderUpdates(ctx, producer, res.updates)

			if len(trades) == 0 {
				return
			}

			outgoing := toOutgoingTrades(trades)
			publishTrades(ctx, producer, outgoing)

			log.Printf("eslesti symbol=%s trades=%d", incoming.Symbol, len(trades))
		})

		if rebuilding && len(processed) > 0 {
			rebuilding = false
			log.Printf("defter yeniden kuruldu, %d eski kayit uygulandi", replayed)
		}

		if len(processed) == 0 {
			continue
		}
		// Sonuclar kafka'ya ulasmadan offset ilerletilmez; aksi halde cokme sonrasi kayip olur.
		if err := producer.Flush(ctx); err != nil {
			log.Printf("flush hatasi, offset ilerletilmedi: %v", err)
			continue
		}
		if err := commitProcessed(ctx, offsets, processed); err != nil {
			log.Printf("offset kayit hatasi: %v", err)
			continue
		}
		for p, next := range processed {
			committed[p] = next
		}
	}
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
	}
}

func toOutgoingTrades(trades []orderbook.Trade) []OutgoingTrade {
	out := make([]OutgoingTrade, len(trades))
	for i, tr := range trades {
		out[i] = OutgoingTrade{
			BuyOrderID:  tr.BuyOrderID,
			SellOrderID: tr.SellOrderID,
			BuyerID:     tr.BuyerID,
			SellerID:    tr.SellerID,
			Symbol:      tr.Symbol,
			Price:       tr.Price,
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
