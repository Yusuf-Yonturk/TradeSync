package orderbook

import (
	"testing"
	"time"
)

func baseOrder() *Order {
	return &Order{
		ID:        "o1",
		UserID:    "u1",
		Symbol:    "BTC-USD",
		Side:      Buy,
		Price:     100,
		Quantity:  10,
		FilledQty: 0,
		Status:    Open,
		Timestamp: time.Now(),
	}
}

func TestRemainingQty(t *testing.T) {
	o := baseOrder()

	if o.RemainingQty() != 10 {
		t.Errorf("beklenen 10, gelen %d", o.RemainingQty())
	}

	o.FilledQty = 4
	if o.RemainingQty() != 6 {
		t.Errorf("beklenen 6, gelen %d", o.RemainingQty())
	}

	o.FilledQty = 10
	if o.RemainingQty() != 0 {
		t.Errorf("beklenen 0, gelen %d", o.RemainingQty())
	}
}

func TestIsFilled(t *testing.T) {
	o := baseOrder()

	if o.IsFilled() {
		t.Error("dolu olmamali")
	}

	o.FilledQty = 5
	if o.IsFilled() {
		t.Error("kismi dolmus, tam dolu olmamali")
	}

	o.FilledQty = 10
	if !o.IsFilled() {
		t.Error("tam dolu olmali")
	}
}

func TestUpdateOrderStatus(t *testing.T) {
	o := baseOrder()

	// Hic dolmamis
	updateOrderStatus(o)
	if o.Status != Open {
		t.Errorf("OPEN beklendi, %s geldi", o.Status)
	}

	// Kismi
	o.FilledQty = 5
	updateOrderStatus(o)
	if o.Status != PartiallyFilled {
		t.Errorf("PARTIALLY_FILLED beklendi, %s geldi", o.Status)
	}

	// Tam dolu
	o.FilledQty = 10
	updateOrderStatus(o)
	if o.Status != Filled {
		t.Errorf("FILLED beklendi, %s geldi", o.Status)
	}
}
