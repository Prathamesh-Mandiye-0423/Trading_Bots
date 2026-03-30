package supervisor

import (
	"fmt"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/events"
)

// RuleViolation is returned when a risk rule fires.
type RuleViolation struct {
	Rule   string // rule name
	BotID  string
	Reason string // human-readable explanation
}

func (v *RuleViolation) Error() string {
	return fmt.Sprintf("[%s] bot=%s reason=%s", v.Rule, v.BotID, v.Reason)
}

// RuleFunc is a function that inspects an order event and returns a
// violation if the rule is breached, or nil if everything is fine.
// Each rule is stateless — all state lives in the ledger.
type RuleFunc func(stats *BotStats, event events.OrderEvent) *RuleViolation

// ---- Rule: order rate limit ----

// RateLimitRule returns a RuleFunc that fires when a bot submits more than
// maxOrders orders within window. Default: 10 orders per second.
//
// This catches:
//   - Runaway loops in bot code (infinite order spam)
//   - Deliberate denial-of-service attacks against the matching engine
//   - Bugs where a bot re-submits on every tick without checking state
func RateLimitRule(maxOrders int, window time.Duration) RuleFunc {
	return func(stats *BotStats, event events.OrderEvent) *RuleViolation {
		// Record this order in the sliding window
		stats.RecordOrder(time.Now())

		count := stats.OrdersInWindow(window)
		if count > maxOrders {
			return &RuleViolation{
				Rule:  "RATE_LIMIT",
				BotID: stats.BotID,
				Reason: fmt.Sprintf(
					"submitted %d orders in %s (max %d) — possible runaway loop",
					count, window, maxOrders,
				),
			}
		}
		return nil
	}
}

// ---- Rule: order burst ----

// BurstRule fires when a bot submits more than maxOrders in a very short
// window (e.g. 50 orders in 1 second). Catches sudden spikes that the
// rolling rate limit might miss at the boundary.
func BurstRule(maxOrders int, window time.Duration) RuleFunc {
	return func(stats *BotStats, event events.OrderEvent) *RuleViolation {
		count := stats.OrdersInWindow(window)
		if count > maxOrders {
			return &RuleViolation{
				Rule:  "BURST",
				BotID: stats.BotID,
				Reason: fmt.Sprintf(
					"burst detected: %d orders in %s (max %d)",
					count, window, maxOrders,
				),
			}
		}
		return nil
	}
}

// DefaultRules returns the production rule set.
// These are the rules the supervisor runs on every order event.
func DefaultRules() []RuleFunc {
	return []RuleFunc{
		// Primary rate limit: max 10 orders per second (rolling window)
		RateLimitRule(10, 1*time.Second),

		// Burst protection: max 30 orders per 5 seconds
		BurstRule(30, 5*time.Second),
	}
}
