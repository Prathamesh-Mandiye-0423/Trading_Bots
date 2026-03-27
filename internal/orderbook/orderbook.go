package orderbook

// Improve TC using RB Tree
import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
)

type OrderBook struct {
	Symbol    string
	mu        sync.RWMutex
	bids      map[int64]*PriceLevel
	asks      map[int64]*PriceLevel
	orderMap  map[string]*models.Order
	bidPrices []decimal.Decimal
	askPrices []decimal.Decimal
}

func NewOrderBook(symbol string) *OrderBook {
	return &OrderBook{
		Symbol:   symbol,
		bids:     make(map[int64]*PriceLevel),
		asks:     make(map[int64]*PriceLevel),
		orderMap: make(map[string]*models.Order),
	}
}

func (ob *OrderBook) AddOrder(order *models.Order) error {
	if order.Type != models.OrderTypeLimit {
		return fmt.Errorf("orderboook only accepts limit orders; got %s", order.Type)
	}
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if order.Side == models.SideBuy {
		ob.addToBids(order)
	} else {
		ob.addToAsks(order)
	}
	ob.orderMap[order.ID] = order
	return nil
}

// addToBids and addToTasks are defined later on
func (ob *OrderBook) CancelOrder(orderID string) (*models.Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	order, ok := ob.orderMap[orderID]
	if !ok {
		return nil, fmt.Errorf("order with ID %s not found", orderID)
	}
	if order.Status != models.OrderStatusOpen && order.Status != models.OrderStatusPartial {
		return nil, fmt.Errorf("Oder Id: %s only open or partial orders can be cancelled; got %s", orderID, order.Status)
	}
	key := order.Price.Raw()
	if order.Side == models.SideBuy {
		if level, exists := ob.bids[key]; exists {
			level.Cancel(orderID)
			if level.IsEmpty() {
				ob.removeBidLevel(order.Price)
			}
		}
	} else {
		if level, exists := ob.asks[key]; exists {
			level.Cancel(orderID)
			if level.IsEmpty() {
				ob.removeAskLevel(order.Price)
			}
		}
	}
	order.Status = models.OrderStatusCancelled
	order.UpdatedAt = time.Now().UTC()
	delete(ob.orderMap, orderID)
	return order, nil
}
func (ob *OrderBook) BestBidPrice() decimal.Decimal {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	if len(ob.bidPrices) == 0 {
		return decimal.FromInt(0)
	}
	return ob.bidPrices[0]
}

func (ob *OrderBook) BestAskPrice() decimal.Decimal {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	if len(ob.askPrices) == 0 {
		return decimal.FromInt(0)
	}
	return ob.askPrices[0]
}

func (ob *OrderBook) Spread() decimal.Decimal {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	if len(ob.bidPrices) == 0 || len(ob.askPrices) == 0 {
		return decimal.FromInt(0)
	}
	return ob.askPrices[0].Sub(ob.bidPrices[0])
}

func (ob *OrderBook) Snapshot(depth int) models.OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	bidPrices := ob.bidPrices
	askPrices := ob.askPrices
	if depth > 0 {
		if len(bidPrices) > depth {
			bidPrices = bidPrices[:depth]
		}
		if len(askPrices) > depth {
			askPrices = askPrices[:depth]
		}
	}
	bids := make([]models.PriceLevelSnapshot, 0, len(bidPrices))
	for _, p := range bidPrices {
		bids = append(bids, ob.bids[p.Raw()].Snapshot())
	}
	asks := make([]models.PriceLevelSnapshot, 0, len(askPrices))
	for _, p := range askPrices {
		asks = append(asks, ob.asks[p.Raw()].Snapshot())
	}
	var spread decimal.Decimal
	if len(ob.bidPrices) > 0 && len(ob.askPrices) > 0 {
		spread = ob.askPrices[0].Sub(ob.bidPrices[0])
	}
	return models.OrderBookSnapshot{
		Symbol:    ob.Symbol,
		Bids:      bids,
		Asks:      asks,
		Spread:    spread,
		Timestamp: time.Now().UTC(),
	}
}

