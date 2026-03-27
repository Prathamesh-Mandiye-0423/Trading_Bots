package matching

import (
	"fmt"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/orderbook"
)

type MatchResult struct {
	Order        *models.Order
	Trades       []*models.Trade
	RestingOrder *models.Order
}

type Engine struct {
	symbol    string
	book      *orderbook.OrderBook
	tradeChan chan<- *models.Trade
}

func NewEngine(symbol string, tradeChan chan<- *models.Trade) *Engine {
	return &Engine{
		symbol:    symbol,
		book:      orderbook.NewOrderBook(symbol),
		tradeChan: tradeChan,
	}
}

func (e *Engine) Book() *orderbook.OrderBook { return e.book }

func (e *Engine) Submit(order *models.Order) (*MatchResult, error) {
	if order.Symbol != e.symbol {
		return nil, fmt.Errorf("engine for %s received order for %s", e.symbol, order.Symbol)
	}
	if !order.Quantity.IsPositive() {
		return nil, fmt.Errorf("order quantity must be positive, got %s", order.Quantity.String())
	}
	if order.Type == models.OrderTypeLimit && !order.Price.IsPositive() {
		return nil, fmt.Errorf("limit order price must be positive, got %s", order.Price.String())
	}

	e.book.LockWrite()
	defer e.book.UnlockWrite()

	result := &MatchResult{
		Order:  order,
		Trades: make([]*models.Trade, 0),
	}

	switch order.Side {
	case models.SideBuy:
		e.matchBuy(order, result)
	case models.SideSell:
		e.matchSell(order, result)
	default:
		return nil, fmt.Errorf("unknown order side: %s", order.Side)
	}

	now := time.Now().UTC()
	if order.Remaining.IsZero() {
		order.Status = models.OrderStatusFilled
		order.UpdatedAt = now
	} else if order.Type == models.OrderTypeLimit {
		if order.Side == models.SideBuy {
			e.book.GetOrCreateBidLevelLocked(order.Price).Add(order)
		} else {
			e.book.GetOrCreateAskLevelLocked(order.Price).Add(order)
		}
		result.RestingOrder = order
	} else {
		order.Status = models.OrderStatusCancelled
		order.UpdatedAt = now
	}

	for _, trade := range result.Trades {
		select {
		case e.tradeChan <- trade:
		default:
		}
	}
	return result, nil
}

func (e *Engine) Cancel(orderID string) (*models.Order, error) {
	return e.book.CancelOrder(orderID)
}

func (e *Engine) matchBuy(order *models.Order, result *MatchResult) {
	for order.Remaining.IsPositive() {
		bestAsk := e.book.BestAskLocked()
		if bestAsk == nil {
			break
		}
		if order.Type == models.OrderTypeLimit && order.Price.Less(bestAsk.Price) {
			break
		}
		e.fillAgainstLevel(order, bestAsk, result)
		if bestAsk.IsEmpty() {
			e.book.RemoveAskLevelLocked(bestAsk.Price)
		}
	}
}

func (e *Engine) matchSell(order *models.Order, result *MatchResult) {
	for order.Remaining.IsPositive() {
		bestBid := e.book.BestBidLocked()
		if bestBid == nil {
			break
		}
		if order.Type == models.OrderTypeLimit && order.Price.Greater(bestBid.Price) {
			break
		}
		e.fillAgainstLevel(order, bestBid, result)
		if bestBid.IsEmpty() {
			e.book.RemoveBidLevelLocked(bestBid.Price)
		}
	}
}

func (e *Engine) fillAgainstLevel(incoming *models.Order, level *orderbook.PriceLevel, result *MatchResult) {
	for incoming.Remaining.IsPositive() {
		resting := level.Front()
		if resting == nil {
			break
		}
		if resting.BotID == incoming.BotID {
			break
		}

		fillQty := decimal.Min(incoming.Remaining, resting.Remaining)
		fillPrice := resting.Price

		var buyOrderID, sellOrderID, buyBotID, sellBotID string
		if incoming.Side == models.SideBuy {
			buyOrderID, buyBotID = incoming.ID, incoming.BotID
			sellOrderID, sellBotID = resting.ID, resting.BotID
		} else {
			sellOrderID, sellBotID = incoming.ID, incoming.BotID
			buyOrderID, buyBotID = resting.ID, resting.BotID
		}

		trade := models.NewTrade(e.symbol, buyOrderID, sellOrderID, buyBotID, sellBotID, fillPrice, fillQty)
		result.Trades = append(result.Trades, trade)

		now := time.Now().UTC()
		incoming.Remaining = incoming.Remaining.Sub(fillQty)
		incoming.UpdatedAt = now
		resting.Remaining = resting.Remaining.Sub(fillQty)
		resting.UpdatedAt = now
		level.ReduceQuantity(fillQty)

		if resting.Remaining.IsZero() {
			resting.Status = models.OrderStatusFilled
			level.PopFront()
			e.book.RemoveOrderFromMapLocked(resting.ID)
		} else {
			resting.Status = models.OrderStatusPartial
		}
		if incoming.Remaining.IsPositive() {
			incoming.Status = models.OrderStatusPartial
		}
	}
}
