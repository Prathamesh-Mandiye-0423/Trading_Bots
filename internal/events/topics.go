package events

import (
	"encoding/json"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
)

// Topic names — single source of truth, never hardcode these elsewhere
const (
	TopicTradeExecuted = "trade.executed"
	TopicOrderUpdated  = "order.updated"
	TopicMarketTicker  = "market.ticker"
)

// ---- Event types ----

// TradeEvent is published to trade.executed on every fill
type TradeEvent struct {
	ID          string          `json:"id"`
	Symbol      string          `json:"symbol"`
	BuyOrderID  string          `json:"buy_order_id"`
	SellOrderID string          `json:"sell_order_id"`
	BuyBotID    string          `json:"buy_bot_id"`
	SellBotID   string          `json:"sell_bot_id"`
	Price       decimal.Decimal `json:"price"`
	Quantity    decimal.Decimal `json:"quantity"`
	Notional    decimal.Decimal `json:"notional"`
	ExecutedAt  time.Time       `json:"executed_at"`
}

// OrderEvent is published to order.updated on every status change
type OrderEvent struct {
	ID        string             `json:"id"`
	BotID     string             `json:"bot_id"`
	Symbol    string             `json:"symbol"`
	Side      models.Side        `json:"side"`
	Type      models.OrderType   `json:"type"`
	Price     decimal.Decimal    `json:"price"`
	Quantity  decimal.Decimal    `json:"quantity"`
	Remaining decimal.Decimal    `json:"remaining"`
	Status    models.OrderStatus `json:"status"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// TickerEvent is published to market.ticker after each trade
type TickerEvent struct {
	Symbol    string          `json:"symbol"`
	BidPrice  decimal.Decimal `json:"bid_price"`
	AskPrice  decimal.Decimal `json:"ask_price"`
	LastPrice decimal.Decimal `json:"last_price"`
	LastQty   decimal.Decimal `json:"last_qty"`
	Spread    decimal.Decimal `json:"spread"`
	Timestamp time.Time       `json:"timestamp"`
}

// ---- Converters from domain models to events ----

func TradeEventFromModel(t *models.Trade) TradeEvent {
	return TradeEvent{
		ID:          t.ID,
		Symbol:      t.Symbol,
		BuyOrderID:  t.BuyOrderID,
		SellOrderID: t.SellOrderID,
		BuyBotID:    t.BuyBotID,
		SellBotID:   t.SellBotID,
		Price:       t.Price,
		Quantity:    t.Quantity,
		Notional:    t.Notional,
		ExecutedAt:  t.ExecutedAt,
	}
}

func OrderEventFromModel(o *models.Order) OrderEvent {
	return OrderEvent{
		ID:        o.ID,
		BotID:     o.BotID,
		Symbol:    o.Symbol,
		Side:      o.Side,
		Type:      o.Type,
		Price:     o.Price,
		Quantity:  o.Quantity,
		Remaining: o.Remaining,
		Status:    o.Status,
		UpdatedAt: o.UpdatedAt,
	}
}

// ---- JSON helpers ----

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func UnmarshalTrade(data []byte) (TradeEvent, error) {
	var e TradeEvent
	return e, json.Unmarshal(data, &e)
}

func UnmarshalOrder(data []byte) (OrderEvent, error) {
	var e OrderEvent
	return e, json.Unmarshal(data, &e)
}

func UnmarshalTicker(data []byte) (TickerEvent, error) {
	var e TickerEvent
	return e, json.Unmarshal(data, &e)
}
