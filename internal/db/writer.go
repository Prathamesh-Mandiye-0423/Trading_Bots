package db

import (
	"context"
	"fmt"
	"time"

	"github.com/Prathamesh-Mandiye-0423/trading-platform/internal/events"
	"github.com/rs/zerolog/log"
)

type Writer struct {
	consumer     *events.Consumer
	candleBuffer map[string]*candleAccumulator
}

// candleAccumulator def:
type candleAccumulator struct {
	symbol     string
	openTime   time.Time
	open       float64
	high       float64
	low        float64
	close      float64
	volume     float64
	tradeCount int
}

func NewWriter(brokers []string) (*Writer, error) {
	consumer, err := events.NewConsumer(
		brokers,
		"db-writer", //consumer group - separate from supervisor
		[]string{events.TopicTradeExecuted, events.TopicOrderUpdated},
	)
	if err != nil {
		return nil, fmt.Errorf("db.wWiter: failed to create consumer: %w", err)
	}
	w := &Writer{
		consumer:     consumer,
		candleBuffer: make(map[string]*candleAccumulator),
	}
	return w, nil
}

// Start begins consuming events. Blocks until ctx is cancelled
// Run in a go routine: go writer.Start(ctx)

func (w *Writer) Start(ctx context.Context) error {
	log.Info().Msg("db writer started")
	return w.consumer.Start(ctx)
}

// Stop closes consumer connection
func (w *Writer) Stop() {
	w.consumer.Close()
}

// Trade Handler

func (w *Writer) handleTrade(ctx context.Context, topic string, key, value []byte, trade events.TradeEvent) error {
	trade, err := events.UnmarshalTrade(value)
	if err != nil {
		log.Error().Err(err).Msg("db failed to unmarshal trade event")
		return nil
	}
	if err := w.insertTrade(ctx, trade); err != nil {
		log.Error().Err(err).Str("trade_id", trade.ID).Msg("db failed to insert trade")
		return err
	}

	w.updateCandle(ctx, trade)

	log.Debug().
		Str("trade_id", trade.ID).
		Str("symbol", trade.Symbol).
		Str("price", trade.Price.String()).
		Msg("trade persisited")
	return nil
}

func (w *Writer) insertTrade(ctx context.Context, trade events.TradeEvent) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO trades(
			id,symbol,
			buy_order_id, sell_order_id,
			buy_bot_id, sell_bot_id,
			price, quantity, notional,
			executed_at
		) VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		) ON CONFLICT (id, executed_at) DO NOTHING`,
		trade.ID,
		trade.Symbol,
		trade.BuyOrderID,
		trade.SellOrderID,
		trade.BuyBotID,
		trade.SellBotID,
		trade.Price.String(),
		trade.Quantity.String(),
		trade.Notional.String(),
		trade.ExecutedAt,
	)
	return err
}

// Order Handler
func (w *Writer) handleOrder(ctx context.Context, topic string, key, value []byte) error {
	order, err := events.UnmarshalOrder(value)
	if err != nil {
		log.Error().Err(err).Msg("db failed to unmarshal order event")
		return nil
	}

	if err := w.upsertOrder(ctx, order); err != nil {
		log.Error().Err(err).Str("order_id", order.ID).Msg("db: faield to upsert order")
		return err
	}

	log.Debug().
		Str("order_id", order.ID).
		Str("status", string(order.Status)).
		Msg("order persisted")
	return nil
}

func (w *Writer) upsertOrder(ctx context.Context, order events.OrderEvent) error {
	_, err := Pool.Exec(ctx, `
	INSERT INTO orders(
		id, bot_id,symbol, side, type,
		price, quantity, remaining,
		status, created_at, updated_at
	)VALUES(
		$1, $2, $3, $4, $5,
		$6, $7, $8,
		$9, $10, $11
	)
	ON CONFLICT (id) DO UPDATE SET
		remaining=EXCLUDED.remaining,
		status = EXCLUDE.status,
		updated_at = EXCLUDED.updated_at`,
		order.ID,
		order.BotID,
		order.Symbol,
		string(order.Side),
		string(order.Type),
		order.Price.String(),
		order.Quantity.String(),
		order.Remaining.String(),
		string(order.Status),
		order.UpdatedAt,
		order.UpdatedAt,
	)
	return err
}

// candle accumulator
// updateCandle builds 1-minute OHLCV candles from individual trades.
// When a new minute starts, it flushes the previous candle to the database.
func (w *Writer) updateCandle(ctx context.Context, trade events.TradeEvent) {
	symbol := trade.Symbol
	price := trade.Price.Float64()
	qty := trade.Quantity.Float64() //Updated syntax, not sure why _
	candleTime := trade.ExecutedAt.Truncate(time.Minute)

	acc, exists := w.candleBuffer[symbol]
	// If no accumulator exists or we've crossed into a new minute — flush and reset
	if !exists || !acc.openTime.Equal(candleTime) {
		// Flush the completed candle
		if exists {
			if err := w.flushCandle(ctx, acc); err != nil {
				log.Error().Err(err).Str("symbol", symbol).Msg("db:failed to flush candle")
			}
		}
		acc = &candleAccumulator{
			symbol:     symbol,
			openTime:   candleTime,
			open:       price,
			high:       price,
			low:        price,
			volume:     qty,
			tradeCount: 1,
		}
		w.candleBuffer[symbol] = acc
		return
	}
	if price > acc.high {
		acc.high = price
	}
	if price < acc.low {
		acc.low = price
	}
	acc.close = price
	acc.volume += qty
	acc.tradeCount++
}

func (w *Writer) flushCandle(ctx context.Context, acc *candleAccumulator) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO price_candles (
		time,symbol,
		open, high, low, close,
		volume, trade_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (time,symbol) DO UPDATE SET
		 high        = GREATEST(price_candles.high, EXCLUDED.high),
			low         = LEAST(price_candles.low, EXCLUDED.low),
			close       = EXCLUDED.close,
			volume      = price_candles.volume + EXCLUDED.volume,
			trade_count = price_candles.trade_count + EXCLUDED.trade_count
	`,
		acc.openTime,
		acc.symbol,
		acc.open,
		acc.high,
		acc.low,
		acc.close,
		acc.volume,
		acc.tradeCount,
	)
	return err
}
