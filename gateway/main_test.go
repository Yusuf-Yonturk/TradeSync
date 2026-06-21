package main

import "testing"

func validRequest() OrderRequest {
	return OrderRequest{UserID: "u1", Symbol: "BTC-USD", Side: "BUY", Price: 650000000, Quantity: 10000}
}

func TestValidate(t *testing.T) {
	if msg := validate(validRequest()); msg != "" {
		t.Fatalf("gecerli emir reddedildi: %s", msg)
	}

	cases := map[string]func(*OrderRequest){
		"bos symbol":      func(r *OrderRequest) { r.Symbol = "" },
		"kucuk harf":      func(r *OrderRequest) { r.Symbol = "btc-usd" },
		"tiresiz symbol":  func(r *OrderRequest) { r.Symbol = "BTCUSD" },
		"bosluklu symbol": func(r *OrderRequest) { r.Symbol = "BTC-USD " },
		"uzun symbol":     func(r *OrderRequest) { r.Symbol = "AAAAAAAAAAAAAAAA-USD" },
		"gecersiz side":   func(r *OrderRequest) { r.Side = "HOLD" },
		"sifir fiyat":     func(r *OrderRequest) { r.Price = 0 },
		"negatif fiyat":   func(r *OrderRequest) { r.Price = -1 },
		"asiri fiyat":     func(r *OrderRequest) { r.Price = maxPrice + 1 },
		"sifir miktar":    func(r *OrderRequest) { r.Quantity = 0 },
		"negatif miktar":  func(r *OrderRequest) { r.Quantity = -1 },
		"asiri miktar":    func(r *OrderRequest) { r.Quantity = maxQuantity + 1 },
	}
	for name, mutate := range cases {
		r := validRequest()
		mutate(&r)
		if msg := validate(r); msg == "" {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
}
