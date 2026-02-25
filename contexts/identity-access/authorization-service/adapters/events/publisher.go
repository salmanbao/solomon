package events

import (
	"context"
	"log/slog"
	"strings"

	"solomon/contexts/identity-access/authorization-service/ports"
)

const defaultPolicyChangedTopic = "authz.policy_changed"

// policyChangedBus is the minimal bus contract needed for publishing.
type policyChangedBus interface {
	Publish(ctx context.Context, topic string, event ports.PolicyChangedEvent) error
}

// Publisher emits policy-change events to either an event bus adapter or logs.
type Publisher struct {
	logger *slog.Logger
	bus    policyChangedBus
	topic  string
}

// NewPublisher constructs a logging-only publisher adapter.
func NewPublisher(logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{logger: logger}
}

// NewKafkaPublisher constructs a bus-backed publisher for policy-change events.
func NewKafkaPublisher(bus policyChangedBus, logger *slog.Logger, topic string) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	if strings.TrimSpace(topic) == "" {
		topic = defaultPolicyChangedTopic
	}
	return &Publisher{
		logger: logger,
		bus:    bus,
		topic:  topic,
	}
}

// PublishPolicyChanged emits a policy-changed event to the configured transport.
func (p Publisher) PublishPolicyChanged(ctx context.Context, event ports.PolicyChangedEvent) error {
	if p.bus != nil {
		if err := p.bus.Publish(ctx, p.topic, event); err != nil {
			p.logger.Error("policy changed event publish failed",
				"event", "authz_policy_changed_publish_failed",
				"module", "identity-access/authorization-service",
				"layer", "adapter",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"partition_key", event.PartitionKey,
				"topic", p.topic,
				"error", err.Error(),
			)
			return err
		}
	}
	p.logger.Info("policy changed event published",
		"event", "authz_policy_changed_published",
		"module", "identity-access/authorization-service",
		"layer", "adapter",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"partition_key", event.PartitionKey,
		"topic", p.topic,
	)
	return nil
}
