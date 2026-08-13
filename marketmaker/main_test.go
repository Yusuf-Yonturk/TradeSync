package main

import (
	"testing"
	"time"
)

func TestSplitStale(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	open := []placedOrder{
		{"a", now.Add(-5 * time.Minute)},
		{"b", now.Add(-2 * time.Minute)},
		{"c", now.Add(-time.Minute)},
		{"d", now},
	}
	stale, fresh := splitStale(open, now, 2*time.Minute)
	if len(stale) != 2 || stale[0].id != "a" || stale[1].id != "b" {
		t.Fatalf("eski emirler: %+v", stale)
	}
	if len(fresh) != 2 || fresh[0].id != "c" {
		t.Fatalf("yeni emirler: %+v", fresh)
	}
	if stale, fresh := splitStale(nil, now, time.Minute); len(stale)+len(fresh) != 0 {
		t.Fatal("bos liste bos kalmali")
	}
}
