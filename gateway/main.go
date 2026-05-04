package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {

	g := http.NewServeMux()

	g.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "gateway",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	err := http.ListenAndServe(":8082", g)
	if err != nil {
		log.Fatalf("HTTP server error")
	}
}
