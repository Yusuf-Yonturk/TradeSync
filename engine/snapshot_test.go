package main

import (
	"math/rand"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"tradesync/engine/orderbook"
)

func randomHistory(rng *rand.Rand, n int) []IncomingOrder {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var ids []string
	history := make([]IncomingOrder, 0, n)
	for i := 0; i < n; i++ {
		user := "u" + strconv.Itoa(rng.Intn(4))
		if len(ids) > 0 && rng.Intn(5) == 0 {
			history = append(history, IncomingOrder{Type: typeCancel, ID: ids[rng.Intn(len(ids))], UserID: user, Symbol: "BTC-USD"})
			continue
		}
		side := "BUY"
		if rng.Intn(2) == 0 {
			side = "SELL"
		}
		id := "o" + strconv.Itoa(i)
		ids = append(ids, id)
		history = append(history, IncomingOrder{
			ID: id, UserID: user, Symbol: "BTC-USD", Side: side,
			Price: int64(95 + rng.Intn(10)), Quantity: int64(1 + rng.Intn(9)),
			// ayni zaman damgali emirler de olsun
			Timestamp: base.Add(time.Duration(i/3) * time.Second),
			STP:       true,
		})
	}
	return history
}

type tradeKey struct {
	buy, sell    string
	price, qty   int64
	buyLimit     int64
	buyer, sell2 string
}

func keys(trades []orderbook.Trade) []tradeKey {
	out := make([]tradeKey, len(trades))
	for i, tr := range trades {
		out[i] = tradeKey{tr.BuyOrderID, tr.SellOrderID, tr.Price, tr.Quantity, tr.BuyLimit, tr.BuyerID, tr.SellerID}
	}
	return out
}

// Goruntuden kurulan defter, hic durmamis defterle ayni islemleri uretmeli.
func TestSnapshotRestoreMatchesContinuousRun(t *testing.T) {
	for seed := int64(1); seed <= 30; seed++ {
		rng := rand.New(rand.NewSource(seed))
		history := randomHistory(rng, 300)
		cut := 100 + rng.Intn(100)

		continuous := map[string]*orderbook.OrderBook{}
		for _, in := range history[:cut] {
			process(getOrCreateBook(continuous, in.Symbol), in)
		}

		path := filepath.Join(t.TempDir(), "sub", "snapshot.json")
		if err := saveSnapshot(path, takeSnapshot(continuous, map[int32]int64{0: int64(cut)})); err != nil {
			t.Fatal(err)
		}
		snap, err := loadSnapshot(path)
		if err != nil || snap == nil {
			t.Fatalf("goruntu okunamadi: %v", err)
		}
		if snap.Offsets[0] != int64(cut) {
			t.Fatalf("offset kayboldu: %v", snap.Offsets)
		}
		restored := restoreBooks(*snap)

		for i, in := range history[cut:] {
			want := process(getOrCreateBook(continuous, in.Symbol), in)
			got := process(getOrCreateBook(restored, in.Symbol), in)
			if !reflect.DeepEqual(keys(want.trades), keys(got.trades)) {
				t.Fatalf("seed=%d adim=%d islemler farkli:\n%+v\n%+v", seed, cut+i, want.trades, got.trades)
			}
			if len(want.updates) != len(got.updates) {
				t.Fatalf("seed=%d adim=%d guncelleme sayisi farkli", seed, cut+i)
			}
			for j := range want.updates {
				w, g := want.updates[j], got.updates[j]
				if w.ID != g.ID || w.Status != g.Status || w.FilledQty != g.FilledQty {
					t.Fatalf("seed=%d adim=%d guncelleme farkli: %+v / %+v", seed, cut+i, w, g)
				}
			}
		}
	}
}

func TestLoadSnapshotMissingFile(t *testing.T) {
	snap, err := loadSnapshot(filepath.Join(t.TempDir(), "yok.json"))
	if err != nil || snap != nil {
		t.Fatalf("dosya yokken (nil, nil) bekleniyordu: %v %v", snap, err)
	}
}

func TestSnapshotUsable(t *testing.T) {
	committed := map[int32]int64{0: 10}
	cases := []struct {
		name string
		snap snapshot
		want bool
	}{
		{"geride", snapshot{Version: snapshotVersion, Offsets: map[int32]int64{0: 7}}, true},
		{"esit", snapshot{Version: snapshotVersion, Offsets: map[int32]int64{0: 10}}, true},
		{"ileride", snapshot{Version: snapshotVersion, Offsets: map[int32]int64{0: 11}}, false},
		{"bilinmeyen partition ileride", snapshot{Version: snapshotVersion, Offsets: map[int32]int64{1: 1}}, false},
		{"eski surum", snapshot{Version: 0, Offsets: map[int32]int64{0: 7}}, false},
	}
	for _, c := range cases {
		if got := snapshotUsable(c.snap, committed); got != c.want {
			t.Errorf("%s: %v bekleniyordu, gelen %v", c.name, c.want, got)
		}
	}
}

func TestTradeIDIsStable(t *testing.T) {
	trades := []orderbook.Trade{{Symbol: "BTC-USD"}, {Symbol: "BTC-USD"}}
	a, b := toOutgoingTrades(trades, 2, 41), toOutgoingTrades(trades, 2, 41)
	if a[0].ID != b[0].ID || a[0].ID == a[1].ID || a[0].ID != "2-41-0" {
		t.Fatalf("beklenmeyen id'ler: %q %q %q", a[0].ID, a[1].ID, b[0].ID)
	}
}
