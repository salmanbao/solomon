package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/ports"
)

type OutboxRelay struct {
	Outbox    ports.OutboxRepository
	Publisher ports.PolicyChangedPublisher
	Clock     ports.Clock
	BatchSize int
	Logger    *slog.Logger
}

func (r OutboxRelay) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(r.Logger)
	limit := r.BatchSize
	if limit <= 0 {
		limit = 100
	}

	pending, err := r.Outbox.ListPendingOutbox(ctx, limit)
	if err != nil {
		logger.Error("authz outbox list failed",
			"event", "authz_outbox_list_failed",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}

	now := time.Now().UTC()
	if r.Clock != nil {
		now = r.Clock.Now().UTC()
	}

	for _, row := range pending {
		var event ports.PolicyChangedEvent
		if err := json.Unmarshal(row.Payload, &event); err != nil {
			return err
		}
		if err := r.Publisher.PublishPolicyChanged(ctx, event); err != nil {
			logger.Error("authz outbox publish failed",
				"event", "authz_outbox_publish_failed",
				"module", "identity-access/authorization-service",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"error", err.Error(),
			)
			return err
		}
		if err := r.Outbox.MarkOutboxPublished(ctx, row.OutboxID, now); err != nil {
			return err
		}
	}
	return nil
}
