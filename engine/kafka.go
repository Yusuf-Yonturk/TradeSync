package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	topicOrders = "orders"
	topicTrades = "trades"
)

// Kafka'dan gelen ham emir
type IncomingOrder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

// Kafka'ya gidecek trade
type OutgoingTrade struct {
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

func newKafkaClient(brokers []string, groupID string) (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topicOrders),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
}

func newKafkaProducer(brokers []string) (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(brokers...),
	)
}

func publishTrades(ctx context.Context, producer *kgo.Client, trades []OutgoingTrade) {
	for _, tr := range trades {
		data, err := json.Marshal(tr)
		if err != nil {
			log.Printf("trade marshal hatasi: %v", err)
			continue
		}
		record := &kgo.Record{
			Topic: topicTrades,
			Key:   []byte(tr.Symbol),
			Value: data,
		}
		producer.Produce(ctx, record, func(r *kgo.Record, err error) {
			if err != nil {
				log.Printf("trade publish hatasi: %v", err)
			}
		})
	}
	// Tampon bos olana kadar bekle
	if err := producer.Flush(ctx); err != nil {
		log.Printf("flush hatasi: %v", err)
	}
}
