package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tradesync/shared/money"
)

// Gercek bir postgres ister: TEST_POSTGRES_DSN tanimli degilse atlanir.
func testLedger(t *testing.T) *ledger {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN tanimli degil")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := applySchema(context.Background(), db); err != nil {
		t.Fatalf("sema: %v", err)
	}
	return &ledger{db: db, funds: []demoFund{{"USD", 1000}, {"BTC", 2}}, exempt: map[string]bool{}}
}

func newUser(t *testing.T, l *ledger) string {
	t.Helper()
	var id string
	err := l.db.QueryRow(context.Background(),
		`INSERT INTO users (username, password_hash) VALUES ($1, 'x') RETURNING id`,
		fmt.Sprintf("u%d", time.Now().UnixNano()),
	).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func balanceOf(t *testing.T, l *ledger, user, asset string) [2]string {
	t.Helper()
	var b [2]string
	err := l.db.QueryRow(context.Background(),
		`SELECT available::text, locked::text FROM balances WHERE user_id = $1 AND asset = $2`, user, asset,
	).Scan(&b[0], &b[1])
	if err != nil {
		return [2]string{"yok", "yok"}
	}
	return b
}

func TestLedgerReserveAndUnreserve(t *testing.T) {
	l := testLedger(t)
	ctx := context.Background()
	user := newUser(t, l)
	order := "ord-" + user[:8]

	// hesap henuz bakiye almadi; ilk emir deneme bakiyesini yukleyip ayirir
	asset, amount, _ := money.Reserve("BTC-USD", "BUY", 1000000, 40000) // 100'den 4 adet = 400 USD
	if err := l.reserve(ctx, order, user, asset, amount); err != nil {
		t.Fatalf("ayirma: %v", err)
	}
	if got := balanceOf(t, l, user, "USD"); got != [2]string{"60000000000", "40000000000"} {
		t.Fatalf("ayirma sonrasi USD %v", got)
	}
	if got := balanceOf(t, l, user, "BTC"); got != [2]string{"200000000", "0"} {
		t.Fatalf("BTC deneme bakiyesi %v", got)
	}

	// kalan 600 USD'den fazlasi istenirse reddedilir ve bakiye degismez
	_, tooMuch, _ := money.Reserve("BTC-USD", "BUY", 1000000, 60001)
	if err := l.reserve(ctx, order+"x", user, "USD", tooMuch); !errors.Is(err, errInsufficient) {
		t.Fatalf("yetersiz bakiye bekleniyordu: %v", err)
	}
	// elinde olmayan varligi satamaz
	_, sol, _ := money.Reserve("SOL-USD", "SELL", 1000000, 1)
	if err := l.reserve(ctx, order+"y", user, "SOL", sol); !errors.Is(err, errInsufficient) {
		t.Fatalf("olmayan varlik icin yetersiz bakiye bekleniyordu: %v", err)
	}
	if got := balanceOf(t, l, user, "USD"); got != [2]string{"60000000000", "40000000000"} {
		t.Fatalf("reddedilen emir bakiyeyi degistirdi: %v", got)
	}

	// deneme bakiyesi ikinci kez verilmez
	if granted, err := l.ensureFunds(ctx, user); err != nil || granted {
		t.Fatalf("ikinci yukleme: granted=%v err=%v", granted, err)
	}

	// emir iletilemezse ayrilan tutar geri doner; ikinci cagri etkisizdir
	for i := 0; i < 2; i++ {
		if err := l.unreserve(ctx, order); err != nil {
			t.Fatal(err)
		}
	}
	if got := balanceOf(t, l, user, "USD"); got != [2]string{"100000000000", "0"} {
		t.Fatalf("geri verme sonrasi USD %v", got)
	}
}

func TestLedgerExemptUserIsNotChecked(t *testing.T) {
	l := testLedger(t)
	user := newUser(t, l)
	l.exempt[user] = true

	_, huge, _ := money.Reserve("BTC-USD", "BUY", 1e13, 1e13)
	if err := l.reserve(context.Background(), "ord-mm", user, "USD", huge); err != nil {
		t.Fatalf("muaf hesap reddedildi: %v", err)
	}
	if got := balanceOf(t, l, user, "USD"); got != [2]string{"yok", "yok"} {
		t.Fatalf("muaf hesap icin bakiye satiri olusmamaliydi: %v", got)
	}
}

func TestProvisionMarketMakers(t *testing.T) {
	l := testLedger(t)
	ctx := context.Background()
	base := fmt.Sprintf("mm%d", time.Now().UnixNano()%1e9)

	none, err := provisionMarketMakers(ctx, l.db, base, "")
	if err != nil || len(none) != 0 {
		t.Fatalf("sifre yokken hesap acilmamali: %v %v", none, err)
	}

	first, err := provisionMarketMakers(ctx, l.db, base, "first-password")
	if err != nil || len(first) != 2 {
		t.Fatalf("iki hesap bekleniyordu: %v %v", first, err)
	}
	// sifre degisince ayni hesaplar guncellenir, yenisi acilmaz
	second, err := provisionMarketMakers(ctx, l.db, base, "second-password")
	if err != nil || len(second) != 2 {
		t.Fatalf("ikinci calistirma: %v %v", second, err)
	}
	for id := range first {
		if !second[id] {
			t.Fatalf("hesap kimligi degisti: %v / %v", first, second)
		}
	}
	if _, err := provisionMarketMakers(ctx, l.db, base, "kisa"); err == nil {
		t.Fatal("kisa sifre reddedilmeliydi")
	}
}
