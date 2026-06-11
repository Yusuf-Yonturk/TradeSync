package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/twmb/franz-go/pkg/kgo"
	"tradesync/engine/orderbook"
)

func main() {
	brokers := []string{envOr("KAFKA_BROKERS", "localhost:9092")}

	consumer, err := newKafkaClient(brokers, "engine-group")
	if err != nil {
		log.Fatalf("consumer olusturulamadi: %v", err)
	}
	defer consumer.Close()

	producer, err := newKafkaProducer(brokers)
	if err != nil {
		log.Fatalf("producer olusturulamadi: %v", err)
	}
	defer producer.Close()

	// Her symbol icin ayri book
	books := make(map[string]*orderbook.OrderBook)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

		fetches.EachRecord(func(r *kgo.Record) {
			var incoming IncomingOrder
			if err := json.Unmarshal(r.Value, &incoming); err != nil {
				log.Printf("order parse hatasi: %v", err)
				return
			}

			book := getOrCreateBook(books, incoming.Symbol)
			order := toOrder(incoming)
			trades := book.AddOrder(order)

			// Her order eklendiğinde tahtanın güncel halini kafka'ya bas
			snapshot := book.Snapshot(incoming.Symbol, 15)
			publishDepth(ctx, producer, snapshot)

			if len(trades) == 0 {
				return
			}

			outgoing := toOutgoingTrades(trades)
			publishTrades(ctx, producer, outgoing)

			log.Printf("eslesti symbol=%s trades=%d", incoming.Symbol, len(trades))
		})
	}
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

