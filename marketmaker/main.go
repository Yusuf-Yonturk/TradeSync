package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type OrderRequest struct {
	UserID   string `json:"user_id"`
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
}

func main() {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://gateway:8072"
	}

	time.Sleep(10 * time.Second) // gateway'in ayaklanmasını bekle
	fmt.Println("Market Maker başladı! Canlı veriler akıtılıyor...")

	btcPrice := 65000.0
	ethPrice := 3500.0

	// Sonsuz döngüde rastgele emirler oluştur
	for {
		// BTC-USD ve ETH-USD için emirler atalım
		simulateMarket(gatewayURL, "BTC-USD", &btcPrice, 50.0) // 50$ volatilite
		simulateMarket(gatewayURL, "ETH-USD", &ethPrice, 5.0)  // 5$ volatilite

		// 300 ile 800 ms arası rastgele bekle (akıcı hissi versin)
		time.Sleep(time.Duration(rand.Intn(500)+300) * time.Millisecond)
	}
}

func simulateMarket(gatewayURL string, symbol string, currentPrice *float64, volatility float64) {
	// Fiyatı rastgele ufak bir miktar oynat
	change := (rand.Float64() * volatility * 2) - volatility
	*currentPrice += change

	side := "BUY"
	if rand.Intn(2) == 0 {
		side = "SELL"
	}

	// Yeni fiyata çok yakın bir limit order oluştur
	// Fiyatın ara sıra spread'i geçmesini sağla (match olması için)
	orderPrice := *currentPrice
	if side == "BUY" {
		orderPrice += (rand.Float64() * volatility * 1.5) - volatility
	} else {
		orderPrice += (rand.Float64() * volatility * 1.5) - volatility
	}

	qty := rand.Float64()*2 + 0.1 // 0.1 ile 2.1 arası miktar

	order := OrderRequest{
		UserID:   "market-maker-bot",
		Symbol:   symbol,
		Side:     side,
		Price:    int64(orderPrice * 10000),
		Quantity: int64(qty * 10000),
	}

	payload, _ := json.Marshal(order)
	resp, err := http.Post(gatewayURL+"/orders", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println("Marketmaker request error:", err)
	} else {
		if resp.StatusCode >= 400 {
			fmt.Println("Marketmaker response error:", resp.Status)
		}
		resp.Body.Close()
	}
}
