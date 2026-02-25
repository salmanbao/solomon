package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type OutboxRelay struct {
	Outbox    ports.OutboxRepository
	Publisher ports.EventPublisher
	Clock     ports.Clock
	Topic     string
	BatchSize int
	Logger    *slog.Logger
}

func (r OutboxRelay) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(r.Logger)
	limit := r.BatchSize
	if limit <= 0 {
		limit = 100
	}
	topic := r.Topic
	if topic == "" {
		topic = "distribution.claimed"
	}

	pending, err := r.Outbox.ListPendingOutbox(ctx, limit)
	if err != nil {
		logger.Error("outbox list pending failed",
			"event", "content_marketplace_outbox_list_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}

	now := time.Now().UTC()
	if r.Clock != nil {
		now = r.Clock.Now().UTC()
	}

	for _, message := range pending {
		var envelope ports.EventEnvelope
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			logger.Error("outbox payload decode failed",
				"event", "content_marketplace_outbox_decode_failed",
				"module", "campaign-editorial/content-library-marketplace",
				"layer", "worker",
				"outbox_id", message.OutboxID,
				"error", err.Error(),
			)
			return err
		}

		if err := r.Publisher.Publish(ctx, topic, envelope); err != nil {
			logger.Error("outbox publish failed",
				"event", "content_marketplace_outbox_publish_failed",
				"module", "campaign-editorial/content-library-marketplace",
				"layer", "worker",
				"outbox_id", message.OutboxID,
				"event_id", envelope.EventID,
				"event_type", envelope.EventType,
				"error", err.Error(),
			)
			return err
		}
		if err := r.Outbox.MarkOutboxSent(ctx, message.OutboxID, now); err != nil {
			logger.Error("outbox mark sent failed",
				"event", "content_marketplace_outbox_mark_sent_failed",
				"module", "campaign-editorial/content-library-marketplace",
				"layer", "worker",
				"outbox_id", message.OutboxID,
				"error", err.Error(),
			)
			return err
		}
	}

	if len(pending) > 0 {
		logger.Info("outbox relay cycle completed",
			"event", "content_marketplace_outbox_relay_completed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"sent_count", len(pending),
		)
	}
	return nil
}
