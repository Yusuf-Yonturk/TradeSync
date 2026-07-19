package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
)

const maxOpenOrders = 200

var orderIDRe = regexp.MustCompile(`^ord-[0-9a-f]{16}$`)

// Processor'in redis'e yazdigi acik emir kaydi
type OpenOrder struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	FilledQty int64     `json:"filled_quantity"`
	Remaining int64     `json:"remaining"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Engine'e giden iptal komutu; siralama bozulmasin diye emirlerle ayni topic'e yazilir.
type CancelMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
}

func openOrdersKey(userID string) string {
	return "orders:" + userID
}

// Ham redis kayitlarini cozer, symbol'e gore suzer, yeniden eskiye siralar.
func decodeOpenOrders(raw map[string]string, symbol string) []OpenOrder {
	orders := make([]OpenOrder, 0, len(raw))
	for _, val := range raw {
		var o OpenOrder
		if err := json.Unmarshal([]byte(val), &o); err != nil {
			continue
		}
		if symbol != "" && o.Symbol != symbol {
			continue
		}
		o.Remaining = o.Quantity - o.FilledQty
		orders = append(orders, o)
	}
	sort.Slice(orders, func(i, j int) bool {
		if !orders[i].Timestamp.Equal(orders[j].Timestamp) {
			return orders[i].Timestamp.After(orders[j].Timestamp)
		}
		return orders[i].ID > orders[j].ID
	})
	if len(orders) > maxOpenOrders {
		orders = orders[:maxOpenOrders]
	}
	return orders
}

// GET /orders?symbol=BTC-USD : giris yapan kullanicinin bekleyen emirleri
func listOrdersHandler(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol != "" && !symbolRe.MatchString(symbol) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz symbol"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		raw, err := rdb.HGetAll(ctx, openOrdersKey(userIDFrom(r.Context()))).Result()
		if err != nil {
			log.Printf("acik emir okuma hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "emirler okunamadi"})
			return
		}
		writeJSON(w, http.StatusOK, decodeOpenOrders(raw, symbol))
	}
}

// DELETE /orders/{id} : iptal istegini engine'e iletir. Iptal asenkrondur;
// emir bu arada dolarsa iptal etkisiz kalir.
func cancelOrderHandler(rdb *redis.Client, producer *kgo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !orderIDRe.MatchString(id) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz emir id"})
			return
		}
		userID := userIDFrom(r.Context())

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Emir kullanicinin kendi listesinde yoksa ya baskasinindir ya da artik acik degildir.
		val, err := rdb.HGet(ctx, openOrdersKey(userID), id).Result()
		if errors.Is(err, redis.Nil) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "acik emir bulunamadi"})
			return
		}
		if err != nil {
			log.Printf("acik emir okuma hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "emir okunamadi"})
			return
		}
		var order OpenOrder
		if err := json.Unmarshal([]byte(val), &order); err != nil || !symbolRe.MatchString(order.Symbol) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "emir kaydi bozuk"})
			return
		}

		data, err := json.Marshal(CancelMessage{
			Type:      "CANCEL",
			ID:        id,
			UserID:    userID,
			Symbol:    order.Symbol,
			Timestamp: time.Now().UTC(),
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal hatasi"})
			return
		}

		record := &kgo.Record{Topic: topicOrders, Key: []byte(order.Symbol), Value: data}
		if err := producer.ProduceSync(r.Context(), record).FirstErr(); err != nil {
			log.Printf("kafka yazma hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "kafka yazma hatasi"})
			return
		}

		log.Printf("iptal istendi id=%s user=%s", id, userID)
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "cancel_requested", "id": id})
	}
}
