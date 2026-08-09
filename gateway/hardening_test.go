package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := newRateLimiter(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if l.blockedFor("a") != 0 {
			t.Fatalf("%d. denemede engellendi", i+1)
		}
		l.add("a")
	}
	if wait := l.blockedFor("a"); wait != time.Minute {
		t.Fatalf("sinirda engellenmeliydi, bekleme=%v", wait)
	}
	if l.blockedFor("b") != 0 {
		t.Fatal("baska anahtar etkilenmemeliydi")
	}

	now = now.Add(61 * time.Second)
	if l.blockedFor("a") != 0 {
		t.Fatal("pencere dolunca engel kalkmaliydi")
	}
	l.add("b")
	if _, ok := l.entries["a"]; ok {
		t.Fatal("suresi dolan kayit temizlenmeliydi")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.5:4321"
	r.Header.Set("X-Real-IP", "203.0.113.9")

	if got := clientIP(r, ""); got != "10.0.0.5" {
		t.Errorf("baslik guvenilmezken baglanti adresi bekleniyordu: %q", got)
	}
	if got := clientIP(r, "X-Real-IP"); got != "203.0.113.9" {
		t.Errorf("baslik okunmaliydi: %q", got)
	}
	if got := clientIP(r, "CF-Connecting-IP"); got != "10.0.0.5" {
		t.Errorf("baslik yoksa baglanti adresine donmeliydi: %q", got)
	}
}

func TestParseDemoFunds(t *testing.T) {
	funds, err := parseDemoFunds(defaultDemoFunds)
	if err != nil || len(funds) != 4 || funds[0] != (demoFund{"USD", 100000}) {
		t.Fatalf("varsayilan cozulemedi: %v %v", funds, err)
	}
	if funds, err := parseDemoFunds(""); err != nil || len(funds) != 0 {
		t.Fatalf("bos tanim gecerli olmali: %v %v", funds, err)
	}
	for _, bad := range []string{"USD", "usd:5", "USD:-1", "USD:abc", "USD:0"} {
		if _, err := parseDemoFunds(bad); err == nil {
			t.Errorf("%q reddedilmeliydi", bad)
		}
	}
}

// Yayina giden islemde kullanici ve emir kimligi olmamali.
func TestPublicTradeMessageDropsIdentities(t *testing.T) {
	raw := []byte(`{"trade_id":"0-7-0","buy_order_id":"ord-1","sell_order_id":"ord-2","buyer_id":"alice","seller_id":"bob","symbol":"BTC-USD","price":650000000,"buy_limit":660000000,"quantity":5000,"timestamp":"2026-01-01T00:00:00Z"}`)
	out, ok := publicTradeMessage(raw)
	if !ok {
		t.Fatal("cevrilemedi")
	}
	got := string(out)
	for _, secret := range []string{"alice", "bob", "ord-1", "ord-2", "buyer", "seller", "buy_limit", "trade_id"} {
		if strings.Contains(got, secret) {
			t.Errorf("%q disari sizdi: %s", secret, got)
		}
	}
	for _, want := range []string{`"symbol":"BTC-USD"`, `"price":650000000`, `"quantity":5000`, `"timestamp":"2026-01-01T00:00:00Z"`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s eksik: %s", want, got)
		}
	}
	if _, ok := publicTradeMessage([]byte("bozuk")); ok {
		t.Error("bozuk kayit yayinlanmamaliydi")
	}
}

func TestMySide(t *testing.T) {
	if side, order := mySide("u1", "u1", "b", "s"); side != "BUY" || order != "b" {
		t.Errorf("alici: %s %s", side, order)
	}
	if side, order := mySide("u2", "u1", "b", "s"); side != "SELL" || order != "s" {
		t.Errorf("satici: %s %s", side, order)
	}
}

func TestParseLimit(t *testing.T) {
	for raw, want := range map[string]int{"": 50, "10": 10, "500": 500, "501": 50, "0": 50, "-3": 50, "x": 50} {
		if got := parseLimit(raw, 50, 500); got != want {
			t.Errorf("%q icin %d bekleniyordu, gelen %d", raw, want, got)
		}
	}
}

func TestMarketMakerUsernames(t *testing.T) {
	got := marketMakerUsernames("marketmaker")
	if len(got) != 2 || got[0] != "marketmaker" || got[1] != "marketmaker-sell" || !usernameRe.MatchString(got[1]) {
		t.Fatalf("beklenmeyen: %v", got)
	}
}
