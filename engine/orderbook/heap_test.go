package orderbook

import (
	"container/heap"
	"testing"
	"time"
)

func TestBuyHeapOrder(t *testing.T) {
	h := &BuyHeap{}
	heap.Init(h)
	now := time.Now()

	heap.Push(h, &Order{ID: "b1", Price: 100, Timestamp: now})
	heap.Push(h, &Order{ID: "b2", Price: 110, Timestamp: now.Add(time.Second)})
	heap.Push(h, &Order{ID: "b3", Price: 105, Timestamp: now.Add(2 * time.Second)})

	// En yüksek fiyat önce gelmeli
	top := heap.Pop(h).(*Order)
	if top.ID != "b2" {
		t.Errorf("b2 beklendi, %s geldi", top.ID)
	}

	top = heap.Pop(h).(*Order)
	if top.ID != "b3" {
		t.Errorf("b3 beklendi, %s geldi", top.ID)
	}
}

func TestSellHeapOrder(t *testing.T) {
	h := &SellHeap{}
	heap.Init(h)
	now := time.Now()

	heap.Push(h, &Order{ID: "s1", Price: 110, Timestamp: now})
	heap.Push(h, &Order{ID: "s2", Price: 100, Timestamp: now.Add(time.Second)})
	heap.Push(h, &Order{ID: "s3", Price: 105, Timestamp: now.Add(2 * time.Second)})

	// En düşük fiyat önce gelmeli
	top := heap.Pop(h).(*Order)
	if top.ID != "s2" {
		t.Errorf("s2 beklendi, %s geldi", top.ID)
	}

	top = heap.Pop(h).(*Order)
	if top.ID != "s3" {
		t.Errorf("s3 beklendi, %s geldi", top.ID)
	}
}

func TestBuyHeapTimePriority(t *testing.T) {
	h := &BuyHeap{}
	heap.Init(h)
	now := time.Now()

	// Aynı fiyat - önce gelen önce çıkmalı
	heap.Push(h, &Order{ID: "b1", Price: 100, Timestamp: now.Add(2 * time.Second)})
	heap.Push(h, &Order{ID: "b2", Price: 100, Timestamp: now})
	heap.Push(h, &Order{ID: "b3", Price: 100, Timestamp: now.Add(time.Second)})

	top := heap.Pop(h).(*Order)
	if top.ID != "b2" {
		t.Errorf("b2 beklendi (en eski), %s geldi", top.ID)
	}
}

func TestRemoveFromBuyHeap(t *testing.T) {
	h := &BuyHeap{}
	heap.Init(h)
	now := time.Now()

	o1 := &Order{ID: "b1", Price: 100, Timestamp: now}
	o2 := &Order{ID: "b2", Price: 110, Timestamp: now.Add(time.Second)}
	o3 := &Order{ID: "b3", Price: 105, Timestamp: now.Add(2 * time.Second)}

	heap.Push(h, o1)
	heap.Push(h, o2)
	heap.Push(h, o3)

	// Ortadaki elemanı sil
	RemoveFromBuyHeap(h, o3)

	if h.Len() != 2 {
		t.Fatalf("2 eleman beklendi, %d var", h.Len())
	}

	// Sıra bozulmamali
	top := heap.Pop(h).(*Order)
	if top.ID != "b2" {
		t.Errorf("b2 beklendi, %s geldi", top.ID)
	}

	top = heap.Pop(h).(*Order)
	if top.ID != "b1" {
		t.Errorf("b1 beklendi, %s geldi", top.ID)
	}
}

func TestRemoveFromSellHeap(t *testing.T) {
	h := &SellHeap{}
	heap.Init(h)
	now := time.Now()

	o1 := &Order{ID: "s1", Price: 110, Timestamp: now}
	o2 := &Order{ID: "s2", Price: 100, Timestamp: now.Add(time.Second)}
	o3 := &Order{ID: "s3", Price: 105, Timestamp: now.Add(2 * time.Second)}

	heap.Push(h, o1)
	heap.Push(h, o2)
	heap.Push(h, o3)

	// En düşük fiyatlıyı sil
	RemoveFromSellHeap(h, o2)

	if h.Len() != 2 {
		t.Fatalf("2 eleman beklendi, %d var", h.Len())
	}

	// Kalan sıra doğru olmali
	top := heap.Pop(h).(*Order)
	if top.ID != "s3" {
		t.Errorf("s3 beklendi, %s geldi", top.ID)
	}

	top = heap.Pop(h).(*Order)
	if top.ID != "s1" {
		t.Errorf("s1 beklendi, %s geldi", top.ID)
	}
}
