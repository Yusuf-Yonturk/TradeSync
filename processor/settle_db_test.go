package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Gercek bir postgres ister: TEST_POSTGRES_DSN tanimli degilse atlanir.
func testDB(t *testing.T) *pgxpool.Pool {
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
	// ikinci kez uygulanabilmeli
	if err := applySchema(context.Background(), db); err != nil {
		t.Fatalf("sema tekrar: %v", err)
	}
	return db
}

type balance struct{ available, locked string }

func balanceOf(t *testing.T, db *pgxpool.Pool, user, asset string) balance {
	t.Helper()
	var b balance
	err := db.QueryRow(context.Background(),
		`SELECT available::text, locked::text FROM balances WHERE user_id = $1 AND asset = $2`, user, asset,
	).Scan(&b.available, &b.locked)
	if err != nil {
		return balance{"0", "0"}
	}
	return b
}

func lockCount(t *testing.T, db *pgxpool.Pool, orders ...string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(context.Background(), `SELECT count(*) FROM order_locks WHERE order_id = ANY($1)`, orders).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Gateway'in emir alirken yaptigi ayirmayi taklit eder.
func reserve(t *testing.T, db *pgxpool.Pool, order, user, asset, start, amount string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Exec(ctx, `
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, $2, $3::text::numeric - $4::text::numeric, $4::text::numeric)
	`, user, asset, start, amount); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO order_locks (order_id, user_id, asset, remaining) VALUES ($1, $2, $3, $4::text::numeric)
	`, order, user, asset, amount); err != nil {
		t.Fatal(err)
	}
}

func TestSettleTradeAndRelease(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	run := fmt.Sprint(time.Now().UnixNano())
	buyer, seller := "buyer-"+run, "seller-"+run
	buyOrder, sellOrder := "b-"+run, "s-"+run

	// alici 101'den 5 adet (505 USD), satici 5 adet ayirdi; tutarlar 10^8 olcekli
	reserve(t, db, buyOrder, buyer, "USD", "100000000000", "50500000000")
	reserve(t, db, sellOrder, seller, "BTC", "1000000000", "500000000")

	// 100'den 2 adet eslesti
	trade := TradeEvent{
		ID: "t-" + run, BuyOrderID: buyOrder, SellOrderID: sellOrder, BuyerID: buyer, SellerID: seller,
		Symbol: "BTC-USD", Price: 1000000, BuyLimit: 1010000, Quantity: 20000, Timestamp: time.Now(),
	}
	inserted, err := settleTrade(ctx, db, trade)
	if err != nil || !inserted {
		t.Fatalf("ilk yazim: inserted=%v err=%v", inserted, err)
	}

	check := func(step string) {
		t.Helper()
		// alici: 505 ayrilmisti, 202'si harcandi (2 iade), 303 kilitli kaldi
		if got := balanceOf(t, db, buyer, "USD"); got != (balance{"49700000000", "30300000000"}) {
			t.Errorf("%s: alici USD %+v", step, got)
		}
		if got := balanceOf(t, db, buyer, "BTC"); got != (balance{"200000000", "0"}) {
			t.Errorf("%s: alici BTC %+v", step, got)
		}
		if got := balanceOf(t, db, seller, "BTC"); got != (balance{"500000000", "300000000"}) {
			t.Errorf("%s: satici BTC %+v", step, got)
		}
		if got := balanceOf(t, db, seller, "USD"); got != (balance{"20000000000", "0"}) {
			t.Errorf("%s: satici USD %+v", step, got)
		}
	}
	check("ilk")

	// ayni islem tekrar gelirse hicbir sey degismemeli
	inserted, err = settleTrade(ctx, db, trade)
	if err != nil || inserted {
		t.Fatalf("tekrar: inserted=%v err=%v", inserted, err)
	}
	check("tekrar")

	// alici kalan 3 adedi iptal etti: 303 serbest kalir, satir silinir
	amount, ok := releaseAmount(OrderUpdate{Status: "CANCELLED", Symbol: "BTC-USD", Side: "BUY", Price: 1010000, Quantity: 50000, FilledQty: 20000})
	if !ok {
		t.Fatal("serbest birakilacak tutar yok")
	}
	for i := 0; i < 2; i++ { // iptal iki kez gelse de bir kez uygulanir
		if err := releaseOrder(ctx, db, buyOrder, amount); err != nil {
			t.Fatal(err)
		}
	}
	if got := balanceOf(t, db, buyer, "USD"); got != (balance{"80000000000", "0"}) {
		t.Errorf("iptal sonrasi alici USD %+v", got)
	}
	if n := lockCount(t, db, buyOrder); n != 0 {
		t.Errorf("alicinin kilit satiri silinmeliydi")
	}
	if n := lockCount(t, db, sellOrder); n != 1 {
		t.Errorf("saticinin kilidi durmaliydi")
	}
}

// Iptal, emrin onceki islemi islenmeden once gelirse o islemin payi serbest kalmamali.
func TestReleaseBeforeEarlierTradeSettles(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	run := fmt.Sprint(time.Now().UnixNano())
	buyer, seller := "buyer2-"+run, "seller2-"+run
	buyOrder := "b2-" + run

	reserve(t, db, buyOrder, buyer, "USD", "50000000000", "50000000000") // 100'den 5 adet

	// engine sirasi: 2 adet eslesme, sonra kalan 3 adedin iptali. Iptal once isleniyor.
	amount, _ := releaseAmount(OrderUpdate{Status: "CANCELLED", Symbol: "BTC-USD", Side: "BUY", Price: 1000000, Quantity: 50000, FilledQty: 20000})
	if err := releaseOrder(ctx, db, buyOrder, amount); err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t, db, buyer, "USD"); got != (balance{"30000000000", "20000000000"}) {
		t.Fatalf("iptal sonrasi %+v", got)
	}

	trade := TradeEvent{
		ID: "t2-" + run, BuyOrderID: buyOrder, SellOrderID: "legacy-" + run, BuyerID: buyer, SellerID: seller,
		Symbol: "BTC-USD", Price: 1000000, BuyLimit: 1000000, Quantity: 20000, Timestamp: time.Now(),
	}
	if _, err := settleTrade(ctx, db, trade); err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t, db, buyer, "USD"); got != (balance{"30000000000", "0"}) {
		t.Errorf("islem sonrasi alici USD %+v", got)
	}
	if n := lockCount(t, db, buyOrder); n != 0 {
		t.Errorf("kilit satiri silinmeliydi")
	}
	// saticinin emri icin kilit yoktu (muaf hesap); yine de odemesini alir
	if got := balanceOf(t, db, seller, "USD"); got != (balance{"20000000000", "0"}) {
		t.Errorf("satici USD %+v", got)
	}
}
