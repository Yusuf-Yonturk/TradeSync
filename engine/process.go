package main

import (
	"time"

	"tradesync/engine/orderbook"
)

const typeCancel = "CANCEL"

// Bir emrin son durumu; processor bundan kullanicinin acik emir listesini tutar.
type OrderUpdate struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side,omitempty"`
	Price     int64     `json:"price,omitempty"`
	Quantity  int64     `json:"quantity,omitempty"`
	FilledQty int64     `json:"filled_quantity"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type result struct {
	trades  []orderbook.Trade
	updates []OrderUpdate
}

func updateFrom(o *orderbook.Order) OrderUpdate {
	return OrderUpdate{
		ID:        o.ID,
		UserID:    o.UserID,
		Symbol:    o.Symbol,
		Side:      string(o.Side),
		Price:     o.Price,
		Quantity:  o.Quantity,
		FilledQty: o.FilledQty,
		Status:    string(o.Status),
		Timestamp: o.Timestamp,
	}
}

// Gelen mesaji deftere uygular. Kafka'ya dokunmaz; yan etkiler cagirana kalir.
func process(book *orderbook.OrderBook, in IncomingOrder) result {
	if in.Type == typeCancel {
		return processCancel(book, in)
	}

	order := toOrder(in)
	trades := book.AddOrder(order)

	updates := []OrderUpdate{updateFrom(order)}
	for _, tr := range trades {
		// karsi taraf: gelen emir alis ise satis emri, degilse alis emri
		id, userID := tr.SellOrderID, tr.SellerID
		if order.Side == orderbook.Sell {
			id, userID = tr.BuyOrderID, tr.BuyerID
		}
		if resting := book.Get(id); resting != nil {
			updates = append(updates, updateFrom(resting))
			continue
		}
		// defterden dustuyse tamamen dolmustur
		updates = append(updates, OrderUpdate{
			ID:        id,
			UserID:    userID,
			Symbol:    tr.Symbol,
			Status:    string(orderbook.Filled),
			Timestamp: tr.Timestamp,
		})
	}
	return result{trades: trades, updates: updates}
}

func processCancel(book *orderbook.OrderBook, in IncomingOrder) result {
	if order := book.CancelByID(in.ID, in.UserID); order != nil {
		return result{updates: []OrderUpdate{updateFrom(order)}}
	}
	// Baskasinin emrine dokunma.
	if other := book.Get(in.ID); other != nil {
		return result{}
	}
	// Emir defterde yok (dolmus ya da engine yeniden baslamis); listede kalmasin diye yine de bildir.
	return result{updates: []OrderUpdate{{
		ID:        in.ID,
		UserID:    in.UserID,
		Symbol:    in.Symbol,
		Status:    string(orderbook.Cancelled),
		Timestamp: in.Timestamp,
	}}}
}
