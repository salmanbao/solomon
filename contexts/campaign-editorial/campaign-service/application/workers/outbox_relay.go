package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

// OutboxRelay publishes pending M04 outbox rows to the event bus.
type OutboxRelay struct {
	Outbox    ports.OutboxRepository
	Publisher ports.EventPublisher
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
		logger.Error("campaign outbox list failed",
			"event", "campaign_outbox_list_failed",
			"module", "campaign-editorial/campaign-service",
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
		var event ports.EventEnvelope
		if err := json.Unmarshal(row.Payload, &event); err != nil {
			logger.Error("campaign outbox decode failed",
				"event", "campaign_outbox_decode_failed",
				"module", "campaign-editorial/campaign-service",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"error", err.Error(),
			)
			return err
		}

		topic := event.EventType
		if topic == "" {
			topic = row.EventType
		}
		if err := r.Publisher.Publish(ctx, topic, event); err != nil {
			logger.Error("campaign outbox publish failed",
				"event", "campaign_outbox_publish_failed",
				"module", "campaign-editorial/campaign-service",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"event_id", event.EventID,
				"event_type", event.EventType,
				"topic", topic,
				"error", err.Error(),
			)
			return err
		}
		if err := r.Outbox.MarkOutboxPublished(ctx, row.OutboxID, now); err != nil {
			logger.Error("campaign outbox mark published failed",
				"event", "campaign_outbox_mark_published_failed",
				"module", "campaign-editorial/campaign-service",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"error", err.Error(),
			)
			return err
		}
	}

	if len(pending) > 0 {
		logger.Info("campaign outbox relay cycle completed",
			"event", "campaign_outbox_relay_completed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"published_count", len(pending),
		)
	}
	return nil
}
