package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Trade queries ----

// TradeRow is a single trade record returned from the database.
type TradeRow struct {
	ID         string
	Symbol     string
	BuyBotID   string
	SellBotID  string
	Price      string
	Quantity   string
	Notional   string
	ExecutedAt time.Time
}

// GetRecentTrades returns the N most recent trades for a symbol.
func GetRecentTrades(ctx context.Context, symbol string, limit int) ([]TradeRow, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, symbol, buy_bot_id, sell_bot_id,
		       price::text, quantity::text, notional::text, executed_at
		FROM trades
		WHERE symbol = $1
		ORDER BY executed_at DESC
		LIMIT $2`,
		symbol, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[TradeRow])
}

// GetBotTrades returns all trades involving a specific bot.
func GetBotTrades(ctx context.Context, botID string, limit int) ([]TradeRow, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, symbol, buy_bot_id, sell_bot_id,
		       price::text, quantity::text, notional::text, executed_at
		FROM trades
		WHERE buy_bot_id = $1 OR sell_bot_id = $1
		ORDER BY executed_at DESC
		LIMIT $2`,
		botID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[TradeRow])
}

// ---- Candle queries ----

// CandleRow is a single OHLCV candle.
type CandleRow struct {
	Time       time.Time
	Symbol     string
	Open       string
	High       string
	Low        string
	Close      string
	Volume     string
	TradeCount int
}

// GetCandles returns candles for a symbol within a time range.
func GetCandles(ctx context.Context, symbol string, from, to time.Time) ([]CandleRow, error) {
	rows, err := Pool.Query(ctx, `
		SELECT time, symbol,
		       open::text, high::text, low::text, close::text,
		       volume::text, trade_count
		FROM price_candles
		WHERE symbol = $1
		  AND time >= $2
		  AND time <= $3
		ORDER BY time ASC`,
		symbol, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[CandleRow])
}

// ---- Bot P&L query ----

// BotPnL holds a bot's aggregated trading statistics.
type BotPnL struct {
	BotID       string
	TradeCount  int64
	TotalVolume string
	AvgPrice    string
}

// GetBotPnL returns aggregated trade stats for a bot.
// This powers the per-bot performance panel in the dashboard.
func GetBotPnL(ctx context.Context, botID string) (*BotPnL, error) {
	row := Pool.QueryRow(ctx, `
		SELECT
			$1::text                           AS bot_id,
			COUNT(*)                           AS trade_count,
			COALESCE(SUM(quantity), 0)::text   AS total_volume,
			COALESCE(AVG(price), 0)::text      AS avg_price
		FROM trades
		WHERE buy_bot_id = $1 OR sell_bot_id = $1`,
		botID,
	)

	var pnl BotPnL
	err := row.Scan(&pnl.BotID, &pnl.TradeCount, &pnl.TotalVolume, &pnl.AvgPrice)
	if err != nil {
		return nil, err
	}
	return &pnl, nil
}

// ---- ML snapshot queries ----

// MLSnapshotRow is a single ML model snapshot.
type MLSnapshotRow struct {
	ID         string
	Symbol     string
	NSamples   int
	Accuracy   float64
	RecordedAt time.Time
}

// GetMLHistory returns the model accuracy history for a symbol.
// Powers the "model accuracy over time" chart in the dashboard.
func GetMLHistory(ctx context.Context, symbol string, limit int) ([]MLSnapshotRow, error) {
	rows, err := Pool.Query(ctx, `
		SELECT id, symbol, n_samples, accuracy, recorded_at
		FROM ml_snapshots
		WHERE symbol = $1
		ORDER BY recorded_at DESC
		LIMIT $2`,
		symbol, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[MLSnapshotRow])
}

// InsertMLSnapshot saves a model state snapshot.
// Called by the ML service periodically via its own Redpanda consumer.
func InsertMLSnapshot(ctx context.Context, symbol string, nSamples int,
	accuracy float64, weights, signalDist, featureImportance []byte) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO ml_snapshots (
			symbol, n_samples, accuracy,
			weights, signal_dist, feature_importance,
			recorded_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
		symbol, nSamples, accuracy,
		weights, signalDist, featureImportance,
	)
	return err
}
