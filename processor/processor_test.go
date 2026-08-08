package main

import (
	"reflect"
	"testing"
	"time"
)

func TestAmountsForPriceImprovement(t *testing.T) {
	// alici 101.0000'dan 2.0000 adet istedi, islem 100.0000'dan gerceklesti
	a, ok := amountsFor(TradeEvent{Symbol: "BTC-USD", Price: 1000000, BuyLimit: 1010000, Quantity: 20000})
	if !ok {
		t.Fatal("tutar hesaplanamadi")
	}
	got := []string{a.base, a.quote, a.buyerReserved.String(), a.buyerRefund.String(), a.baseAmount.String(), a.quoteAmount.String()}
	want := []string{"BTC", "USD", "20200000000", "200000000", "200000000", "20000000000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("beklenen %v, gelen %v", want, got)
	}
}

// Eski kayitlarda buy_limit yoktur; iade cikmamali, eksi tutar hic cikmamali.
func TestAmountsForWithoutLimit(t *testing.T) {
	a, ok := amountsFor(TradeEvent{Symbol: "ETH-USD", Price: 35000000, Quantity: 5000})
	if !ok || a.buyerRefund.Sign() != 0 || a.buyerReserved.Cmp(a.quoteAmount) != 0 {
		t.Fatalf("beklenmeyen tutarlar: %+v", a)
	}
}

func TestAmountsForRejectsBadInput(t *testing.T) {
	for _, tr := range []TradeEvent{
		{Symbol: "BTCUSD", Price: 1, Quantity: 1},
		{Symbol: "BTC-USD", Price: 0, Quantity: 1},
		{Symbol: "BTC-USD", Price: 1, Quantity: 0},
	} {
		if _, ok := amountsFor(tr); ok {
			t.Errorf("reddedilmeliydi: %+v", tr)
		}
	}
}

func TestReleaseAmount(t *testing.T) {
	cases := []struct {
		name string
		u    OrderUpdate
		want string
	}{
		{"alis kalani", OrderUpdate{Status: "CANCELLED", Symbol: "BTC-USD", Side: "BUY", Price: 1000000, Quantity: 50000, FilledQty: 20000}, "30000000000"},
		{"satis kalani", OrderUpdate{Status: "CANCELLED", Symbol: "BTC-USD", Side: "SELL", Price: 1000000, Quantity: 50000, FilledQty: 20000}, "300000000"},
		{"acik emir", OrderUpdate{Status: "OPEN", Symbol: "BTC-USD", Side: "BUY", Price: 1000000, Quantity: 50000}, ""},
		{"dolmus emir", OrderUpdate{Status: "FILLED", Symbol: "BTC-USD", Side: "BUY", Price: 1000000, Quantity: 50000, FilledQty: 50000}, ""},
		// engine emri bulamadiysa fiyat ve miktar gelmez
		{"bilinmeyen emir", OrderUpdate{Status: "CANCELLED", Symbol: "BTC-USD"}, ""},
	}
	for _, c := range cases {
		amount, ok := releaseAmount(c.u)
		got := ""
		if ok {
			got = amount.String()
		}
		if got != c.want {
			t.Errorf("%s: beklenen %q, gelen %q", c.name, c.want, got)
		}
	}
}

func TestKlineAccumulates(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 34, 56, 0, time.UTC)
	minute := klineMinute(ts)
	if minute != time.Date(2026, 3, 1, 12, 34, 0, 0, time.UTC).Unix() {
		t.Fatalf("dakika yanlis: %d", minute)
	}
	k := newKline(minute, 100, 5).with(120, 1).with(90, 2).with(110, 3)
	want := Kline{Time: minute, Open: 100, High: 120, Low: 90, Close: 110, Volume: 11}
	if k != want {
		t.Fatalf("beklenen %+v, gelen %+v", want, k)
	}
}

func TestStaleFields(t *testing.T) {
	got := staleFields([]string{"100", "200", "300", "bozuk"}, 200)
	want := []string{"100", "bozuk"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("beklenen %v, gelen %v", want, got)
	}
	if stale := staleFields([]string{"200", "300"}, 200); stale != nil {
		t.Fatalf("silinecek alan olmamaliydi: %v", stale)
	}
}

func TestIsOpenStatus(t *testing.T) {
	for status, want := range map[string]bool{"OPEN": true, "PARTIALLY_FILLED": true, "FILLED": false, "CANCELLED": false, "": false} {
		if isOpenStatus(status) != want {
			t.Errorf("%q icin %v bekleniyordu", status, want)
		}
	}
}
