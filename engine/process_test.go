package main

import (
	"testing"

	"tradesync/engine/orderbook"
)

func newOrder(id, user, side string, price, qty int64) IncomingOrder {
	return IncomingOrder{ID: id, UserID: user, Symbol: "BTC-USD", Side: side, Price: price, Quantity: qty}
}

func statusOf(updates []OrderUpdate, id string) string {
	for _, u := range updates {
		if u.ID == id {
			return u.Status
		}
	}
	return ""
}

func TestProcessRestingOrderIsOpen(t *testing.T) {
	book := orderbook.NewOrderBook()
	res := process(book, newOrder("s1", "u1", "SELL", 100, 5))

	if len(res.trades) != 0 {
		t.Fatalf("eslesme beklenmiyordu: %+v", res.trades)
	}
	if len(res.updates) != 1 || res.updates[0].Status != "OPEN" || res.updates[0].UserID != "u1" {
		t.Fatalf("tek OPEN guncellemesi bekleniyordu: %+v", res.updates)
	}
}

func TestProcessMatchReportsBothSides(t *testing.T) {
	book := orderbook.NewOrderBook()
	process(book, newOrder("s1", "u1", "SELL", 100, 5))
	process(book, newOrder("s2", "u1", "SELL", 101, 5))

	// s1 tamamen, s2 kismen dolar; gelen emir tamamen dolar
	res := process(book, newOrder("b1", "u2", "BUY", 101, 7))

	if len(res.trades) != 2 {
		t.Fatalf("2 trade bekleniyordu, gelen %d", len(res.trades))
	}
	want := map[string]string{"b1": "FILLED", "s1": "FILLED", "s2": "PARTIALLY_FILLED"}
	for id, status := range want {
		if got := statusOf(res.updates, id); got != status {
			t.Errorf("%s icin %s bekleniyordu, gelen %q", id, status, got)
		}
	}
	for _, u := range res.updates {
		if u.ID == "s1" && u.UserID != "u1" {
			t.Errorf("dolan emrin sahibi kayboldu: %+v", u)
		}
		if u.ID == "s2" && u.FilledQty != 2 {
			t.Errorf("s2 icin dolan miktar 2 bekleniyordu, gelen %d", u.FilledQty)
		}
	}
}

func TestProcessCancel(t *testing.T) {
	book := orderbook.NewOrderBook()
	process(book, newOrder("b1", "u1", "BUY", 90, 5))

	cancel := func(id, user string) result {
		return process(book, IncomingOrder{Type: typeCancel, ID: id, UserID: user, Symbol: "BTC-USD"})
	}

	if res := cancel("b1", "saldirgan"); len(res.updates) != 0 || book.Get("b1") == nil {
		t.Fatalf("baska kullanici emri iptal edebildi: %+v", res.updates)
	}

	res := cancel("b1", "u1")
	if statusOf(res.updates, "b1") != "CANCELLED" || book.Get("b1") != nil {
		t.Fatalf("iptal bekleniyordu: %+v", res.updates)
	}

	// defterde olmayan emir icin de liste temizlensin
	res = cancel("b1", "u1")
	if statusOf(res.updates, "b1") != "CANCELLED" {
		t.Errorf("bilinmeyen emir icin CANCELLED bekleniyordu: %+v", res.updates)
	}

	// iptal edilen emir eslesmemeli
	if res := process(book, newOrder("s1", "u2", "SELL", 90, 5)); len(res.trades) != 0 {
		t.Errorf("iptal edilen emir eslesti: %+v", res.trades)
	}
}
