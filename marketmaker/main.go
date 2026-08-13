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

// Bu sureden uzun bekleyen emirler iptal edilir; aksi halde fiyat uzaklastikca
// dolmayan emirler defterde birikir.
const orderTTL = 2 * time.Minute

// Gateway kullanicinin kendi emriyle eslesmesini engelledigi icin bot iki
// hesap kullanir: biri yalniz alir, digeri yalniz satar.
type account struct {
	username string
	token    string
	open     []placedOrder
}

type placedOrder struct {
	id string
	at time.Time
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func send(method, url string, body any, bearer string) (*http.Response, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return client.Do(req)
}

func postJSON(url string, body any, bearer string) (*http.Response, error) {
	return send(http.MethodPost, url, body, bearer)
}

// Hesaplari normalde gateway acar (MM_PASSWORD tanimliysa); degilse bot kendisi
// kaydolur ve siradan bir kullanici gibi deneme bakiyesiyle calisir.
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
	buyer := &account{username: username}
	seller := &account{username: username + "-sell"}
	for _, acc := range []*account{buyer, seller} {
		acc.token = mustLogin(gatewayURL, acc.username, password)
	}
	fmt.Println("Market Maker başladı! Canlı veriler akıtılıyor...")

	btcPrice := 65000.0
	ethPrice := 3500.0

	// Sonsuz döngüde rastgele emirler oluştur
	for {
		// BTC-USD ve ETH-USD için emirler atalım
		simulateMarket(gatewayURL, "BTC-USD", &btcPrice, 50.0, buyer, seller) // 50$ volatilite
		simulateMarket(gatewayURL, "ETH-USD", &ethPrice, 5.0, buyer, seller)  // 5$ volatilite

		for _, acc := range []*account{buyer, seller} {
			cancelStale(gatewayURL, acc, time.Now())
			if acc.token == "" {
				acc.token = mustLogin(gatewayURL, acc.username, password)
			}
		}

		// 300 ile 800 ms arası rastgele bekle (akıcı hissi versin)
		time.Sleep(time.Duration(rand.Intn(500)+300) * time.Millisecond)
	}
}

// Suresi dolanlari ve kalanlari ayirir; emirler eskiden yeniye siralidir.
func splitStale(open []placedOrder, now time.Time, ttl time.Duration) (stale, fresh []placedOrder) {
	i := 0
	for i < len(open) && now.Sub(open[i].at) >= ttl {
		i++
	}
	return open[:i], open[i:]
}

// Suresi dolan emirleri iptal eder. Dolmus emir icin gateway 404 doner, sorun degil.
func cancelStale(gatewayURL string, acc *account, now time.Time) {
	stale, fresh := splitStale(acc.open, now, orderTTL)
	for i, o := range stale {
		resp, err := send(http.MethodDelete, gatewayURL+"/orders/"+o.id, nil, acc.token)
		if err != nil {
			fmt.Println("Marketmaker iptal hatasi:", err)
			// kalanlar bir sonraki turda yeniden denenir
			fresh = append(append([]placedOrder{}, stale[i:]...), fresh...)
			break
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			acc.token = ""
			fresh = append(append([]placedOrder{}, stale[i:]...), fresh...)
			break
		}
	}
	acc.open = append([]placedOrder{}, fresh...)
}

func simulateMarket(gatewayURL string, symbol string, currentPrice *float64, volatility float64, buyer, seller *account) {
	// Fiyatı rastgele ufak bir miktar oynat
	change := (rand.Float64() * volatility * 2) - volatility
	*currentPrice += change
	// rastgele yuruyus sifirin altina inmesin
	if *currentPrice < volatility*4 {
		*currentPrice = volatility * 4
	}

	side, acc := "BUY", buyer
	if rand.Intn(2) == 0 {
		side, acc = "SELL", seller
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

	resp, err := postJSON(gatewayURL+"/orders", order, acc.token)
	if err != nil {
		fmt.Println("Marketmaker request error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		acc.token = ""
		return
	}
	if resp.StatusCode >= 400 {
		fmt.Println("Marketmaker response error:", resp.Status)
		return
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err == nil && out.ID != "" {
		acc.open = append(acc.open, placedOrder{id: out.ID, at: time.Now()})
	}
}
