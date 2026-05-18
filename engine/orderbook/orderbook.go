package orderbook

import (
	"container/heap"
	"time"
)

type OrderBook struct {
	Buys  *BuyHeap
	Sells *SellHeap
}

type Trade struct {
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func ExecuteTrade(buy, sell *Order) Trade {
	qty := min(buy.RemainingQty(), sell.RemainingQty())
	buy.FilledQty += qty
	sell.FilledQty += qty

	updateOrderStatus(buy)
	updateOrderStatus(sell)

	return Trade{
		BuyOrderID:  buy.ID,
		SellOrderID: sell.ID,
		Price:       sell.Price,
		Quantity:    qty,
		Timestamp:   time.Now(),
	}
}

func NewOrderBook() *OrderBook {
	buys := &BuyHeap{}
	sells := &SellHeap{}

	heap.Init(buys)
	heap.Init(sells)

	return &OrderBook{
		Buys:  buys,
		Sells: sells,
	}
}

func (ob *OrderBook) AddToBook(order *Order) {
	if order.Side == Buy {
		heap.Push(ob.Buys, order)
	} else {
		heap.Push(ob.Sells, order)
	}
}

func (ob *OrderBook) BestBuy() *Order {
	if ob.Buys.Len() == 0 {
		return nil
	}
	return (*ob.Buys)[0]
}
func (ob *OrderBook) BestSell() *Order {
	if ob.Sells.Len() == 0 {
		return nil
	}
	return (*ob.Sells)[0]
}

func CanMatch(buy, sell *Order) bool {
	if buy == nil || sell == nil {
		return false
	}
	return buy.Price >= sell.Price
}

func (ob *OrderBook) addOrder(order *Order) []Trade {
	var trades []Trade

	if order.Side == Buy {
		bestSell := ob.BestSell()
		if CanMatch(order, bestSell) {
			trade := ExecuteTrade(order, bestSell)
			trades = append(trades, trade)

			if bestSell.IsFilled() {
				heap.Pop(ob.Sells)
			}
			if !order.IsFilled() {
				heap.Push(ob.Buys, order)
			}
			return trades
		}

		heap.Push(ob.Buys, order)
		return trades
	}
	bestbuy := ob.BestBuy()
	if CanMatch(bestbuy, order) {
		trade := ExecuteTrade(bestbuy, order)
		trades := append(trades, trade)

		if bestbuy.IsFilled() {
			heap.Pop(ob.Buys)
		}
		if !order.IsFilled() {
			heap.Push(ob.Sells, order)
		}
		return trades
	}
	heap.Push(ob.Sells, order)
	return trades
}
