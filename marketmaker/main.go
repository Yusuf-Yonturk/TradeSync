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
	Symbol   string `json:"symbol"`
	Side     string `json:"side"`
	Price    int64  `json:"price"`
	Quantity int64  `json:"quantity"`
}

var client = &http.Client{Timeout: 5 * time.Second}

// Gateway artik token istiyor; bot kendi hesabiyla giris yapar.
var token string

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func postJSON(url string, body any, bearer string) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return client.Do(req)
}

// Hesap yoksa olusturur, sonra giris yapip token doner.
func login(gatewayURL, username, password string) (string, error) {
	creds := map[string]string{"name": "Market Maker", "username": username, "password": password}

	resp, err := postJSON(gatewayURL+"/auth/register", creds, "")
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return "", fmt.Errorf("register: %s", resp.Status)
	}

	resp, err = postJSON(gatewayURL+"/auth/login", creds, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login: %s", resp.Status)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("login: token bos")
	}
	return out.Token, nil
}

func mustLogin(gatewayURL, username, password string) string {
	for {
		t, err := login(gatewayURL, username, password)
		if err == nil {
			return t
		}
		fmt.Println("Marketmaker giris yapamadi, tekrar denenecek:", err)
		time.Sleep(3 * time.Second)
	}
}

func main() {
	gatewayURL := envOr("GATEWAY_URL", "http://gateway:8072")
	username := envOr("MM_USERNAME", "marketmaker")
	password := envOr("MM_PASSWORD", "marketmaker-dev")

	// gateway ayaga kalkana kadar dener
	token = mustLogin(gatewayURL, username, password)
	fmt.Println("Market Maker başladı! Canlı veriler akıtılıyor...")

	btcPrice := 65000.0
	ethPrice := 3500.0

	// Sonsuz döngüde rastgele emirler oluştur
	for {
		// BTC-USD ve ETH-USD için emirler atalım
		expired := simulateMarket(gatewayURL, "BTC-USD", &btcPrice, 50.0)          // 50$ volatilite
		expired = simulateMarket(gatewayURL, "ETH-USD", &ethPrice, 5.0) || expired // 5$ volatilite
		if expired {
			token = mustLogin(gatewayURL, username, password)
		}

		// 300 ile 800 ms arası rastgele bekle (akıcı hissi versin)
		time.Sleep(time.Duration(rand.Intn(500)+300) * time.Millisecond)
	}
}

// Emir gonderir; token reddedildiyse true doner.
func simulateMarket(gatewayURL string, symbol string, currentPrice *float64, volatility float64) bool {
	// Fiyatı rastgele ufak bir miktar oynat
	change := (rand.Float64() * volatility * 2) - volatility
	*currentPrice += change
	// rastgele yuruyus sifirin altina inmesin
	if *currentPrice < volatility*4 {
		*currentPrice = volatility * 4
	}

	side := "BUY"
	if rand.Intn(2) == 0 {
		side = "SELL"
	}

	// Yeni fiyata çok yakın bir limit order oluştur
	// Fiyatın ara sıra spread'i geçmesini sağla (match olması için)
	orderPrice := *currentPrice + (rand.Float64() * volatility * 1.5) - volatility

	qty := rand.Float64()*2 + 0.1 // 0.1 ile 2.1 arası miktar

	order := OrderRequest{
		Symbol:   symbol,
		Side:     side,
		Price:    int64(orderPrice * 10000),
		Quantity: int64(qty * 10000),
	}

	resp, err := postJSON(gatewayURL+"/orders", order, token)
	if err != nil {
		fmt.Println("Marketmaker request error:", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		fmt.Println("Marketmaker response error:", resp.Status)
	}
	return resp.StatusCode == http.StatusUnauthorized
}
