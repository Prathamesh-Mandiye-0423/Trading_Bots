package supervisor

import (
	"context"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/events"
	"github.com/rs/zerolog/log"
)

// Supervisor is the risk referee. It:
//   - Maintains a shadow ledger of every bot's order activity
//   - Runs a chain of risk rules on every order event
//   - Triggers the kill action when a rule fires
//
// It runs as a long-lived goroutine consuming from Redpanda.
type Supervisor struct {
	ledger     *Ledger
	rules      []RuleFunc
	killAction *KillAction
	consumer   *events.Consumer
}

// Config holds all tunable parameters for the supervisor.
type Config struct {
	// BrokerAddrs is the list of Redpanda broker addresses.
	BrokerAddrs []string

	// MarketEngineURL is used by the kill action to cancel orders.
	MarketEngineURL string

	// Rules is the list of risk rules to enforce.
	// Defaults to DefaultRules() if nil.
	Rules []RuleFunc
}

// New creates a Supervisor and connects it to Redpanda.
func New(cfg Config) (*Supervisor, error) {
	rules := cfg.Rules
	if len(rules) == 0 {
		rules = DefaultRules()
	}

	consumer, err := events.NewConsumer(
		cfg.BrokerAddrs,
		"supervisor", // consumer group ID
		[]string{events.TopicOrderUpdated},
	)
	if err != nil {
		return nil, err
	}

	s := &Supervisor{
		ledger:     NewLedger(),
		rules:      rules,
		killAction: NewKillAction(cfg.MarketEngineURL),
		consumer:   consumer,
	}

	// Register handler for order events
	consumer.Register(events.TopicOrderUpdated, s.handleOrderEvent)

	return s, nil
}

// Start begins consuming order events. Blocks until ctx is cancelled.
// Run this in a goroutine: go supervisor.Start(ctx)
func (s *Supervisor) Start(ctx context.Context) error {
	log.Info().Msg("supervisor starting")
	return s.consumer.Start(ctx)
}

// Stop closes the consumer connection.
func (s *Supervisor) Stop() {
	s.consumer.Close()
}

// Ledger exposes the shadow ledger for the MCP server tools.
func (s *Supervisor) Ledger() *Ledger {
	return s.ledger
}

// handleOrderEvent is called by the consumer for every order.updated message.
func (s *Supervisor) handleOrderEvent(
	ctx context.Context,
	topic string,
	key, value []byte,
) error {
	event, err := events.UnmarshalOrder(value)
	if err != nil {
		log.Error().Err(err).Msg("supervisor: failed to unmarshal order event")
		return nil // don't crash the consumer on bad messages
	}

	stats := s.ledger.GetOrCreate(event.BotID)

	// Skip already-suspended bots — they've been dealt with
	if stats.Suspended {
		return nil
	}

	// Run every rule — stop at first violation
	for _, rule := range s.rules {
		if violation := rule(stats, event); violation != nil {
			s.killAction.Execute(ctx, stats, violation)
			return nil // one violation is enough
		}
	}

	return nil
}

// SuspendBot allows external callers (e.g. MCP tools) to manually suspend a bot.
func (s *Supervisor) SuspendBot(ctx context.Context, botID, reason string) {
	stats := s.ledger.GetOrCreate(botID)
	violation := &RuleViolation{
		Rule:   "MANUAL",
		BotID:  botID,
		Reason: reason,
	}
	s.killAction.Execute(ctx, stats, violation)
}

// BotStatus returns the current status of a bot for monitoring.
func (s *Supervisor) BotStatus(botID string) map[string]any {
	stats := s.ledger.Get(botID)
	if stats == nil {
		return map[string]any{"bot_id": botID, "known": false}
	}
	return map[string]any{
		"bot_id":             stats.BotID,
		"known":              true,
		"suspended":          stats.Suspended,
		"suspended_by":       stats.SuspendedBy,
		"suspended_at":       stats.SuspendedAt,
		"orders_last_second": stats.OrdersInWindow(0), // filled in by caller
	}
}
