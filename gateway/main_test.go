package main

import "testing"

func validRequest() OrderRequest {
	return OrderRequest{Symbol: "BTC-USD", Side: "BUY", Price: 650000000, Quantity: 10000}
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

func TestValidateRegister(t *testing.T) {
	ok := RegisterRequest{Name: "Ada", Username: "ada_1", Password: "cokgizli1"}
	if msg := validateRegister(ok); msg != "" {
		t.Fatalf("gecerli kayit reddedildi: %s", msg)
	}

	long := make([]byte, maxPasswordLen+1)
	for i := range long {
		long[i] = 'a'
	}
	cases := map[string]func(*RegisterRequest){
		"kisa kullanici adi":   func(r *RegisterRequest) { r.Username = "ab" },
		"bosluklu kullanici":   func(r *RegisterRequest) { r.Username = "ada lovelace" },
		"kisa sifre":           func(r *RegisterRequest) { r.Password = "1234567" },
		"bcrypt sinirini asan": func(r *RegisterRequest) { r.Password = string(long) },
	}
	for name, mutate := range cases {
		r := ok
		mutate(&r)
		if msg := validateRegister(r); msg == "" {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
}
