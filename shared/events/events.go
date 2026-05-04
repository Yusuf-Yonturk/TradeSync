package events

import "time"

type EventType string

const(
	OrderSubmitted EventType = "ORDER_SUBMITTED"
	OrderCancelled EventType = "ORDER_CANCELLED"
	TradeExecuted  EventType = "TRADE_EXECUTED"
	OrderPartialFill EventType ="ORDER_PARTIAL_FILL"
)

type Event struct{
	ID	string `json:"id"`
	Type EventType `json:"type"`
	AggregateID string `json:"aggregate_id"`
	Sequence int64 `json:"sequence"`
	Payload any `json:"payload"`
	Timestamp time.Time	`json:"timestamp"`
}