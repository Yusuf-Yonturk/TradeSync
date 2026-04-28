package engine

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	eR := http.NewServeMux()

	eR.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "engine",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	err := http.ListenAndServe(":8081", eR)
	if err != nil {
		log.Fatalf("HTTP server error")
	}
}
