package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

const (
	campaignPausedTopic    = "campaign.paused"
	campaignCompletedTopic = "campaign.completed"
	defaultCampaignCG      = "voting-engine-campaign-cg"
)

// CampaignStateConsumer reacts to campaign state events and updates/finishes
// voting rounds, emitting round lifecycle events through the outbox.
type CampaignStateConsumer struct {
	Subscriber    ports.EventSubscriber
	Dedup         ports.EventDedupStore
	Votes         ports.VoteRepository
	Outbox        ports.OutboxWriter
	Clock         ports.Clock
	IDGen         ports.IDGenerator
	ConsumerGroup string
	DedupTTL      time.Duration
	Disabled      bool
	Logger        *slog.Logger
}

// Start subscribes M08 to campaign lifecycle topics that affect round states.
// The consumer group can be overridden for environment-specific deployment.
func (c CampaignStateConsumer) Start(ctx context.Context) error {
	logger := application.ResolveLogger(c.Logger)
	if c.Disabled {
		logger.Info("campaign state consumer disabled by feature flag",
			"event", "voting_campaign_consumer_disabled",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
		)
		return nil
	}
	group := strings.TrimSpace(c.ConsumerGroup)
	if group == "" {
		group = defaultCampaignCG
	}
	logger.Info("campaign consumer starting subscriptions",
		"event", "voting_campaign_consumer_starting",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"consumer_group", group,
	)
	if err := c.Subscriber.Subscribe(ctx, campaignPausedTopic, group, c.handleCampaignPaused); err != nil {
		logger.Error("campaign consumer subscribe failed",
			"event", "voting_campaign_consumer_subscribe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"topic", campaignPausedTopic,
			"consumer_group", group,
			"error", err.Error(),
		)
		return err
	}
	if err := c.Subscriber.Subscribe(ctx, campaignCompletedTopic, group, c.handleCampaignCompleted); err != nil {
		logger.Error("campaign consumer subscribe failed",
			"event", "voting_campaign_consumer_subscribe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"topic", campaignCompletedTopic,
			"consumer_group", group,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("campaign consumer subscriptions active",
		"event", "voting_campaign_consumer_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"consumer_group", group,
	)
	return nil
}

func (c CampaignStateConsumer) handleCampaignPaused(ctx context.Context, event ports.EventEnvelope) error {
	// pause -> closing_soon keeps rounds writable while signaling imminent close.
	logger := application.ResolveLogger(c.Logger)
	if alreadyProcessed, err := c.reserveEvent(ctx, event); err != nil {
		return err
	} else if alreadyProcessed {
		logger.Debug("campaign.paused replay skipped",
			"event", "voting_campaign_paused_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}
	var payload struct {
		CampaignID string `json:"campaign_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("campaign.paused payload decode failed",
			"event", "voting_campaign_paused_decode_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	updatedRounds, err := c.Votes.TransitionRoundsForCampaign(
		ctx,
		payload.CampaignID,
		entities.RoundStatusClosingSoon,
		c.now(),
	)
	if err != nil {
		logger.Error("campaign.paused transition failed",
			"event", "voting_campaign_paused_transition_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"campaign_id", strings.TrimSpace(payload.CampaignID),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("campaign.paused consumed",
		"event", "voting_campaign_paused_consumed",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"event_id", event.EventID,
		"campaign_id", strings.TrimSpace(payload.CampaignID),
		"updated_rounds", len(updatedRounds),
	)
	return nil
}

func (c CampaignStateConsumer) handleCampaignCompleted(ctx context.Context, event ports.EventEnvelope) error {
	// completion -> closed and fan-out voting_round.closed for each updated round.
	logger := application.ResolveLogger(c.Logger)
	if alreadyProcessed, err := c.reserveEvent(ctx, event); err != nil {
		return err
	} else if alreadyProcessed {
		logger.Debug("campaign.completed replay skipped",
			"event", "voting_campaign_completed_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}

	var payload struct {
		CampaignID string `json:"campaign_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("campaign.completed payload decode failed",
			"event", "voting_campaign_completed_decode_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	now := c.now()
	updatedRounds, err := c.Votes.TransitionRoundsForCampaign(
		ctx,
		payload.CampaignID,
		entities.RoundStatusClosed,
		now,
	)
	if err != nil {
		logger.Error("campaign.completed transition failed",
			"event", "voting_campaign_completed_transition_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"campaign_id", strings.TrimSpace(payload.CampaignID),
			"error", err.Error(),
		)
		return err
	}

	for _, round := range updatedRounds {
		eventID, err := c.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("campaign.completed event id generation failed",
				"event", "voting_campaign_completed_event_id_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"round_id", round.RoundID,
				"error", err.Error(),
			)
			return err
		}
		envelope, err := newVotingEnvelope(
			eventID,
			"voting_round.closed",
			round.RoundID,
			"round_id",
			now,
			map[string]any{
				"round_id":    round.RoundID,
				"campaign_id": round.CampaignID,
				"status":      string(round.Status),
				"closed_at":   now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("campaign.completed envelope build failed",
				"event", "voting_campaign_completed_envelope_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"round_id", round.RoundID,
				"error", err.Error(),
			)
			return err
		}
		if err := c.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("campaign.completed outbox append failed",
				"event", "voting_campaign_completed_outbox_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"round_id", round.RoundID,
				"error", err.Error(),
			)
			return err
		}
	}

	logger.Info("campaign.completed consumed",
		"event", "voting_campaign_completed_consumed",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"event_id", event.EventID,
		"campaign_id", strings.TrimSpace(payload.CampaignID),
		"closed_rounds", len(updatedRounds),
	)
	return nil
}

func (c CampaignStateConsumer) reserveEvent(ctx context.Context, event ports.EventEnvelope) (bool, error) {
	// ReserveEvent is used as dedupe gate for at-least-once delivery semantics.
	logger := application.ResolveLogger(c.Logger)
	alreadyProcessed, err := c.Dedup.ReserveEvent(ctx, event.EventID, hashPayload(event.Data), c.now().Add(c.dedupTTL()))
	if err != nil {
		logger.Error("campaign event dedupe failed",
			"event", "voting_campaign_event_dedupe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"error", err.Error(),
		)
		return false, err
	}
	return alreadyProcessed, nil
}

func (c CampaignStateConsumer) now() time.Time {
	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}
	return now
}

func (c CampaignStateConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}
