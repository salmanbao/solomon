package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

const (
	distributionPublishedTopic = "distribution.published"
	distributionFailedTopic    = "distribution.failed"
	defaultConsumerGroup       = "content-marketplace-distribution-cg"
)

type DistributionStatusConsumer struct {
	Subscriber    ports.EventSubscriber
	Claims        ports.ClaimRepository
	Dedup         ports.EventDedupStore
	Clock         ports.Clock
	ConsumerGroup string
	DedupTTL      time.Duration
	Logger        *slog.Logger
}

type distributionStatusPayload struct {
	ClaimID string `json:"claim_id"`
}

func (c DistributionStatusConsumer) Start(ctx context.Context) error {
	group := c.ConsumerGroup
	if group == "" {
		group = defaultConsumerGroup
	}

	if err := c.Subscriber.Subscribe(ctx, distributionPublishedTopic, group, c.handlePublished); err != nil {
		return err
	}
	if err := c.Subscriber.Subscribe(ctx, distributionFailedTopic, group, c.handleFailed); err != nil {
		return err
	}
	return nil
}

func (c DistributionStatusConsumer) handlePublished(ctx context.Context, event ports.EventEnvelope) error {
	return c.handleDistributionStatus(ctx, event, entities.ClaimStatusPublished)
}

func (c DistributionStatusConsumer) handleFailed(ctx context.Context, event ports.EventEnvelope) error {
	return c.handleDistributionStatus(ctx, event, entities.ClaimStatusFailed)
}

func (c DistributionStatusConsumer) handleDistributionStatus(
	ctx context.Context,
	event ports.EventEnvelope,
	targetStatus entities.ClaimStatus,
) error {
	logger := application.ResolveLogger(c.Logger)
	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}

	payloadHash := hashPayload(event.Data)
	alreadyProcessed, err := c.Dedup.ReserveEvent(ctx, event.EventID, payloadHash, now.Add(c.dedupTTL()))
	if err != nil {
		logger.Error("distribution event dedupe failed",
			"event", "content_marketplace_distribution_dedupe_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"error", err.Error(),
		)
		return err
	}
	if alreadyProcessed {
		logger.Debug("distribution event already processed",
			"event", "content_marketplace_distribution_event_replayed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
		)
		return nil
	}

	var payload distributionStatusPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("decode distribution event payload: %w", err)
	}
	if payload.ClaimID == "" {
		return fmt.Errorf("distribution event missing claim_id")
	}

	if err := c.Claims.UpdateClaimStatus(ctx, payload.ClaimID, targetStatus, now); err != nil {
		logger.Error("distribution event claim status update failed",
			"event", "content_marketplace_distribution_claim_update_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"event_id", event.EventID,
			"claim_id", payload.ClaimID,
			"target_status", targetStatus,
			"error", err.Error(),
		)
		return err
	}

	logger.Info("distribution event processed",
		"event", "content_marketplace_distribution_event_processed",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "worker",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"claim_id", payload.ClaimID,
		"target_status", targetStatus,
	)
	return nil
}

func (c DistributionStatusConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
