package workers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

const (
	distributionClaimedTopic        = "distribution.claimed"
	defaultClaimedConsumerGroupName = "distribution-service-claimed-cg"
)

type claimedPayload struct {
	ClaimID   string `json:"claim_id"`
	ClipID    string `json:"clip_id"`
	UserID    string `json:"user_id"`
	ClaimType string `json:"claim_type"`
}

type ClaimedConsumer struct {
	Subscriber    ports.EventSubscriber
	Repository    ports.Repository
	Clock         ports.Clock
	IDGen         ports.IDGenerator
	ConsumerGroup string
	Logger        *slog.Logger
}

func (c ClaimedConsumer) Start(ctx context.Context) error {
	logger := application.ResolveLogger(c.Logger)
	group := c.ConsumerGroup
	if group == "" {
		group = defaultClaimedConsumerGroupName
	}
	if err := c.Subscriber.Subscribe(ctx, distributionClaimedTopic, group, c.handle); err != nil {
		logger.Error("distribution claimed consumer subscribe failed",
			"event", "distribution_claimed_consumer_subscribe_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"topic", distributionClaimedTopic,
			"consumer_group", group,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution claimed consumer subscribed",
		"event", "distribution_claimed_consumer_subscribed",
		"module", "campaign-editorial/distribution-service",
		"layer", "worker",
		"topic", distributionClaimedTopic,
		"consumer_group", group,
	)
	return nil
}

func (c ClaimedConsumer) handle(ctx context.Context, event ports.EventEnvelope) error {
	logger := application.ResolveLogger(c.Logger)
	var payload claimedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("distribution claimed event decode failed",
			"event", "distribution_claimed_decode_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	if payload.ClaimID == "" || payload.ClipID == "" || payload.UserID == "" {
		logger.Warn("distribution claimed payload invalid",
			"event", "distribution_claimed_payload_invalid",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"event_id", event.EventID,
			"has_claim_id", payload.ClaimID != "",
			"has_clip_id", payload.ClipID != "",
			"has_user_id", payload.UserID != "",
		)
		return domainerrors.ErrInvalidDistributionInput
	}

	campaignID, err := c.Repository.GetCampaignIDByClip(ctx, payload.ClipID)
	if err != nil {
		logger.Error("distribution claimed campaign lookup failed",
			"event", "distribution_claimed_campaign_lookup_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"event_id", event.EventID,
			"claim_id", payload.ClaimID,
			"clip_id", payload.ClipID,
			"error", err.Error(),
		)
		return err
	}

	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}
	claimUseCase := commands.UseCase{
		Repository: c.Repository,
		Clock:      c.Clock,
		IDGen:      c.IDGen,
		Logger:     c.Logger,
	}
	// M31 idempotency convention:
	// incoming M09 claim_id is reused as distribution_items.id so event replays
	// converge on the same aggregate row.
	_, err = claimUseCase.Claim(ctx, commands.ClaimItemCommand{
		ItemID:       payload.ClaimID,
		InfluencerID: payload.UserID,
		ClipID:       payload.ClipID,
		CampaignID:   campaignID,
	})
	if err != nil && !errors.Is(err, domainerrors.ErrDistributionItemExists) {
		logger.Error("distribution claim ingestion failed",
			"event", "distribution_claim_ingestion_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"event_id", event.EventID,
			"claim_id", payload.ClaimID,
			"error", err.Error(),
		)
		return err
	}

	logger.Info("distribution claim ingested",
		"event", "distribution_claim_ingested",
		"module", "campaign-editorial/distribution-service",
		"layer", "worker",
		"event_id", event.EventID,
		"claim_id", payload.ClaimID,
		"clip_id", payload.ClipID,
		"influencer_id", payload.UserID,
		"claimed_at", now.Format(time.RFC3339),
	)
	return nil
}
