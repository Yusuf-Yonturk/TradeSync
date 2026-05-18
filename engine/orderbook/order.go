package orderbook

import "time"

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type OrderStatus string

const (
	Open            OrderStatus = "OPEN"
	PartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	Filled          OrderStatus = "FILLED"
	Cancelled       OrderStatus = "CANCELLED"
)

type Order struct {
	ID        string
	UserID    string
	Side      Side
	Price     int64
	Quantity  int64
	FilledQty int64
	Status    OrderStatus
	Timestamp time.Time

	HeapIndex int
}

func updateOrderStatus(order *Order) {
	switch {
	case order.FilledQty == 0:
		order.Status = Open
	case order.FilledQty < order.Quantity:
		order.Status = PartiallyFilled
	default:
		order.Status = Filled
	}
}

func (o *Order) RemainingQty() int64 {
	return o.Quantity - o.FilledQty
}

func (o *Order) IsFilled() bool {
	return o.FilledQty >= o.Quantity
}
