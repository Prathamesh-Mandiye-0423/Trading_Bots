package supervisor

import (
	"sync"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/decimal"
)

// BotStats holds the running financial state of a single bot.
// Updated on every trade event from Redpanda.
type BotStats struct {
	BotID string

	// Order rate tracking — sliding window of order timestamps
	orderTimestamps []time.Time
	orderMu         sync.Mutex

	// Suspension state
	Suspended   bool
	SuspendedAt time.Time
	SuspendedBy string // which rule triggered suspension
}

// RecordOrder records a new order submission timestamp.
// Called every time an order event is received for this bot.
func (s *BotStats) RecordOrder(at time.Time) {
	s.orderMu.Lock()
	defer s.orderMu.Unlock()
	s.orderTimestamps = append(s.orderTimestamps, at)
}

// OrdersInWindow counts how many orders were placed within the last `window` duration.
// Used by the rate limit rule.
func (s *BotStats) OrdersInWindow(window time.Duration) int {
	s.orderMu.Lock()
	defer s.orderMu.Unlock()

	cutoff := time.Now().Add(-window)
	count := 0

	// Prune old timestamps while we're here — keep the slice bounded
	pruned := s.orderTimestamps[:0]
	for _, t := range s.orderTimestamps {
		if t.After(cutoff) {
			count++
			pruned = append(pruned, t)
		}
	}
	s.orderTimestamps = pruned
	return count
}

// Suspend marks this bot as suspended.
func (s *BotStats) Suspend(reason string) {
	s.Suspended = true
	s.SuspendedAt = time.Now().UTC()
	s.SuspendedBy = reason
}

// Ledger is a thread-safe registry of per-bot stats.
type Ledger struct {
	mu   sync.RWMutex
	bots map[string]*BotStats
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{bots: make(map[string]*BotStats)}
}

// GetOrCreate returns the stats for a bot, creating a fresh entry if needed.
func (l *Ledger) GetOrCreate(botID string) *BotStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	if s, ok := l.bots[botID]; ok {
		return s
	}
	s := &BotStats{BotID: botID}
	l.bots[botID] = s
	return s
}

// Get returns the stats for a bot, or nil if not found.
func (l *Ledger) Get(botID string) *BotStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.bots[botID]
}

// All returns a snapshot of all bot stats (for monitoring/MCP tools).
func (l *Ledger) All() []*BotStats {
	l.mu.RLock()
	defer l.mu.RUnlock()
	all := make([]*BotStats, 0, len(l.bots))
	for _, s := range l.bots {
		all = append(all, s)
	}
	return all
}

// placeholder so decimal import is used in later files
var _ = decimal.FromInt
