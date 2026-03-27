package events

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Publisher writes events to Redpanda topics.
// It is safe for concurrent use — franz-go handles internal batching.
type Publisher struct {
	client *kgo.Client
}

// NewPublisher creates a connected publisher.
// brokers example: []string{"localhost:19092"}
func NewPublisher(brokers []string) (*Publisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		// Wait up to 10ms to batch records before flushing — reduces syscalls
		kgo.ProducerLinger(10),
		// Retry up to 5 times on transient errors
		kgo.RecordRetries(5),
	)
	if err != nil {
		return nil, fmt.Errorf("events: failed to create publisher: %w", err)
	}

	// Verify connectivity
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("events: cannot reach broker: %w", err)
	}

	log.Info().Strs("brokers", brokers).Msg("event publisher connected")
	return &Publisher{client: client}, nil
}

// PublishTrade publishes a TradeEvent to trade.executed.
// The symbol is used as the partition key so all trades for the
// same market land on the same partition — preserving order.
func (p *Publisher) PublishTrade(ctx context.Context, e TradeEvent) error {
	return p.publish(ctx, TopicTradeExecuted, e.Symbol, e)
}

// PublishOrder publishes an OrderEvent to order.updated.
func (p *Publisher) PublishOrder(ctx context.Context, e OrderEvent) error {
	return p.publish(ctx, TopicOrderUpdated, e.Symbol, e)
}

// PublishTicker publishes a TickerEvent to market.ticker.
func (p *Publisher) PublishTicker(ctx context.Context, e TickerEvent) error {
	return p.publish(ctx, TopicMarketTicker, e.Symbol, e)
}

// publish is the internal generic publish method
func (p *Publisher) publish(ctx context.Context, topic, key string, payload any) error {
	data, err := Marshal(payload)
	if err != nil {
		return fmt.Errorf("events: marshal failed for topic %s: %w", topic, err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}

	// ProduceSync waits for broker ack before returning
	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		return fmt.Errorf("events: publish to %s failed: %w", topic, err)
	}

	log.Debug().
		Str("topic", topic).
		Str("key", key).
		Int("bytes", len(data)).
		Msg("event published")

	return nil
}

// Close flushes pending records and closes the connection
func (p *Publisher) Close() {
	p.client.Flush(context.Background())
	p.client.Close()
	log.Info().Msg("event publisher closed")
}
