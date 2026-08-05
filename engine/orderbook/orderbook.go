package orderbook

import (
	"container/heap"
	"sort"
	"time"
)

type OrderBook struct {
	Buys  *BuyHeap
	Sells *SellHeap

	// defterde bekleyen emirler, id ile iptal ve durum sorgusu icin
	resting map[string]*Order
}

type Trade struct {
	BuyOrderID  string `json:"buy_order_id"`
	SellOrderID string `json:"sell_order_id"`
	BuyerID     string `json:"buyer_id"`
	SellerID    string `json:"seller_id"`
	Symbol      string `json:"symbol"`
	Price       int64  `json:"price"`
	// Alis emrinin limit fiyati; islem daha ucuza gerceklesirse fark aliciya iade edilir.
	BuyLimit  int64     `json:"buy_limit"`
	Quantity  int64     `json:"quantity"`
	Timestamp time.Time `json:"timestamp"`
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
		BuyLimit:    buy.Price,
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
		Buys:    buys,
		Sells:   sells,
		resting: make(map[string]*Order),
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
	delete(ob.resting, order.ID)
	return true
}

// Defterde bekleyen emri dondurur; dolmus ya da iptal edilmisse nil.
func (ob *OrderBook) Get(id string) *Order {
	return ob.resting[id]
}

// Emri id ile iptal eder. Emir yoksa ya da baska kullaniciya aitse nil doner.
func (ob *OrderBook) CancelByID(id, userID string) *Order {
	order := ob.resting[id]
	if order == nil || order.UserID != userID {
		return nil
	}
	if !ob.CancelOrder(order) {
		return nil
	}
	return order
}

// Defterde bekleyen butun emirler; sira garanti edilmez.
func (ob *OrderBook) Orders() []*Order {
	out := make([]*Order, 0, len(ob.resting))
	for _, o := range ob.resting {
		out = append(out, o)
	}
	return out
}

// Anlik goruntuden gelen emri eslestirmeden deftere koyar.
func (ob *OrderBook) Restore(order *Order) {
	if order.Side == Buy {
		heap.Push(ob.Buys, order)
	} else {
		heap.Push(ob.Sells, order)
	}
	ob.resting[order.ID] = order
}

// Kullanici kendi bekleyen emriyle eslesecekse gelen emrin kalani iptal edilir.
func selfTrade(incoming, resting *Order) bool {
	return incoming.PreventSelfTrade && incoming.UserID != "" && incoming.UserID == resting.UserID
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
			if selfTrade(order, best) {
				order.Status = Cancelled
				return trades
			}
			trade := executeTrade(order, best, best.Price)
			trades = append(trades, trade)

			if best.IsFilled() {
				heap.Pop(ob.Sells)
				delete(ob.resting, best.ID)
			}
			if order.IsFilled() {
				return trades
			}
		}
		heap.Push(ob.Buys, order)
		ob.resting[order.ID] = order
		return trades
	}

	// Sell tarafı
	for {
		best := ob.BestBuy()
		if !CanMatch(best, order) {
			break
		}
		if selfTrade(order, best) {
			order.Status = Cancelled
			return trades
		}
		trade := executeTrade(best, order, best.Price)
		trades = append(trades, trade)

		if best.IsFilled() {
			heap.Pop(ob.Buys)
			delete(ob.resting, best.ID)
		}
		if order.IsFilled() {
			return trades
		}
	}
	heap.Push(ob.Sells, order)
	ob.resting[order.ID] = order
	return trades
}
