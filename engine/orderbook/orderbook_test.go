package orderbook

import (
	"testing"
	"time"
)

func newOrder(id string, side Side, price, qty int64, t time.Time) *Order {
	return &Order{
		ID:        id,
		UserID:    "user-1",
		Symbol:    "BTC-USD",
		Side:      side,
		Price:     price,
		Quantity:  qty,
		Status:    Open,
		Timestamp: t,
	}
}

func TestFullMatch(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	sell := newOrder("s1", Sell, 100, 10, now)
	ob.AddOrder(sell)

	buy := newOrder("b1", Buy, 100, 10, now.Add(time.Second))
	trades := ob.AddOrder(buy)

	if len(trades) != 1 {
		t.Fatalf("1 trade beklendi, %d geldi", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("qty=10 beklendi, %d geldi", trades[0].Quantity)
	}
	if ob.Buys.Len() != 0 || ob.Sells.Len() != 0 {
		t.Error("book boş olmalıydı")
	}
}

func TestPartialFill(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	// 10 lot sat
	ob.AddOrder(newOrder("s1", Sell, 100, 10, now))

	// 15 lot al — 10 eşleşir, 5 book'ta kalır
	buy := newOrder("b1", Buy, 100, 15, now.Add(time.Second))
	trades := ob.AddOrder(buy)

	if len(trades) != 1 {
		t.Fatalf("1 trade beklendi, %d geldi", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("qty=10 beklendi, %d geldi", trades[0].Quantity)
	}
	if ob.Buys.Len() != 1 {
		t.Error("kalan buy book'ta olmalıydı")
	}
	if ob.BestBuy().RemainingQty() != 5 {
		t.Errorf("kalan qty=5 beklendi, %d geldi", ob.BestBuy().RemainingQty())
	}
}

func TestMultipleSellsPartialFill(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	// 3 ayrı sell emri
	ob.AddOrder(newOrder("s1", Sell, 100, 5, now))
	ob.AddOrder(newOrder("s2", Sell, 100, 5, now.Add(time.Second)))
	ob.AddOrder(newOrder("s3", Sell, 100, 5, now.Add(2*time.Second)))

	// 12 lot buy — s1 ve s2 tamamen, s3 kısmen eşleşmeli
	buy := newOrder("b1", Buy, 100, 12, now.Add(3*time.Second))
	trades := ob.AddOrder(buy)

	if len(trades) != 3 {
		t.Fatalf("3 trade beklendi, %d geldi", len(trades))
	}
	totalQty := int64(0)
	for _, tr := range trades {
		totalQty += tr.Quantity
	}
	if totalQty != 12 {
		t.Errorf("toplam qty=12 beklendi, %d geldi", totalQty)
	}
}

func TestNoMatchQueues(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	// Sell 110, buy 100 — fiyat uymaz
	ob.AddOrder(newOrder("s1", Sell, 110, 10, now))
	trades := ob.AddOrder(newOrder("b1", Buy, 100, 10, now.Add(time.Second)))

	if len(trades) != 0 {
		t.Error("eşleşme olmamalıydı")
	}
	if ob.Buys.Len() != 1 || ob.Sells.Len() != 1 {
		t.Error("her iki emir book'ta olmalıydı")
	}
}

func TestPriceTimePriority(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	// Aynı fiyat, farklı zaman — önce gelen önce eşleşmeli
	ob.AddOrder(newOrder("s1", Sell, 100, 5, now))
	ob.AddOrder(newOrder("s2", Sell, 100, 5, now.Add(time.Second)))

	buy := newOrder("b1", Buy, 100, 5, now.Add(2*time.Second))
	trades := ob.AddOrder(buy)

	if trades[0].SellOrderID != "s1" {
		t.Errorf("s1 önce eşleşmeliydi, %s eşleşti", trades[0].SellOrderID)
	}
}

func TestCancelBuyOrder(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	buy := newOrder("b1", Buy, 100, 10, now)
	ob.AddOrder(buy)

	ok := ob.CancelOrder(buy)
	if !ok {
		t.Fatal("iptal basarisiz")
	}
	if buy.Status != Cancelled {
		t.Errorf("CANCELLED beklendi, %s geldi", buy.Status)
	}
	if ob.Buys.Len() != 0 {
		t.Error("book bos olmali")
	}
}

func TestCancelFilledOrderFails(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	sell := newOrder("s1", Sell, 100, 10, now)
	buy := newOrder("b1", Buy, 100, 10, now.Add(time.Second))
	ob.AddOrder(sell)
	ob.AddOrder(buy)

	// Dolu emri iptal etmeye calis
	ok := ob.CancelOrder(sell)
	if ok {
		t.Error("dolu emir iptal edilemez")
	}
}

func TestCancelledOrderDoesNotMatch(t *testing.T) {
	ob := NewOrderBook()
	now := time.Now()

	sell := newOrder("s1", Sell, 100, 10, now)
	ob.AddOrder(sell)
	ob.CancelOrder(sell)

	// Sell iptal edildikten sonra buy gelse eslesmemeli
	trades := ob.AddOrder(newOrder("b1", Buy, 100, 10, now.Add(time.Second)))
	if len(trades) != 0 {
		t.Error("iptal edilen emirle eslesmemeli")
	}
}
