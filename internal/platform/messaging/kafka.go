package messaging

import (
	"context"
	"log/slog"
	"sync"

	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

// Kafka is the event bus adapter used by worker/outbox relay.
// Current implementation is in-process publish/subscribe while runtime wiring
// is finalized for external brokers.
type Kafka struct {
	mu          sync.RWMutex
	subscribers map[string][]chan ports.EventEnvelope
	logger      *slog.Logger
}

func NewKafka(_ []string, logger *slog.Logger) (*Kafka, error) {
	return &Kafka{
		subscribers: make(map[string][]chan ports.EventEnvelope),
		logger:      logger,
	}, nil
}

func (k *Kafka) Publish(ctx context.Context, topic string, event ports.EventEnvelope) error {
	k.mu.RLock()
	subs := append([]chan ports.EventEnvelope(nil), k.subscribers[topic]...)
	k.mu.RUnlock()

	for _, sub := range subs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sub <- event:
		default:
			if k.logger != nil {
				k.logger.Warn("dropping event for slow subscriber",
					"event", "kafka_publish_drop",
					"module", "internal/platform/messaging",
					"layer", "platform",
					"topic", topic,
					"event_id", event.EventID,
				)
			}
		}
	}

	if k.logger != nil {
		k.logger.Info("event published",
			"event", "kafka_publish",
			"module", "internal/platform/messaging",
			"layer", "platform",
			"topic", topic,
			"event_id", event.EventID,
			"event_type", event.EventType,
		)
	}
	return nil
}

func (k *Kafka) Subscribe(
	ctx context.Context,
	topic string,
	consumerGroup string,
	handler func(context.Context, ports.EventEnvelope) error,
) error {
	ch := make(chan ports.EventEnvelope, 128)

	k.mu.Lock()
	k.subscribers[topic] = append(k.subscribers[topic], ch)
	k.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				k.removeSubscriber(topic, ch)
				return
			case event := <-ch:
				if err := handler(ctx, event); err != nil && k.logger != nil {
					k.logger.Error("consumer handler failed",
						"event", "kafka_consume_failed",
						"module", "internal/platform/messaging",
						"layer", "platform",
						"topic", topic,
						"consumer_group", consumerGroup,
						"event_id", event.EventID,
						"event_type", event.EventType,
						"error", err.Error(),
					)
				}
			}
		}
	}()
	return nil
}

func (k *Kafka) removeSubscriber(topic string, target chan ports.EventEnvelope) {
	k.mu.Lock()
	defer k.mu.Unlock()

	items := k.subscribers[topic]
	if len(items) == 0 {
		return
	}
	filtered := make([]chan ports.EventEnvelope, 0, len(items))
	for _, item := range items {
		if item != target {
			filtered = append(filtered, item)
		}
	}
	k.subscribers[topic] = filtered
}
