package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

// OutboxRelay publishes persisted outbox records to the event bus.
type OutboxRelay struct {
	Outbox    ports.OutboxRepository
	Publisher ports.EventPublisher
	Clock     ports.Clock
	BatchSize int
	Logger    *slog.Logger
}

// RunOnce publishes a bounded batch of pending outbox rows and marks each row
// published only after broker publish succeeds. It stops on the first failure
// so the retry loop can reprocess remaining rows safely.
func (r OutboxRelay) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(r.Logger)
	limit := r.BatchSize
	if limit <= 0 {
		limit = 100
	}
	logger.Info("voting outbox relay cycle started",
		"event", "voting_outbox_relay_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"batch_size", limit,
	)

	pending, err := r.Outbox.ListPendingOutbox(ctx, limit)
	if err != nil {
		logger.Error("voting outbox list failed",
			"event", "voting_outbox_list_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}
	if len(pending) == 0 {
		logger.Debug("voting outbox relay found no pending rows",
			"event", "voting_outbox_relay_noop",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"batch_size", limit,
		)
		return nil
	}

	now := time.Now().UTC()
	if r.Clock != nil {
		now = r.Clock.Now().UTC()
	}

	for _, row := range pending {
		var event ports.EventEnvelope
		if err := json.Unmarshal(row.Payload, &event); err != nil {
			logger.Error("voting outbox decode failed",
				"event", "voting_outbox_decode_failed",
				"module", "campaign-editorial/voting-engine",
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
			logger.Error("voting outbox publish failed",
				"event", "voting_outbox_publish_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"event_id", event.EventID,
				"event_type", event.EventType,
				"error", err.Error(),
			)
			return err
		}
		if err := r.Outbox.MarkOutboxPublished(ctx, row.OutboxID, now); err != nil {
			logger.Error("voting outbox mark published failed",
				"event", "voting_outbox_mark_published_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"outbox_id", row.OutboxID,
				"error", err.Error(),
			)
			return err
		}
	}

	logger.Info("voting outbox relay cycle completed",
		"event", "voting_outbox_relay_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"published_count", len(pending),
	)
	return nil
}
