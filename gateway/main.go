package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
)

const topicOrders = "orders"

type OrderRequest struct {
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
	UserID   string `json:"user_id"` // frontend'den gelir, auth'dan sonra set edilir
}

type OrderMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
}

type TradeRow struct {
	ID          int64     `json:"id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	TradedAt    time.Time `json:"traded_at"`
}

type Kline struct {
	Time   int64 `json:"time"`
	Open   int64 `json:"open"`
	High   int64 `json:"high"`
	Low    int64 `json:"low"`
	Close  int64 `json:"close"`
	Volume int64 `json:"volume"`
}

func validate(r OrderRequest) string {
	if strings.TrimSpace(r.UserID) == "" {
		return "user_id bos olamaz"
	}
	if strings.TrimSpace(r.Symbol) == "" {
		return "symbol bos olamaz"
	}
	if r.Side != "BUY" && r.Side != "SELL" {
		return "side BUY veya SELL olmali"
	}
	if r.Price <= 0 {
		return "price sifirdan buyuk olmali"
	}
	if r.Quantity <= 0 {
		return "quantity sifirdan buyuk olmali"
	}
	return ""
}

func newOrderID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "ord-" + hex.EncodeToString(b)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	brokers := strings.Split(envOr("KAFKA_BROKERS", "localhost:9092"), ",")
	dsn := envOr("POSTGRES_DSN", "postgres://ts_user:ts_pass@localhost:5432/tradesync")

	producer, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		log.Fatalf("kafka producer olusturulamadi: %v", err)
	}
	defer producer.Close()

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

	hub := NewWSHub()
	startKafkaTradeConsumer(brokers, hub)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ws", hub.HandleWS)

	mux.HandleFunc("GET /depth", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol gerekli", http.StatusBadRequest)
			return
		}

		ctxTimeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		val, err := rdb.Get(ctxTimeout, "orderbook:"+symbol).Result()
		if err != nil {
			if err == redis.Nil {
				writeJSON(w, http.StatusOK, map[string]interface{}{"bids": []interface{}{}, "asks": []interface{}{}})
				return
			}
			http.Error(w, "redis okuma hatasi", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(val))
	})

	mux.HandleFunc("GET /klines", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol param gerekli", http.StatusBadRequest)
			return
		}

		res, err := rdb.HGetAll(context.Background(), "klines:"+symbol).Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var klines []Kline
		for _, val := range res {
			var k Kline
			json.Unmarshal([]byte(val), &k)
			klines = append(klines, k)
		}

		sort.Slice(klines, func(i, j int) bool {
			return klines[i].Time < klines[j].Time
		})

		writeJSON(w, http.StatusOK, klines)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "gateway",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /auth/register", registerHandler(db))
	mux.HandleFunc("POST /auth/login", loginHandler(db))

	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		if msg := validate(req); msg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}

		msg := OrderMessage{
			ID:        newOrderID(),
			UserID:    req.UserID,
			Symbol:    req.Symbol,
			Side:      req.Side,
			Price:     req.Price,
			Quantity:  req.Quantity,
			Timestamp: time.Now().UTC(),
		}

		data, err := json.Marshal(msg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal hatasi"})
			return
		}

		ctx := r.Context()
		record := &kgo.Record{
			Topic: topicOrders,
			Key:   []byte(msg.Symbol),
			Value: data,
		}

		var produceErr error
		producer.Produce(ctx, record, func(_ *kgo.Record, err error) {
			produceErr = err
		})
		if err := producer.Flush(ctx); err != nil || produceErr != nil {
			log.Printf("kafka yazma hatasi: flush=%v produce=%v", err, produceErr)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "kafka yazma hatasi"})
			return
		}

		log.Printf("order alindi id=%s user=%s symbol=%s side=%s", msg.ID, msg.UserID, msg.Symbol, msg.Side)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted", "id": msg.ID})
	})

	// GET /trades?symbol=BTC-USD&limit=50&user_id=...
	mux.HandleFunc("GET /trades", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		limitStr := r.URL.Query().Get("limit")
		userID := r.URL.Query().Get("user_id")

		limit := 50
		if limitStr != "" {
			if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}

		var rows []TradeRow
		var err error

		if userID != "" {
			rows, err = queryTradesByUser(r.Context(), db, userID, symbol, limit)
		} else if symbol != "" {
			rows, err = queryTradesBySymbol(r.Context(), db, symbol, limit)
		} else {
			rows, err = queryAllTrades(r.Context(), db, limit)
		}

		if err != nil {
			log.Printf("trades sorgu hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sorgu hatasi"})
			return
		}

		writeJSON(w, http.StatusOK, rows)
	})

	addr := ":8072"
	log.Printf("gateway basliyor %s", addr)
	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatalf("sunucu hatasi: %v", err)
	}
}

func queryTradesBySymbol(ctx context.Context, db *pgxpool.Pool, symbol string, limit int) ([]TradeRow, error) {
	rows, err := db.Query(ctx, `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at
		FROM trades
		WHERE symbol = $1
		ORDER BY traded_at DESC
		LIMIT $2
	`, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTrades(rows)
}

func queryTradesByUser(ctx context.Context, db *pgxpool.Pool, userID string, symbol string, limit int) ([]TradeRow, error) {
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var err error
	
	if symbol != "" {
		rows, err = db.Query(ctx, `
			SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at
			FROM trades
			WHERE (buyer_id = $1 OR seller_id = $1) AND symbol = $2
			ORDER BY traded_at DESC
			LIMIT $3
		`, userID, symbol, limit)
	} else {
		rows, err = db.Query(ctx, `
			SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at
			FROM trades
			WHERE buyer_id = $1 OR seller_id = $1
			ORDER BY traded_at DESC
			LIMIT $2
		`, userID, limit)
	}
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTrades(rows)
}

func queryAllTrades(ctx context.Context, db *pgxpool.Pool, limit int) ([]TradeRow, error) {
	rows, err := db.Query(ctx, `
		SELECT id, buy_order_id, sell_order_id, buyer_id, seller_id, symbol, price, quantity, traded_at
		FROM trades
		ORDER BY traded_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTrades(rows)
}

func scanTrades(rows interface {
	Next() bool
	Scan(...any) error
}) ([]TradeRow, error) {
	var result []TradeRow
	for rows.Next() {
		var t TradeRow
		if err := rows.Scan(&t.ID, &t.BuyOrderID, &t.SellOrderID, &t.BuyerID, &t.SellerID, &t.Symbol, &t.Price, &t.Quantity, &t.TradedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	if result == nil {
		result = []TradeRow{}
	}
	return result, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
