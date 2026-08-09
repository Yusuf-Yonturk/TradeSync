package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Herkese acik islem kaydi. Kimin kiminle islem yaptigi disari verilmez.
type PublicTrade struct {
	ID       int64     `json:"id"`
	Symbol   string    `json:"symbol"`
	Price    int64     `json:"price"`
	Quantity int64     `json:"quantity"`
	TradedAt time.Time `json:"traded_at"`
}

// Kullanicinin kendi islemi; karsi tarafin kimligi yine verilmez.
type MyTrade struct {
	PublicTrade
	Side    string `json:"side"`
	OrderID string `json:"order_id"`
}

func parseLimit(raw string, fallback, max int) int {
	if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= max {
		return n
	}
	return fallback
}

// GET /trades?symbol=BTC-USD&limit=50
func tradesHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol != "" && !symbolRe.MatchString(symbol) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz symbol"})
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"), 50, 500)

		rows, err := queryPublicTrades(r.Context(), db, symbol, limit)
		if err != nil {
			log.Printf("trades sorgu hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sorgu hatasi"})
			return
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

// GET /trades/mine?symbol=BTC-USD&limit=50 : giris yapan kullanicinin islemleri
func myTradesHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol != "" && !symbolRe.MatchString(symbol) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecersiz symbol"})
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"), 50, 500)

		rows, err := queryMyTrades(r.Context(), db, userIDFrom(r.Context()), symbol, limit)
		if err != nil {
			log.Printf("trades sorgu hatasi: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sorgu hatasi"})
			return
		}
		writeJSON(w, http.StatusOK, rows)
	}
}

func queryPublicTrades(ctx context.Context, db *pgxpool.Pool, symbol string, limit int) ([]PublicTrade, error) {
	rows, err := db.Query(ctx, `
		SELECT id, symbol, price, quantity, traded_at
		FROM trades
		WHERE $1 = '' OR symbol = $1
		ORDER BY traded_at DESC
		LIMIT $2
	`, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []PublicTrade{}
	for rows.Next() {
		var t PublicTrade
		if err := rows.Scan(&t.ID, &t.Symbol, &t.Price, &t.Quantity, &t.TradedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func queryMyTrades(ctx context.Context, db *pgxpool.Pool, userID, symbol string, limit int) ([]MyTrade, error) {
	rows, err := db.Query(ctx, `
		SELECT id, symbol, price, quantity, traded_at, buyer_id, buy_order_id, sell_order_id
		FROM trades
		WHERE (buyer_id = $1 OR seller_id = $1) AND ($2 = '' OR symbol = $2)
		ORDER BY traded_at DESC
		LIMIT $3
	`, userID, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []MyTrade{}
	for rows.Next() {
		var t MyTrade
		var buyerID, buyOrderID, sellOrderID string
		if err := rows.Scan(&t.ID, &t.Symbol, &t.Price, &t.Quantity, &t.TradedAt, &buyerID, &buyOrderID, &sellOrderID); err != nil {
			return nil, err
		}
		t.Side, t.OrderID = mySide(userID, buyerID, buyOrderID, sellOrderID)
		out = append(out, t)
	}
	return out, rows.Err()
}

func mySide(userID, buyerID, buyOrderID, sellOrderID string) (side, orderID string) {
	if buyerID == userID {
		return "BUY", buyOrderID
	}
	return "SELL", sellOrderID
}

// Engine'in trade kaydindan websocket'e gidecek kimliksiz hali uretir.
func publicTradeMessage(raw []byte) ([]byte, bool) {
	var in struct {
		Symbol    string    `json:"symbol"`
		Price     int64     `json:"price"`
		Quantity  int64     `json:"quantity"`
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(raw, &in); err != nil || in.Symbol == "" {
		return nil, false
	}
	out, err := json.Marshal(in)
	if err != nil {
		return nil, false
	}
	return out, true
}
