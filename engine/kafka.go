package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	topicOrders       = "orders"
	topicTrades       = "trades"
	topicOrderUpdates = "order-updates"
)

// Kafka'dan gelen ham emir
type IncomingOrder struct {
	// bos ise yeni emir, "CANCEL" ise iptal
	Type      string    `json:"type,omitempty"`
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

// Grup uyeligi olmadan topic'i her zaman bastan okur. Nereye kadar islendigi
// offsetGroup altinda elle saklanir (fetchCommitted / commitProcessed).
func newKafkaClient(brokers []string) (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics(topicOrders),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
}

// Partition basina, islenmis son kaydin bir sonraki offsetini doner.
func fetchCommitted(ctx context.Context, adm *kadm.Client) (map[int32]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := adm.FetchOffsets(ctx, offsetGroup)
	if err != nil {
		return nil, err
	}
	committed := make(map[int32]int64)
	for partition, o := range resp[topicOrders] {
		if o.Err != nil {
			return nil, o.Err
		}
		if o.At > 0 {
			committed[partition] = o.At
		}
	}
	return committed, nil
}

func commitProcessed(ctx context.Context, adm *kadm.Client, processed map[int32]int64) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var offsets kadm.Offsets
	for partition, next := range processed {
		offsets.AddOffset(topicOrders, partition, next, -1)
	}
	resp, err := adm.CommitOffsets(ctx, offsetGroup, offsets)
	if err != nil {
		return err
	}
	return resp.Error()
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

func publishOrderUpdates(ctx context.Context, producer *kgo.Client, updates []OrderUpdate) {
	for _, u := range updates {
		data, err := json.Marshal(u)
		if err != nil {
			log.Printf("order update marshal hatasi: %v", err)
			continue
		}
		record := &kgo.Record{
			Topic: topicOrderUpdates,
			Key:   []byte(u.Symbol),
			Value: data,
		}
		producer.Produce(ctx, record, func(_ *kgo.Record, err error) {
			if err != nil {
				log.Printf("order update publish hatasi: %v", err)
			}
		})
	}
}
