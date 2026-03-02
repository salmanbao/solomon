package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

const (
	submissionCreatedTopic         = "submission.created"
	defaultSubmissionConsumerGroup = "campaign-service-submission-created-cg"
)

// SubmissionCreatedConsumer applies campaign projections from submission.created.
type SubmissionCreatedConsumer struct {
	Subscriber    ports.EventSubscriber
	Campaigns     ports.SubmissionCreatedRepository
	Dedup         ports.EventDedupStore
	Clock         ports.Clock
	ConsumerGroup string
	DedupTTL      time.Duration
	Disabled      bool
	Logger        *slog.Logger
}

func (c SubmissionCreatedConsumer) Start(ctx context.Context) error {
	logger := application.ResolveLogger(c.Logger)
	if c.Disabled {
		logger.Info("submission.created consumer disabled by feature flag",
			"event", "campaign_submission_created_consumer_disabled",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
		)
		return nil
	}
	group := strings.TrimSpace(c.ConsumerGroup)
	if group == "" {
		group = defaultSubmissionConsumerGroup
	}
	return c.Subscriber.Subscribe(ctx, submissionCreatedTopic, group, c.handleSubmissionCreated)
}

func (c SubmissionCreatedConsumer) handleSubmissionCreated(ctx context.Context, event ports.EventEnvelope) error {
	logger := application.ResolveLogger(c.Logger)
	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}

	alreadyProcessed, err := c.Dedup.ReserveEvent(ctx, event.EventID, hashPayload(event.Data), now.Add(c.dedupTTL()))
	if err != nil {
		logger.Error("submission.created dedupe failed",
			"event", "campaign_submission_created_dedupe_failed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	if alreadyProcessed {
		logger.Debug("submission.created already processed",
			"event", "campaign_submission_created_replayed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}

	var payload struct {
		CampaignID string `json:"campaign_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("decode submission.created payload: %w", err)
	}
	if strings.TrimSpace(payload.CampaignID) == "" {
		return fmt.Errorf("submission.created payload missing campaign_id")
	}

	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = now
	}
	result, err := c.Campaigns.ApplySubmissionCreated(ctx, payload.CampaignID, event.EventID, occurredAt)
	if err != nil {
		logger.Error("submission.created projection failed",
			"event", "campaign_submission_created_projection_failed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"event_id", event.EventID,
			"campaign_id", payload.CampaignID,
			"error", err.Error(),
		)
		return err
	}

	logger.Info("submission.created projected",
		"event", "campaign_submission_created_projected",
		"module", "campaign-editorial/campaign-service",
		"layer", "worker",
		"event_id", event.EventID,
		"campaign_id", result.CampaignID,
		"budget_reserved_delta", result.BudgetReservedDelta,
		"budget_remaining", result.BudgetRemaining,
		"auto_paused", result.AutoPaused,
		"new_status", string(result.NewStatus),
	)
	return nil
}

func (c SubmissionCreatedConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
