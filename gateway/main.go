package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
	"tradesync/shared/money"
)

const topicOrders = "orders"

const (
	// /klines varsayilan olarak son bir gunun, en fazla bir haftanin mumlarini doner
	defaultKlines = 1440
	maxKlines     = 10080
	// Istemci baglantiyi kesse de kafka yazimi yarida birakilmaz; aksi halde
	// emir engine'e ulasmisken ayrilan bakiye geri verilebilirdi.
	produceTimeout = 10 * time.Second
)

const (
	maxBodyBytes = 1 << 16
	// Degerler 10^4 olcekli tutuluyor; ust sinir defterdeki toplamlarin tasmasini onler.
	maxPrice    = int64(1e13)
	maxQuantity = int64(1e13)
)

// BTC-USD gibi; serbest metin kabul edilirse her istek yeni bir defter ve redis anahtari acar.
var symbolRe = regexp.MustCompile(`^[A-Z0-9]{2,10}-[A-Z0-9]{2,10}$`)

type OrderRequest struct {
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
}

type OrderMessage struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
	// engine kullanicinin kendi emriyle eslesmesini engeller
	STP bool `json:"stp"`
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
	if !symbolRe.MatchString(r.Symbol) {
		return "gecersiz symbol"
	}
	if r.Side != "BUY" && r.Side != "SELL" {
		return "side BUY veya SELL olmali"
	}
	if r.Price <= 0 {
		return "price sifirdan buyuk olmali"
	}
	if r.Price > maxPrice {
		return "price cok buyuk"
	}
	if r.Quantity <= 0 {
		return "quantity sifirdan buyuk olmali"
	}
	if r.Quantity > maxQuantity {
		return "quantity cok buyuk"
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Govde boyutunu sinirlayarak JSON okur.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	return json.NewDecoder(r.Body).Decode(v)
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

	signer := newTokenSigner(os.Getenv("AUTH_SECRET"))

	funds, err := parseDemoFunds(envOr("DEMO_FUNDS", defaultDemoFunds))
	if err != nil {
		log.Fatalf("DEMO_FUNDS: %v", err)
	}
	exempt, err := provisionMarketMakers(context.Background(), db, envOr("MM_USERNAME", "marketmaker"), os.Getenv("MM_PASSWORD"))
	if err != nil {
		log.Fatalf("market maker hesaplari hazirlanamadi: %v", err)
	}
	if len(exempt) == 0 {
		log.Println("MM_PASSWORD tanimli degil, bakiye kontrolunden muaf hesap yok")
	}
	books := &ledger{db: db, funds: funds, exempt: exempt}

	auth := &authLimits{
		// ayni adresten 5 dakikada 10 hatali giris, saatte 10 kayit
		loginFailures: newRateLimiter(10, 5*time.Minute),
		registrations: newRateLimiter(10, time.Hour),
		ipHeader:      os.Getenv("CLIENT_IP_HEADER"),
	}

	hub := NewWSHub()
	startKafkaTradeConsumer(brokers, hub)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /ws", hub.HandleWS)

	mux.HandleFunc("GET /depth", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if !symbolRe.MatchString(symbol) {
			http.Error(w, "symbol gerekli", http.StatusBadRequest)
			return
		}

		ctxTimeout, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		val, err := rdb.Get(ctxTimeout, "orderbook:"+symbol).Result()
		if err != nil {
			if err == redis.Nil {
				writeJSON(w, http.StatusOK, map[string]interface{}{"bids": []interface{}{}, "asks": []interface{}{}})
				return
			}
			log.Printf("depth okuma hatasi: %v", err)
			http.Error(w, "redis okuma hatasi", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(val))
	})

	mux.HandleFunc("GET /klines", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if !symbolRe.MatchString(symbol) {
			http.Error(w, "symbol param gerekli", http.StatusBadRequest)
			return
		}

		ctxTimeout, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		res, err := rdb.HGetAll(ctxTimeout, "klines:"+symbol).Result()
		if err != nil {
			log.Printf("klines okuma hatasi: %v", err)
			http.Error(w, "redis okuma hatasi", http.StatusInternalServerError)
			return
		}

		// bos sonucta null degil [] donmeli, istemci dizi bekliyor
		klines := make([]Kline, 0, len(res))
		for _, val := range res {
			var k Kline
			if err := json.Unmarshal([]byte(val), &k); err != nil {
				continue
			}
			klines = append(klines, k)
		}

		sort.Slice(klines, func(i, j int) bool {
			return klines[i].Time < klines[j].Time
		})
		if limit := parseLimit(r.URL.Query().Get("limit"), defaultKlines, maxKlines); len(klines) > limit {
			klines = klines[len(klines)-limit:]
		}

		writeJSON(w, http.StatusOK, klines)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "gateway",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("POST /auth/register", registerHandler(db, signer, books, auth))
	mux.HandleFunc("POST /auth/login", loginHandler(db, signer, books, auth))

	mux.HandleFunc("GET /balances", signer.requireAuth(books.balancesHandler()))
	mux.HandleFunc("GET /trades", tradesHandler(db))
	mux.HandleFunc("GET /trades/mine", signer.requireAuth(myTradesHandler(db)))

	mux.HandleFunc("GET /orders", signer.requireAuth(listOrdersHandler(rdb)))
	mux.HandleFunc("DELETE /orders/{id}", signer.requireAuth(cancelOrderHandler(rdb, producer)))

	// user_id govdeden degil token'dan alinir; baskasi adina emir verilemez.
	mux.HandleFunc("POST /orders", signer.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		var req OrderRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz json"})
			return
		}

		if msg := validate(req); msg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}

		msg := OrderMessage{
			ID:        newOrderID(),
			UserID:    userIDFrom(r.Context()),
			Symbol:    req.Symbol,
			Side:      req.Side,
			Price:     req.Price,
			Quantity:  req.Quantity,
			Timestamp: time.Now().UTC(),
			STP:       true,
		}

		data, err := json.Marshal(msg)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal hatasi"})
			return
		}

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), produceTimeout)
		defer cancel()

		// Emir engine'e gitmeden once karsiligi bakiyeden ayrilir.
		asset, amount, ok := money.Reserve(msg.Symbol, msg.Side, msg.Price, msg.Quantity)
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz emir"})
			return
		}
		if err := books.reserve(ctx, msg.ID, msg.UserID, asset, amount); err != nil {
			if errors.Is(err, errInsufficient) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "yetersiz " + asset + " bakiyesi"})
				return
			}
			log.Printf("bakiye ayirma hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "bakiye ayrilamadi"})
			return
		}

		record := &kgo.Record{
			Topic: topicOrders,
			Key:   []byte(msg.Symbol),
			Value: data,
		}

		if err := producer.ProduceSync(ctx, record).FirstErr(); err != nil {
			log.Printf("kafka yazma hatasi: %v", err)
			if uerr := books.unreserve(ctx, msg.ID); uerr != nil {
				log.Printf("bakiye geri verilemedi id=%s: %v", msg.ID, uerr)
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "kafka yazma hatasi"})
			return
		}

		log.Printf("order alindi id=%s user=%s symbol=%s side=%s", msg.ID, msg.UserID, msg.Symbol, msg.Side)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted", "id": msg.ID})
	}))

	addr := ":8072"
	log.Printf("gateway basliyor %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("sunucu hatasi: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
