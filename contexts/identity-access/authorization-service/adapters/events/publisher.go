package events

import (
	"context"
	"log/slog"

	"solomon/contexts/identity-access/authorization-service/ports"
)

// Publisher is a placeholder policy-change event publisher.
// It is intentionally minimal while messaging runtime integration is completed.
type Publisher struct {
	logger *slog.Logger
}

func NewPublisher(logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{logger: logger}
}

func (p Publisher) PublishPolicyChanged(_ context.Context, event ports.PolicyChangedEvent) error {
	p.logger.Info("policy changed event published",
		"event", "authz_policy_changed_published",
		"module", "identity-access/authorization-service",
		"layer", "adapter",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"partition_key", event.PartitionKey,
	)
	return nil
}
