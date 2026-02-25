package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

// OutboxRelay publishes pending M26 outbox rows to the event bus.
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
		logger.Error("submission outbox list failed",
			"event", "submission_outbox_list_failed",
			"module", "campaign-editorial/submission-service",
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
			logger.Error("submission outbox decode failed",
				"event", "submission_outbox_decode_failed",
				"module", "campaign-editorial/submission-service",
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
			logger.Error("submission outbox publish failed",
				"event", "submission_outbox_publish_failed",
				"module", "campaign-editorial/submission-service",
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
			logger.Error("submission outbox mark published failed",
				"event", "submission_outbox_mark_published_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"error", err.Error(),
			)
			return err
		}
	}

	if len(pending) > 0 {
		logger.Info("submission outbox relay cycle completed",
			"event", "submission_outbox_relay_completed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"published_count", len(pending),
		)
	}
	return nil
}