func (ob *OrderBook) GetOrder(orderID string) (*models.Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	o, ok := ob.orderMap[orderID]
	return o, ok
}

func (ob *OrderBook) addToBids(order *models.Order) {
	key := order.Price.Raw()
	level, exists := ob.bids[key]
	if !exists {
		level = newPriceLevel(order.Price)
		ob.bids[key] = level
		ob.insertBidPrice(order.Price)
	}
	level.Add(order)
}

func (ob *OrderBook) addToAsks(order *models.Order) {
	key := order.Price.Raw()
	level, exists := ob.asks[key]
	if !exists {
		level = newPriceLevel(order.Price)
		ob.asks[key] = level
		ob.insertAskPrice(order.Price)
	}
	level.Add(order)
}

func (ob *OrderBook) insertBidPrice(price decimal.Decimal) {
	i := sort.Search(len(ob.bidPrices), func(i int) bool {
		return ob.bidPrices[i].LessEq(price)
	})
	ob.bidPrices = append(ob.bidPrices, decimal.FromInt(0))
	copy(ob.bidPrices[i+1:], ob.bidPrices[i:])
	ob.bidPrices[i] = price
}

func (ob *OrderBook) insertAskPrice(price decimal.Decimal) {
	i := sort.Search(len(ob.askPrices), func(i int) bool {
		return ob.askPrices[i].GreaterEq(price)
	})
	ob.askPrices = append(ob.askPrices, decimal.FromInt(0))
	copy(ob.askPrices[i+1:], ob.askPrices[i:])
	ob.askPrices[i] = price
}

func (ob *OrderBook) removeBidLevel(price decimal.Decimal) {
	delete(ob.bids, price.Raw())
	ob.bidPrices = removeDecimal(ob.bidPrices, price)
}

func (ob *OrderBook) removeAskLevel(price decimal.Decimal) {
	delete(ob.asks, price.Raw())
	ob.askPrices = removeDecimal(ob.askPrices, price)
}

func removeDecimal(s []decimal.Decimal, v decimal.Decimal) []decimal.Decimal {
	for i, d := range s {
		if d.Equal(v) {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func (ob *OrderBook) BestBidLocked() *PriceLevel {
	if len(ob.bidPrices) == 0 {
		return nil
	}
	return ob.bids[ob.bidPrices[0].Raw()]
}

func (ob *OrderBook) BestAskLocked() *PriceLevel {
	if len(ob.askPrices) == 0 {
		return nil
	}
	return ob.asks[ob.askPrices[0].Raw()]
}

func (ob *OrderBook) RemoveBidLevelLocked(price decimal.Decimal) { ob.removeBidLevel(price) }
func (ob *OrderBook) RemoveAskLevelLocked(price decimal.Decimal) { ob.removeAskLevel(price) }
func (ob *OrderBook) RemoveOrderFromMapLocked(orderID string)    { delete(ob.orderMap, orderID) }
func (ob *OrderBook) LockWrite()                                 { ob.mu.Lock() }
func (ob *OrderBook) UnlockWrite()                               { ob.mu.Unlock() }

func (ob *OrderBook) GetOrCreateBidLevelLocked(price decimal.Decimal) *PriceLevel {
	key := price.Raw()
	level, exists := ob.bids[key]
	if !exists {
		level = newPriceLevel(price)
		ob.bids[key] = level
		ob.insertBidPrice(price)
	}
	return level
}

func (ob *OrderBook) GetOrCreateAskLevelLocked(price decimal.Decimal) *PriceLevel {
	key := price.Raw()
	level, exists := ob.asks[key]
	if !exists {
		level = newPriceLevel(price)
		ob.asks[key] = level
		ob.insertAskPrice(price)
	}
	return level
}
