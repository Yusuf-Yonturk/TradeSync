package orderbook

import "container/heap"

type BuyHeap[] *Order

func (h BuyHeap) Len() int {return len(h)}

func (h BuyHeap) Less(i, j int) bool {
	if h[i].Price != h[j].Price {
		return h[i].Price > h[j].Price
	}
	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h BuyHeap) Swap(i, j int){
	h[i], h[j] =h[j], h[i]
	h[i].HeapIndex=i
	h[j].HeapIndex=j
}

func (h *BuyHeap) Push(x any){
	order:=x.(*Order)
	order.HeapIndex=len(*h)
	*h = append(*h,order)
}

func (h *BuyHeap) Pop() any {
	old:=*h
	n:=len(old)
	order:=old[n-1]
	old[n-1]=nil
	order.HeapIndex=-1
	*h=old[:n-1]
	return order
}

type SellHeap []*Order

func (h SellHeap) Len() int {return len(h)}

func (h SellHeap) Less(i, j int) bool {
	if h[i].Price != h[j].Price {
		return h[i].Price < h[j].Price
	}
	return h[i].Timestamp.Before(h[j].Timestamp)
}

func (h SellHeap) Swap(i,j int){
	h[i],h[j] = h[j],h[i]
	h[i].HeapIndex=i
	h[j].HeapIndex=j
}

func (h *SellHeap) Push(x any){
	order:=x.(*Order)
	order.HeapIndex=len(*h)
	*h = append(*h, order)
}

func(h* SellHeap) Pop() any{
	old:=*h
	n:=len(old)
	order:=old[n-1]
	old[n-1]=nil
	order.HeapIndex=-1
	*h=old[:n-1]
	return order
}

func RemoveFromBuyHeap(h *BuyHeap, order *Order) {
	heap.Remove(h, order.HeapIndex)
}

func RemoveFromSellHeap(h *SellHeap, order *Order){
	heap.Remove(h, order.HeapIndex)
}