package matching

import (
	"fmt"
	"sync"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
)

type Registry struct {
	mu        sync.RWMutex
	engines   map[string]*Engine
	tradeChan chan *models.Trade
}

func NewRegistry(tradeChanSize int) *Registry {
	return &Registry{
		engines:   make(map[string]*Engine),
		tradeChan: make(chan *models.Trade, tradeChanSize),
	}
}

func (r *Registry) AddMarket(symbol string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.engines[symbol]; exists {
		return fmt.Errorf("market %s already exists", symbol)
	}
	r.engines[symbol] = NewEngine(symbol, r.tradeChan)
	return nil
}

func (r *Registry) Submit(order *models.Order) (*MatchResult, error) {
	r.mu.RLock()
	engine, ok := r.engines[order.Symbol]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no market for symbol %s", order.Symbol)
	}
	return engine.Submit(order)
}

func (r *Registry) Cancel(symbol, orderID string) (*models.Order, error) {
	r.mu.RLock()
	engine, ok := r.engines[symbol]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no market for symbol %s", symbol)
	}
	return engine.Cancel(orderID)
}

func (r *Registry) Snapshot(symbol string, depth int) (*models.OrderBookSnapshot, error) {
	r.mu.RLock()
	engine, ok := r.engines[symbol]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no market for symbol %s", symbol)
	}
	snap := engine.Book().Snapshot(depth)
	return &snap, nil
}

func (r *Registry) TradeChan() <-chan *models.Trade { return r.tradeChan }

func (r *Registry) Symbols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	symbols := make([]string, 0, len(r.engines))
	for s := range r.engines {
		symbols = append(symbols, s)
	}
	return symbols
}
