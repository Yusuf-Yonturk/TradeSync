package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
)

const topicTrades = "trades"

type TradeEvent struct {
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

type Kline struct {
	Time   int64 `json:"time"`
	Open   int64 `json:"open"`
	High   int64 `json:"high"`
	Low    int64 `json:"low"`
	Close  int64 `json:"close"`
	Volume int64 `json:"volume"`
}

func main() {
	brokers := strings.Split(envOr("KAFKA_BROKERS", "localhost:9092"), ",")
	dsn := envOr("POSTGRES_DSN", "postgres://ts_user:ts_pass@localhost:5432/tradesync")

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("postgres baglanamadi: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("postgres ping hatasi: %v", err)
	}
	log.Println("postgres baglandi")

	rdb := redis.NewClient(&redis.Options{
		Addr: envOr("REDIS_ADDR", "localhost:6380"),
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis baglanamadi: %v", err)
	}
	log.Println("redis baglandi")

	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("processor-group"),
		kgo.ConsumeTopics(topicTrades, "depth"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		log.Fatalf("kafka consumer olusturulamadi: %v", err)
	}
	defer consumer.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("processor basliyor")

	for {
		fetches := consumer.PollFetches(ctx)

		if ctx.Err() != nil {
			log.Println("processor durdu")
			return
		}

		fetches.EachError(func(t string, p int32, err error) {
			log.Printf("fetch hatasi topic=%s partition=%d: %v", t, p, err)
		})

		fetches.EachRecord(func(r *kgo.Record) {
			if r.Topic == "depth" {
				symbol := string(r.Key)
				ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
				defer cancel()
				if err := rdb.Set(ctxTimeout, "orderbook:"+symbol, r.Value, 0).Err(); err != nil {
					log.Printf("redis kayit hatasi: %v", err)
				}
				return
			}

			var trade TradeEvent
			if err := json.Unmarshal(r.Value, &trade); err != nil {
				log.Printf("trade parse hatasi: %v", err)
				return
			}

			if err := saveTrade(ctx, db, trade); err != nil {
				log.Printf("trade kayit hatasi: %v", err)
				return
			}

			updateKline(ctx, rdb, trade)

			log.Printf("trade kaydedildi symbol=%s price=%d qty=%d", trade.Symbol, trade.Price, trade.Quantity)
		})
	}
}

func saveTrade(ctx context.Context, db *pgxpool.Pool, t TradeEvent) error {
	_, err := db.Exec(ctx, `
		INSERT INTO trades (buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.BuyOrderID, t.SellOrderID, t.BuyerID, t.SellerID, t.Symbol, t.Price, t.Quantity, t.Timestamp)
	return err
}

func updateKline(ctx context.Context, rdb *redis.Client, t TradeEvent) {
	minute := t.Timestamp.Truncate(time.Minute).Unix()
	key := "klines:" + t.Symbol
	field := strconv.FormatInt(minute, 10)

	val, err := rdb.HGet(ctx, key, field).Result()
	var k Kline
	if err == redis.Nil {
		k = Kline{
			Time:   minute,
			Open:   t.Price,
			High:   t.Price,
			Low:    t.Price,
			Close:  t.Price,
			Volume: t.Quantity,
		}
	} else if err == nil {
		json.Unmarshal([]byte(val), &k)
		if t.Price > k.High {
			k.High = t.Price
		}
		if t.Price < k.Low {
			k.Low = t.Price
		}
		k.Close = t.Price
		k.Volume += t.Quantity
	} else {
		log.Printf("kline okuma hatasi: %v", err)
		return
	}

	b, _ := json.Marshal(k)
	rdb.HSet(ctx, key, field, b)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
