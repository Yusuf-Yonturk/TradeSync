package main

import (
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
	log.Printf("engine basliyor")
	err := http.ListenAndServe(":8081", eR)
	if err != nil {
		log.Fatalf("HTTP server error")
	}
}
