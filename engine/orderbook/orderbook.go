package orderbook

import (
	"container/heap"
	"sort"
	"time"
)

type OrderBook struct {
	Buys  *BuyHeap
	Sells *SellHeap
}

type Trade struct {
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
	Symbol      string    `json:"symbol"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
	Timestamp   time.Time `json:"timestamp"`
}

type DepthLevel struct {
	Price    int64 `json:"price"`
	Quantity int64 `json:"quantity"`
}

type OrderBookSnapshot struct {
	Symbol string       `json:"symbol"`
	Bids   []DepthLevel `json:"bids"`
	Asks   []DepthLevel `json:"asks"`
}

func minQty(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func executeTrade(buy, sell *Order, price int64) Trade {
	qty := minQty(buy.RemainingQty(), sell.RemainingQty())
	buy.FilledQty += qty
	sell.FilledQty += qty

	updateOrderStatus(buy)
	updateOrderStatus(sell)

	return Trade{
		BuyOrderID:  buy.ID,
		SellOrderID: sell.ID,
		BuyerID:     buy.UserID,
		SellerID:    sell.UserID,
		Symbol:      buy.Symbol,
		Price:       price,
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

func aggregate(orders []*Order, descending bool, maxDepth int) []DepthLevel {
	m := make(map[int64]int64)
	for _, o := range orders {
		m[o.Price] += o.RemainingQty()
	}

	levels := make([]DepthLevel, 0, len(m))
	for p, q := range m {
		levels = append(levels, DepthLevel{Price: p, Quantity: q})
	}

	sort.Slice(levels, func(i, j int) bool {
		if descending {
			return levels[i].Price > levels[j].Price
		}
		return levels[i].Price < levels[j].Price
	})

	if len(levels) > maxDepth {
		levels = levels[:maxDepth]
	}
	return levels
}

func (ob *OrderBook) Snapshot(symbol string, maxDepth int) OrderBookSnapshot {
	return OrderBookSnapshot{
		Symbol: symbol,
		Bids:   aggregate(*ob.Buys, true, maxDepth),
		Asks:   aggregate(*ob.Sells, false, maxDepth),
	}
}

func CanMatch(buy, sell *Order) bool {
	if buy == nil || sell == nil {
		return false
	}
	return buy.Price >= sell.Price
}

// Emri iptal et
func (ob *OrderBook) CancelOrder(order *Order) bool {
	if order.Status == Filled || order.Status == Cancelled {
		return false
	}
	if order.Side == Buy {
		if order.HeapIndex < 0 || order.HeapIndex >= ob.Buys.Len() {
			return false
		}
		RemoveFromBuyHeap(ob.Buys, order)
	} else {
		if order.HeapIndex < 0 || order.HeapIndex >= ob.Sells.Len() {
			return false
		}
		RemoveFromSellHeap(ob.Sells, order)
	}
	order.Status = Cancelled
	return true
}

// Emri book'a ekle, eşleşen trade'leri döndür
func (ob *OrderBook) AddOrder(order *Order) []Trade {
	var trades []Trade

	if order.Side == Buy {
		// Eşleşebildiği sürece dön
		for {
			best := ob.BestSell()
			if !CanMatch(order, best) {
				break
			}
			trade := executeTrade(order, best, best.Price)
			trades = append(trades, trade)

			if best.IsFilled() {
				heap.Pop(ob.Sells)
			}
			if order.IsFilled() {
				return trades
			}
		}
		heap.Push(ob.Buys, order)
		return trades
	}

	// Sell tarafı
	for {
		best := ob.BestBuy()
		if !CanMatch(best, order) {
			break
		}
		trade := executeTrade(best, order, best.Price)
		trades = append(trades, trade)

		if best.IsFilled() {
			heap.Pop(ob.Buys)
		}
		if order.IsFilled() {
			return trades
		}
	}
	heap.Push(ob.Sells, order)
	return trades
}
