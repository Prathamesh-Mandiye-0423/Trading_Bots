package supervisor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// KillAction defines what happens when the supervisor decides to suspend a bot.
// It has two responsibilities:
//  1. Cancel all open orders in the market engine (prevents orphaned positions)
//  2. Signal the sandbox manager to stop the container
//
// Both are fire-and-best-effort — we log failures but don't block.
type KillAction struct {
	marketEngineURL string
	http            *http.Client
}

// NewKillAction creates a kill action pointed at the market engine.
func NewKillAction(marketEngineURL string) *KillAction {
	return &KillAction{
		marketEngineURL: marketEngineURL,
		http:            &http.Client{Timeout: 5 * time.Second},
	}
}

// Execute carries out the suspension:
//  1. Marks the bot as suspended in the ledger
//  2. Cancels all open orders via the market engine API
//  3. Calls the sandbox manager suspend endpoint
func (k *KillAction) Execute(ctx context.Context, stats *BotStats, violation *RuleViolation) {
	log.Warn().
		Str("bot_id", stats.BotID).
		Str("rule", violation.Rule).
		Str("reason", violation.Reason).
		Msg("SUPERVISOR: suspending bot")

	// Step 1 — mark suspended in the ledger immediately
	stats.Suspend(violation.Rule)

	// Step 2 — cancel all open orders for this bot
	if err := k.cancelBotOrders(ctx, stats.BotID); err != nil {
		log.Error().Err(err).Str("bot_id", stats.BotID).Msg("failed to cancel orders on suspension")
	}

	// Step 3 — notify sandbox manager (implemented when sandbox is built)
	k.notifySandbox(ctx, stats.BotID, violation)
}

// cancelBotOrders calls the market engine to cancel all open orders for a bot.
func (k *KillAction) cancelBotOrders(ctx context.Context, botID string) error {
	url := fmt.Sprintf("%s/api/v1/bots/%s/orders", k.marketEngineURL, botID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := k.http.Do(req)
	if err != nil {
		return fmt.Errorf("cancel orders request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("cancel orders returned HTTP %d", resp.StatusCode)
	}

	log.Info().Str("bot_id", botID).Msg("open orders cancelled")
	return nil
}

// notifySandbox sends a suspend signal to the sandbox manager.
// When the sandbox manager is built, it will expose POST /api/v1/bots/:id/suspend.
func (k *KillAction) notifySandbox(ctx context.Context, botID string, v *RuleViolation) {
	url := fmt.Sprintf("%s/api/v1/bots/%s/suspend", k.marketEngineURL, botID)

	payload, _ := json.Marshal(map[string]string{
		"rule":   v.Rule,
		"reason": v.Reason,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url,
		bytes.NewReader(payload))
	if err != nil {
		log.Error().Err(err).Msg("failed to build sandbox suspend request")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.http.Do(req)
	if err != nil {
		// Sandbox manager not yet running — expected during dev
		log.Debug().Str("bot_id", botID).Msg("sandbox manager not reachable (not yet built)")
		return
	}
	defer resp.Body.Close()
	log.Info().Str("bot_id", botID).Msg("sandbox suspend signal sent")
}
