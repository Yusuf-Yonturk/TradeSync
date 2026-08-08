package money

import "testing"

func TestReserve(t *testing.T) {
	// 65000.00 USD fiyattan 0.5 BTC
	asset, amount, ok := Reserve("BTC-USD", "BUY", 650000000, 5000)
	if !ok || asset != "USD" || amount.String() != "3250000000000" {
		t.Fatalf("alis: %s %v %v", asset, amount, ok)
	}
	asset, amount, ok = Reserve("BTC-USD", "SELL", 650000000, 5000)
	if !ok || asset != "BTC" || amount.String() != "50000000" {
		t.Fatalf("satis: %s %v %v", asset, amount, ok)
	}
	for _, bad := range []struct {
		symbol, side string
		price, qty   int64
	}{
		{"BTCUSD", "BUY", 1, 1},
		{"BTC-USD-X", "BUY", 1, 1},
		{"BTC-USD", "HOLD", 1, 1},
		{"BTC-USD", "BUY", 0, 1},
		{"BTC-USD", "SELL", 1, 0},
	} {
		if _, _, ok := Reserve(bad.symbol, bad.side, bad.price, bad.qty); ok {
			t.Errorf("reddedilmeliydi: %+v", bad)
		}
	}
}

// int64'e sigmayan carpimlar tasmamali.
func TestNotionalDoesNotOverflow(t *testing.T) {
	got := Notional(1e13, 1e13).String()
	if got != "100000000000000000000000000" {
		t.Fatalf("beklenmeyen tutar: %s", got)
	}
}

// Kismi dolumlarla iptalin toplami, bastaki ayrilan tutara esit olmali.
func TestPartialFillsAddUp(t *testing.T) {
	const price, qty = 123456789, 98765
	fills := []int64{1, 333, 40000}
	sum := Notional(price, 0)
	var filled int64
	for _, f := range fills {
		sum.Add(sum, Notional(price, f))
		filled += f
	}
	sum.Add(sum, Notional(price, qty-filled))
	if sum.Cmp(Notional(price, qty)) != 0 {
		t.Fatalf("toplam tutmuyor: %s != %s", sum, Notional(price, qty))
	}
}

func TestFromUnits(t *testing.T) {
	if got := FromUnits(5).String(); got != "500000000" {
		t.Fatalf("beklenmeyen: %s", got)
	}
}
