package main

import "testing"

func validOrder() IncomingOrder {
	return IncomingOrder{ID: "o1", UserID: "u1", Symbol: "BTC-USD", Side: "BUY", Price: 100, Quantity: 5}
}

func TestValidateIncoming(t *testing.T) {
	if err := validateIncoming(validOrder()); err != nil {
		t.Fatalf("gecerli emir reddedildi: %v", err)
	}

	cases := map[string]func(*IncomingOrder){
		"bos id":         func(o *IncomingOrder) { o.ID = "" },
		"bos symbol":     func(o *IncomingOrder) { o.Symbol = "" },
		"gecersiz side":  func(o *IncomingOrder) { o.Side = "buy" },
		"sifir fiyat":    func(o *IncomingOrder) { o.Price = 0 },
		"negatif fiyat":  func(o *IncomingOrder) { o.Price = -1 },
		"sifir miktar":   func(o *IncomingOrder) { o.Quantity = 0 },
		"negatif miktar": func(o *IncomingOrder) { o.Quantity = -5 },
	}
	for name, mutate := range cases {
		o := validOrder()
		mutate(&o)
		if err := validateIncoming(o); err == nil {
			t.Errorf("%s: hata bekleniyordu", name)
		}
	}
}
