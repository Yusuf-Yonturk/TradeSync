package processor

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	p := http.NewServeMux()

	p.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "processor",
			"status":  "healthy",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})

		err := http.ListenAndServe(":8073", p)
		if err != nil {
			log.Fatalf("HTTP Server Error")
		}
	})
}
