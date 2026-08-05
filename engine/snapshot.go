package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"tradesync/engine/orderbook"
)

const snapshotVersion = 1

// Defterin diske yazilan hali. Offsets, deftere uygulanmis son kaydin bir
// sonraki offsetidir; acilista topic buradan okunur, bastan degil.
type snapshot struct {
	Version int             `json:"version"`
	Offsets map[int32]int64 `json:"offsets"`
	Orders  []snapshotOrder `json:"orders"`
}

type snapshotOrder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Price     int64     `json:"price"`
	Quantity  int64     `json:"quantity"`
	FilledQty int64     `json:"filled_quantity"`
	Timestamp time.Time `json:"timestamp"`
}

func takeSnapshot(books map[string]*orderbook.OrderBook, position map[int32]int64) snapshot {
	snap := snapshot{Version: snapshotVersion, Offsets: make(map[int32]int64, len(position))}
	for p, next := range position {
		snap.Offsets[p] = next
	}
	for _, book := range books {
		for _, o := range book.Orders() {
			snap.Orders = append(snap.Orders, snapshotOrder{
				ID:        o.ID,
				UserID:    o.UserID,
				Symbol:    o.Symbol,
				Side:      string(o.Side),
				Price:     o.Price,
				Quantity:  o.Quantity,
				FilledQty: o.FilledQty,
				Timestamp: o.Timestamp,
			})
		}
	}
	return snap
}

func restoreBooks(snap snapshot) map[string]*orderbook.OrderBook {
	books := make(map[string]*orderbook.OrderBook)
	for _, so := range snap.Orders {
		status := orderbook.Open
		if so.FilledQty > 0 {
			status = orderbook.PartiallyFilled
		}
		getOrCreateBook(books, so.Symbol).Restore(&orderbook.Order{
			ID:        so.ID,
			UserID:    so.UserID,
			Symbol:    so.Symbol,
			Side:      orderbook.Side(so.Side),
			Price:     so.Price,
			Quantity:  so.Quantity,
			FilledQty: so.FilledQty,
			Status:    status,
			Timestamp: so.Timestamp,
		})
	}
	return books
}

// Goruntu, yayinlanmis sonuclarin ilerisindeyse kullanilamaz: aradaki
// kayitlarin sonuclari bir daha uretilemez.
func snapshotUsable(snap snapshot, committed map[int32]int64) bool {
	if snap.Version != snapshotVersion {
		return false
	}
	for p, next := range snap.Offsets {
		if next > committed[p] {
			return false
		}
	}
	return true
}

// Dosya yoksa (nil, nil) doner.
func loadSnapshot(path string) (*snapshot, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// Yarim yazilmis dosya kalmasin diye once gecici dosyaya yazar, sonra yerine tasir.
func saveSnapshot(path string, snap snapshot) error {
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".snapshot-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
