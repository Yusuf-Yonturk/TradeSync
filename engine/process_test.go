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

func TestIsLive(t *testing.T) {
	committed := map[int32]int64{0: 5}

	if isLive(committed, 0, 4) {
		t.Error("offset 4 daha once islenmis, canli sayildi")
	}
	if !isLive(committed, 0, 5) {
		t.Error("offset 5 henuz islenmemis, tekrar sayildi")
	}
	// hic offset kaydi olmayan partition bastan canli islenir
	if !isLive(committed, 1, 0) {
		t.Error("kaydi olmayan partition tekrar sayildi")
	}
}

// Ayni mesajlar bastan uygulandiginda defter ayni duruma gelmeli;
// yeniden baslatmada defterin geri kurulmasi buna dayanir.
func TestReplayRebuildsSameBook(t *testing.T) {
	history := []IncomingOrder{
		newOrder("s1", "u1", "SELL", 100, 5),
		newOrder("s2", "u1", "SELL", 101, 5),
		newOrder("b1", "u2", "BUY", 100, 2),
		newOrder("b2", "u2", "BUY", 90, 4),
		{Type: typeCancel, ID: "s2", UserID: "u1", Symbol: "BTC-USD"},
		newOrder("b3", "u3", "BUY", 95, 1),
	}

	build := func() *orderbook.OrderBook {
		book := orderbook.NewOrderBook()
		for _, in := range history {
			process(book, in)
		}
		return book
	}

	first, second := build(), build()
	a, b := first.Snapshot("BTC-USD", 15), second.Snapshot("BTC-USD", 15)
	if len(a.Bids) != 2 || len(a.Asks) != 1 || a.Asks[0].Quantity != 3 {
		t.Fatalf("beklenmeyen defter: %+v", a)
	}
	if len(a.Bids) != len(b.Bids) || len(a.Asks) != len(b.Asks) {
		t.Fatalf("tekrar farkli defter uretti: %+v / %+v", a, b)
	}
	for i := range a.Bids {
		if a.Bids[i] != b.Bids[i] {
			t.Errorf("bid %d farkli: %+v / %+v", i, a.Bids[i], b.Bids[i])
		}
	}
	if second.Get("s2") != nil || second.Get("s1") == nil || second.Get("s1").RemainingQty() != 3 {
		t.Error("tekrar sonrasi emir durumlari tutmuyor")
	}
}

func TestSelfTradeCancelsIncomingRemainder(t *testing.T) {
	book := orderbook.NewOrderBook()
	process(book, newOrder("s1", "u2", "SELL", 100, 2))
	process(book, newOrder("s2", "u1", "SELL", 101, 5))

	in := newOrder("b1", "u1", "BUY", 105, 6)
	in.STP = true
	res := process(book, in)

	// once baskasinin emriyle eslesir, kendi emrine gelince durur
	if len(res.trades) != 1 || res.trades[0].SellOrderID != "s1" || res.trades[0].Quantity != 2 {
		t.Fatalf("yalniz s1 ile eslesme bekleniyordu: %+v", res.trades)
	}
	if res.trades[0].BuyLimit != 105 || res.trades[0].Price != 100 {
		t.Errorf("limit/fiyat yanlis: %+v", res.trades[0])
	}
	u := res.updates[0]
	if u.ID != "b1" || u.Status != "CANCELLED" || u.FilledQty != 2 || u.Quantity != 6 || u.Price != 105 {
		t.Errorf("gelen emrin kalani iptal edilmeliydi: %+v", u)
	}
	if book.Get("b1") != nil {
		t.Error("iptal edilen emir defterde kaldi")
	}
	if s2 := book.Get("s2"); s2 == nil || s2.RemainingQty() != 5 {
		t.Error("kullanicinin bekleyen emri degismemeliydi")
	}
}

// Kural isaretlenmemis (eski) kayitlarda uygulanmaz; yeniden oynatma eski sonucu vermeli.
func TestSelfTradeAllowedForLegacyOrders(t *testing.T) {
	book := orderbook.NewOrderBook()
	process(book, newOrder("s1", "u1", "SELL", 100, 2))
	res := process(book, newOrder("b1", "u1", "BUY", 100, 2))
	if len(res.trades) != 1 {
		t.Fatalf("eski emirler eslesmeliydi: %+v", res.trades)
	}
}
