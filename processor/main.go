package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	topicTrades       = "trades"
	topicOrderUpdates = "order-updates"
)

// Engine'in yayinladigi emir durumu
type OrderUpdate struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Symbol    string `json:"symbol"`
	Side      string `json:"side"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
	FilledQty int64  `json:"filled_quantity"`
	Status    string `json:"status"`
}

type TradeEvent struct {
	ID          string    `json:"trade_id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Price       int64     `json:"price"`
	BuyLimit    int64     `json:"buy_limit"`
	Quantity    int64     `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

// Veritabani hatasinda kaydi atlamak yerine bu aralikla yeniden dener.
const retryDelay = 2 * time.Second

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
	if err := applySchema(context.Background(), db); err != nil {
		log.Fatalf("sema uygulanamadi: %v", err)
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
		kgo.ConsumeTopics(topicTrades, "depth", topicOrderUpdates),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		// Offset, kayit veritabanina yazildiktan sonra elle ilerletilir.
		kgo.DisableAutoCommit(),
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
			if ctx.Err() != nil {
				return
			}
			switch r.Topic {
			case "depth":
				saveDepth(ctx, rdb, r)
			case topicOrderUpdates:
				handleOrderUpdate(ctx, db, rdb, r.Value)
			default:
				handleTrade(ctx, db, rdb, r.Value)
			}
		})

		// Kapanirken yarida kalan kayit olabilir; offset ilerletilmez, acilista yeniden islenir.
		if ctx.Err() != nil {
			log.Println("processor durdu")
			return
		}
		if err := consumer.CommitUncommittedOffsets(ctx); err != nil {
			log.Printf("offset kayit hatasi: %v", err)
		}
	}
}

// Basarili olana ya da servis kapanana kadar dener. Atlanan bir islem
// veritabaninda kalici bir eksik olurdu.
func untilDone(ctx context.Context, what string, fn func() error) {
	for {
		err := fn()
		if err == nil || ctx.Err() != nil {
			return
		}
		log.Printf("%s hatasi, yeniden denenecek: %v", what, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(retryDelay):
		}
	}
}

func saveDepth(ctx context.Context, rdb *redis.Client, r *kgo.Record) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := rdb.Set(ctx, "orderbook:"+string(r.Key), r.Value, 0).Err(); err != nil {
		log.Printf("redis kayit hatasi: %v", err)
	}
}

func handleTrade(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, raw []byte) {
	var trade TradeEvent
	if err := json.Unmarshal(raw, &trade); err != nil {
		log.Printf("trade parse hatasi: %v", err)
		return
	}

	inserted := false
	untilDone(ctx, "trade kayit", func() error {
		var err error
		inserted, err = settleTrade(ctx, db, trade)
		return err
	})
	if !inserted {
		return
	}

	updateKline(ctx, rdb, trade)
	log.Printf("trade kaydedildi symbol=%s price=%d qty=%d", trade.Symbol, trade.Price, trade.Quantity)
}

func handleOrderUpdate(ctx context.Context, db *pgxpool.Pool, rdb *redis.Client, raw []byte) {
	var u OrderUpdate
	if err := json.Unmarshal(raw, &u); err != nil {
		log.Printf("order update parse hatasi: %v", err)
		return
	}
	if u.ID == "" || u.UserID == "" {
		return
	}

	if amount, ok := releaseAmount(u); ok {
		untilDone(ctx, "bakiye serbest birakma", func() error {
			return releaseOrder(ctx, db, u.ID, amount)
		})
	}
	applyOrderUpdate(ctx, rdb, u, raw)
}

// Acik emirler kullanici basina bir redis hash'inde tutulur: orders:<user_id> -> id -> emir.
// Dolan ya da iptal edilen emir listeden silinir.
func applyOrderUpdate(ctx context.Context, rdb *redis.Client, u OrderUpdate, raw []byte) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := "orders:" + u.UserID
	var err error
	if isOpenStatus(u.Status) {
		err = rdb.HSet(ctxTimeout, key, u.ID, raw).Err()
	} else {
		err = rdb.HDel(ctxTimeout, key, u.ID).Err()
	}
	if err != nil {
		log.Printf("acik emir yazma hatasi: %v", err)
	}
}

func isOpenStatus(status string) bool {
	return status == "OPEN" || status == "PARTIALLY_FILLED"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
