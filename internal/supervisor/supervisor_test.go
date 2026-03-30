package supervisor_test

import (
	"testing"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/events"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/models"
	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/supervisor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func orderEvent(botID string) events.OrderEvent {
	return events.OrderEvent{
		ID:     "order-1",
		BotID:  botID,
		Symbol: "BTC-USD",
		Side:   models.SideBuy,
		Type:   models.OrderTypeLimit,
		Status: models.OrderStatusOpen,
	}
}

// ---- Ledger tests ----

func TestLedger_GetOrCreate(t *testing.T) {
	l := supervisor.NewLedger()
	s1 := l.GetOrCreate("bot-1")
	s2 := l.GetOrCreate("bot-1")
	assert.Same(t, s1, s2, "same bot should return same pointer")
}

func TestLedger_OrdersInWindow_Empty(t *testing.T) {
	l := supervisor.NewLedger()
	s := l.GetOrCreate("bot-1")
	assert.Equal(t, 0, s.OrdersInWindow(1*time.Second))
}

func TestLedger_OrdersInWindow_Counts(t *testing.T) {
	l := supervisor.NewLedger()
	s := l.GetOrCreate("bot-1")

	now := time.Now()
	s.RecordOrder(now)
	s.RecordOrder(now)
	s.RecordOrder(now)

	assert.Equal(t, 3, s.OrdersInWindow(1*time.Second))
}

func TestLedger_OrdersInWindow_PrunesOld(t *testing.T) {
	l := supervisor.NewLedger()
	s := l.GetOrCreate("bot-1")

	// Record one order 2 seconds ago — outside the 1s window
	s.RecordOrder(time.Now().Add(-2 * time.Second))
	// Record one order now — inside the window
	s.RecordOrder(time.Now())

	assert.Equal(t, 1, s.OrdersInWindow(1*time.Second))
}

func TestLedger_Suspend(t *testing.T) {
	l := supervisor.NewLedger()
	s := l.GetOrCreate("bot-1")
	assert.False(t, s.Suspended)
	s.Suspend("RATE_LIMIT")
	assert.True(t, s.Suspended)
	assert.Equal(t, "RATE_LIMIT", s.SuspendedBy)
}

// ---- Rule tests ----

func TestRateLimitRule_NoViolation(t *testing.T) {
	rule := supervisor.RateLimitRule(10, 1*time.Second)
	stats := &supervisor.BotStats{BotID: "bot-1"}
	event := orderEvent("bot-1")

	// Submit 5 orders — under the limit
	for i := 0; i < 5; i++ {
		v := rule(stats, event)
		assert.Nil(t, v)
	}
}

func TestRateLimitRule_Violation(t *testing.T) {
	rule := supervisor.RateLimitRule(5, 1*time.Second)
	stats := &supervisor.BotStats{BotID: "bot-1"}
	event := orderEvent("bot-1")

	var lastViolation *supervisor.RuleViolation
	// Submit 10 orders — should trigger after the 6th
	for i := 0; i < 10; i++ {
		lastViolation = rule(stats, event)
	}

	assert.NotNil(t, lastViolation)
	assert.Equal(t, "RATE_LIMIT", lastViolation.Rule)
	assert.Equal(t, "bot-1", lastViolation.BotID)
}

func TestBurstRule_Violation(t *testing.T) {
	rule := supervisor.BurstRule(3, 5*time.Second)
	stats := &supervisor.BotStats{BotID: "bot-1"}
	event := orderEvent("bot-1")

	// Record 4 orders to trigger the > 3 limit
	for i := 0; i < 4; i++ {
		stats.RecordOrder(time.Now())
	}

	v := rule(stats, event)

	// STOP HERE if v is nil (avoids the crash on the next line)
	require.NotNil(t, v)

	// Continue with detailed checks if v is present
	assert.Equal(t, "BURST", v.Rule)
}

func TestRateLimitRule_ResetsAfterWindow(t *testing.T) {
	rule := supervisor.RateLimitRule(5, 100*time.Millisecond)
	stats := &supervisor.BotStats{BotID: "bot-1"}
	event := orderEvent("bot-1")

	// Hit the limit
	for i := 0; i < 6; i++ {
		rule(stats, event)
	}

	// Wait for the window to expire
	time.Sleep(150 * time.Millisecond)

	// Should not violate now — window reset
	v := rule(stats, event)
	assert.Nil(t, v, "rule should not fire after window expires")
}
