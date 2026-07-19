package main

import "testing"

func TestDecodeOpenOrders(t *testing.T) {
	raw := map[string]string{
		"ord-1": `{"id":"ord-1","symbol":"BTC-USD","side":"BUY","price":100,"quantity":10,"filled_quantity":4,"status":"PARTIALLY_FILLED","timestamp":"2026-01-01T10:00:00Z"}`,
		"ord-2": `{"id":"ord-2","symbol":"BTC-USD","side":"SELL","price":120,"quantity":5,"filled_quantity":0,"status":"OPEN","timestamp":"2026-01-01T11:00:00Z"}`,
		"ord-3": `{"id":"ord-3","symbol":"ETH-USD","side":"BUY","price":30,"quantity":1,"filled_quantity":0,"status":"OPEN","timestamp":"2026-01-01T12:00:00Z"}`,
		"bozuk": `{`,
	}

	all := decodeOpenOrders(raw, "")
	if len(all) != 3 {
		t.Fatalf("3 emir bekleniyordu, gelen %d", len(all))
	}
	if all[0].ID != "ord-3" || all[2].ID != "ord-1" {
		t.Errorf("yeniden eskiye siralama bekleniyordu: %s, %s, %s", all[0].ID, all[1].ID, all[2].ID)
	}

	btc := decodeOpenOrders(raw, "BTC-USD")
	if len(btc) != 2 {
		t.Fatalf("BTC icin 2 emir bekleniyordu, gelen %d", len(btc))
	}
	if btc[1].ID != "ord-1" || btc[1].Remaining != 6 {
		t.Errorf("ord-1 icin kalan 6 bekleniyordu: %+v", btc[1])
	}

	if got := decodeOpenOrders(nil, ""); got == nil || len(got) != 0 {
		t.Errorf("bos liste [] olmali, gelen %#v", got)
	}
}

func TestOrderIDFormat(t *testing.T) {
	if !orderIDRe.MatchString(newOrderID()) {
		t.Fatal("uretilen id kendi kalibina uymuyor")
	}
	for _, bad := range []string{"", "ord-", "ord-XYZ", "../etc", "ord-0123456789abcdef0"} {
		if orderIDRe.MatchString(bad) {
			t.Errorf("gecersiz id kabul edildi: %q", bad)
		}
	}
}
