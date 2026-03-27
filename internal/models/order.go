package models

import (
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/google/uuid"
)

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

type OrderStatus string

const (
	OrderStatusOpen      OrderStatus = "OPEN"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusPartial   OrderStatus = "PARTIAL"
)

type Order struct {
	ID        string          `json:"id"`
	BotID     string          `json:"bot_id"`
	Symbol    string          `json:"symbol"`
	Side      Side            `json:"side"`
	Type      OrderType       `json:"type"`
	Price     decimal.Decimal `json:"price"`
	Quantity  decimal.Decimal `json:"quantity"`
	Remaining decimal.Decimal `json:"remaining"`
	Status    OrderStatus     `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func NewLimitOrder(botID, symbol string, side Side, price, quantity decimal.Decimal) *Order {
	now := time.Now().UTC()
	return &Order{
		ID:        uuid.New().String(),
		BotID:     botID,
		Symbol:    symbol,
		Side:      side,
		Type:      OrderTypeLimit,
		Price:     price,
		Quantity:  quantity,
		Remaining: quantity,
		Status:    OrderStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewMarketOrder(botID, symbol string, side Side, quantity decimal.Decimal) *Order {
	now := time.Now().UTC()
	return &Order{
		ID:        uuid.New().String(),
		BotID:     botID,
		Symbol:    symbol,
		Side:      side,
		Type:      OrderTypeMarket,
		Price:     decimal.FromInt(0),
		Quantity:  quantity,
		Remaining: quantity,
		Status:    OrderStatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Trade struct {
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

func NewTrade(symbol, BuyOrderID, sellOrderID, buyBotID, sellBotID string, price, quantity decimal.Decimal) *Trade {
	return &Trade{
		ID:          uuid.New().String(),
		Symbol:      symbol,
		BuyOrderID:  BuyOrderID,
		SellOrderID: sellOrderID,
		BuyBotID:    buyBotID,
		SellBotID:   sellBotID,
		Price:       price,
		Quantity:    quantity,
		Notional:    price.Mul(quantity),
		ExecutedAt:  time.Now().UTC(),
	}
}

type PriceLevelSnapshot struct {
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Orders   int             `json:"orders"`
}

type OrderBookSnapshot struct {
	Symbol    string               `json:"symbol"`
	Bids      []PriceLevelSnapshot `json:"bids"`
	Asks      []PriceLevelSnapshot `json:"asks"`
	Spread    decimal.Decimal      `json:"spread"`
	Timestamp time.Time            `json:"timestamp"`
}
