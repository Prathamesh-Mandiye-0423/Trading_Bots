package events

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// HandlerFunc is called for every message received on a topic
type HandlerFunc func(ctx context.Context, topic string, key, value []byte) error

// Consumer polls Redpanda topics and dispatches to registered handlers.
// Each consumer belongs to a consumer group — Redpanda tracks its offset
// so it resumes from where it left off after a restart.
type Consumer struct {
	client   *kgo.Client
	handlers map[string]HandlerFunc // topic → handler
}

// NewConsumer creates a consumer in the given group subscribing to topics.
// groupID example: "db-writer", "bot-feed", "ticker-emitter"
func NewConsumer(brokers []string, groupID string, topics []string) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		// Start from the earliest unread offset when the group is new
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	)
	if err != nil {
		return nil, fmt.Errorf("events: failed to create consumer (group=%s): %w", groupID, err)
	}

	log.Info().
		Str("group", groupID).
		Strs("topics", topics).
		Msg("event consumer connected")

	return &Consumer{
		client:   client,
		handlers: make(map[string]HandlerFunc),
	}, nil
}

// Register binds a handler to a topic.
// Must be called before Start().
func (c *Consumer) Register(topic string, fn HandlerFunc) {
	c.handlers[topic] = fn
}

// Start begins polling. Blocks until ctx is cancelled.
// Run this in a goroutine: go consumer.Start(ctx)
func (c *Consumer) Start(ctx context.Context) error {
	log.Info().Msg("event consumer starting poll loop")

	for {
		// PollFetches blocks until records arrive or ctx is done
		fetches := c.client.PollFetches(ctx)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Error().
					Err(e.Err).
					Str("topic", e.Topic).
					Int32("partition", e.Partition).
					Msg("consumer fetch error")
			}
			continue
		}

		fetches.EachRecord(func(record *kgo.Record) {
			handler, ok := c.handlers[record.Topic]
			if !ok {
				log.Warn().Str("topic", record.Topic).Msg("no handler registered for topic")
				return
			}

			if err := handler(ctx, record.Topic, record.Key, record.Value); err != nil {
				log.Error().
					Err(err).
					Str("topic", record.Topic).
					Str("key", string(record.Key)).
					Msg("handler error")
				// In production you'd dead-letter this record
			}
		})

		// Commit offsets after processing the batch
		if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
			log.Warn().Err(err).Msg("offset commit failed")
		}
	}
}

// Close stops the consumer cleanly
func (c *Consumer) Close() {
	c.client.Close()
	log.Info().Msg("event consumer closed")
}
