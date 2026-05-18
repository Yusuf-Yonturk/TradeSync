package main

import (
	"container/heap"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"tradesync/engine/orderbook"
)

func main() {
	eR := http.NewServeMux()

	eR.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "engine",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	eR.HandleFunc("GET /test-order", func(w http.ResponseWriter, r *http.Request) {
		order := orderbook.Order{
			ID:        "ord-1",
			UserID:    "user-1",
			Side:      orderbook.Buy,
			Price:     10500,
			Quantity:  10,
			FilledQty: 3,
			Status:    orderbook.Open,
			Timestamp: time.Now(),
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":        order.ID,
			"remaining": order.RemainingQty(),
			"is_filled": order.IsFilled(),
		})
	})

	eR.HandleFunc("GET /test-heap", func(w http.ResponseWriter, r *http.Request) {
		bh := &orderbook.BuyHeap{}
		heap.Init(bh)

		now := time.Now()

		heap.Push(bh, &orderbook.Order{
			ID:        "a1",
			Side:      orderbook.Buy,
			Price:     10050,
			Quantity:  10,
			Timestamp: now,
			Status:    orderbook.Open,
		})
		heap.Push(bh, &orderbook.Order{
			ID:        "a2",
			Side:      orderbook.Buy,
			Price:     10100,
			Quantity:  10,
			Timestamp: now.Add(1 * time.Second),
			Status:    orderbook.Open,
		})
		top := (*bh)[0]
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"top_order_id": top.ID,
			"top_price":    top.Price,
		})
	})

	eR.HandleFunc("GET /test-book", func(w http.ResponseWriter, r *http.Request) {
		ob := orderbook.NewOrderBook()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"buy_count":  ob.Buys.Len(),
			"sell_count": ob.Sells.Len(),
		})
	})
	log.Printf("engine basliyor")
	err := http.ListenAndServe(":8071", eR)
	if err != nil {
		log.Fatalf("HTTP server error")
	}
}
