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

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

const (
	campaignLaunchedTopic         = "campaign.launched"
	defaultCampaignLaunchConsumer = "submission-service-campaign-launched-cg"
)

// CampaignLaunchedConsumer handles M04 campaign launch events.
type CampaignLaunchedConsumer struct {
	Subscriber    ports.EventSubscriber
	Dedup         ports.EventDedupStore
	Clock         ports.Clock
	ConsumerGroup string
	DedupTTL      time.Duration
	Logger        *slog.Logger
}

func (c CampaignLaunchedConsumer) Start(ctx context.Context) error {
	group := strings.TrimSpace(c.ConsumerGroup)
	if group == "" {
		group = defaultCampaignLaunchConsumer
	}
	return c.Subscriber.Subscribe(ctx, campaignLaunchedTopic, group, c.handleCampaignLaunched)
}

func (c CampaignLaunchedConsumer) handleCampaignLaunched(ctx context.Context, event ports.EventEnvelope) error {
	logger := application.ResolveLogger(c.Logger)
	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}

	alreadyProcessed, err := c.Dedup.ReserveEvent(ctx, event.EventID, hashPayload(event.Data), now.Add(c.dedupTTL()))
	if err != nil {
		logger.Error("campaign.launched dedupe failed",
			"event", "submission_campaign_launched_dedupe_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	if alreadyProcessed {
		logger.Debug("campaign.launched already processed",
			"event", "submission_campaign_launched_replayed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}

	var payload struct {
		CampaignID string `json:"campaign_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return fmt.Errorf("decode campaign.launched payload: %w", err)
	}
	if strings.TrimSpace(payload.CampaignID) == "" {
		return fmt.Errorf("campaign.launched payload missing campaign_id")
	}

	logger.Info("campaign launch event consumed",
		"event", "submission_campaign_launched_consumed",
		"module", "campaign-editorial/submission-service",
		"layer", "worker",
		"event_id", event.EventID,
		"campaign_id", payload.CampaignID,
	)
	return nil
}

func (c CampaignLaunchedConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
