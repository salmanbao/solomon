package commands

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

const (
	defaultScheduleBuffer = 5 * time.Minute
	defaultScheduleWindow = 30 * 24 * time.Hour
	defaultPublishLatency = 4 * time.Minute
)

type ClaimItemCommand struct {
	ItemID       string
	InfluencerID string
	ClipID       string
	CampaignID   string
}

type AddOverlayCommand struct {
	ItemID          string
	OverlayType     string
	AssetPath       string
	DurationSeconds float64
}

type ScheduleCommand struct {
	ItemID       string
	InfluencerID string
	Platform     string
	ScheduledFor time.Time
	Timezone     string
}

type RescheduleCommand struct {
	ItemID       string
	InfluencerID string
	Platform     string
	ScheduledFor time.Time
	Timezone     string
}

type PublishMultiCommand struct {
	ItemID       string
	InfluencerID string
	Platforms    []string
	Caption      string
}

type RetryCommand struct {
	ItemID       string
	InfluencerID string
}

type UseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	IDGen      ports.IDGenerator
	Outbox     ports.OutboxWriter
	Logger     *slog.Logger
}

func (uc UseCase) Claim(ctx context.Context, cmd ClaimItemCommand) (entities.DistributionItem, error) {
	logger := application.ResolveLogger(uc.Logger)
	now := uc.now()
	itemID := strings.TrimSpace(cmd.ItemID)
	if itemID == "" {
		var err error
		itemID, err = uc.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("distribution claim id generation failed",
				"event", "distribution_claim_id_generation_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"error", err.Error(),
			)
			return entities.DistributionItem{}, err
		}
	}
	item := entities.DistributionItem{
		ID:             itemID,
		InfluencerID:   strings.TrimSpace(cmd.InfluencerID),
		ClipID:         strings.TrimSpace(cmd.ClipID),
		CampaignID:     strings.TrimSpace(cmd.CampaignID),
		Status:         entities.DistributionStatusClaimed,
		ClaimedAt:      now,
		ClaimExpiresAt: now.Add(24 * time.Hour),
		UpdatedAt:      now,
	}
	if item.InfluencerID == "" || item.ClipID == "" || item.CampaignID == "" {
		logger.Warn("distribution claim invalid input",
			"event", "distribution_claim_invalid_input",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", item.InfluencerID,
			"clip_id", item.ClipID,
			"campaign_id", item.CampaignID,
		)
		return entities.DistributionItem{}, domainerrors.ErrInvalidDistributionInput
	}
	if err := uc.Repository.CreateItem(ctx, item); err != nil {
		if err == domainerrors.ErrDistributionItemExists {
			logger.Warn("distribution claim already exists",
				"event", "distribution_claim_already_exists",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"influencer_id", item.InfluencerID,
				"clip_id", item.ClipID,
				"campaign_id", item.CampaignID,
			)
			return uc.Repository.GetItem(ctx, item.ID)
		}
		logger.Error("distribution claim create failed",
			"event", "distribution_claim_create_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", item.InfluencerID,
			"clip_id", item.ClipID,
			"campaign_id", item.CampaignID,
			"error", err.Error(),
		)
		return entities.DistributionItem{}, err
	}
	logger.Info("distribution item claimed",
		"event", "distribution_item_claimed",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"influencer_id", item.InfluencerID,
		"clip_id", item.ClipID,
	)
	return item, nil
}

func (uc UseCase) AddOverlay(ctx context.Context, cmd AddOverlayCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		logger.Warn("distribution add overlay item lookup failed",
			"event", "distribution_add_overlay_item_lookup_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", strings.TrimSpace(cmd.ItemID),
			"error", err.Error(),
		)
		return err
	}
	if item.Status == entities.DistributionStatusPublished || item.Status == entities.DistributionStatusCancelled {
		logger.Warn("distribution add overlay invalid state",
			"event", "distribution_add_overlay_invalid_state",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"status", item.Status,
		)
		return domainerrors.ErrInvalidStateTransition
	}
	overlayType := strings.ToLower(strings.TrimSpace(cmd.OverlayType))
	if overlayType != string(entities.OverlayTypeIntro) && overlayType != string(entities.OverlayTypeOutro) {
		logger.Warn("distribution add overlay invalid type",
			"event", "distribution_add_overlay_invalid_type",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"overlay_type", overlayType,
		)
		return domainerrors.ErrInvalidDistributionInput
	}
	if strings.TrimSpace(cmd.AssetPath) == "" || cmd.DurationSeconds <= 0 || cmd.DurationSeconds > 3 {
		logger.Warn("distribution add overlay invalid payload",
			"event", "distribution_add_overlay_invalid_payload",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"overlay_type", overlayType,
			"duration_seconds", cmd.DurationSeconds,
		)
		return domainerrors.ErrInvalidDistributionInput
	}
	overlayID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("distribution add overlay id generation failed",
			"event", "distribution_add_overlay_id_generation_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}
	if err := uc.Repository.AddOverlay(ctx, entities.Overlay{
		ID:                 overlayID,
		DistributionItemID: item.ID,
		OverlayType:        overlayType,
		AssetPath:          strings.TrimSpace(cmd.AssetPath),
		DurationSeconds:    cmd.DurationSeconds,
		CreatedAt:          uc.now(),
	}); err != nil {
		logger.Error("distribution add overlay persistence failed",
			"event", "distribution_add_overlay_persistence_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"overlay_id", overlayID,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution overlay added",
		"event", "distribution_overlay_added",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"overlay_type", overlayType,
	)
	return nil
}

func (uc UseCase) Schedule(ctx context.Context, cmd ScheduleCommand) error {
	return uc.schedule(ctx, scheduleInput{
		itemID:       cmd.ItemID,
		influencerID: cmd.InfluencerID,
		platform:     cmd.Platform,
		scheduledFor: cmd.ScheduledFor,
		timezone:     cmd.Timezone,
		reschedule:   false,
	})
}

func (uc UseCase) Reschedule(ctx context.Context, cmd RescheduleCommand) error {
	return uc.schedule(ctx, scheduleInput{
		itemID:       cmd.ItemID,
		influencerID: cmd.InfluencerID,
		platform:     cmd.Platform,
		scheduledFor: cmd.ScheduledFor,
		timezone:     cmd.Timezone,
		reschedule:   true,
	})
}

func (uc UseCase) PublishMulti(ctx context.Context, cmd PublishMultiCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		logger.Warn("distribution publish multi item lookup failed",
			"event", "distribution_publish_multi_item_lookup_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", strings.TrimSpace(cmd.ItemID),
			"influencer_id", strings.TrimSpace(cmd.InfluencerID),
			"error", err.Error(),
		)
		return err
	}
	if err := uc.publishItem(ctx, &item, strings.TrimSpace(cmd.InfluencerID), cmd.Platforms, cmd.Caption); err != nil {
		logger.Warn("distribution publish multi failed",
			"event", "distribution_publish_multi_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", strings.TrimSpace(cmd.InfluencerID),
			"platform_count", len(cmd.Platforms),
			"error", err.Error(),
		)
		return err
	}
	return nil
}

func (uc UseCase) Retry(ctx context.Context, cmd RetryCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		logger.Warn("distribution retry item lookup failed",
			"event", "distribution_retry_item_lookup_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", strings.TrimSpace(cmd.ItemID),
			"influencer_id", strings.TrimSpace(cmd.InfluencerID),
			"error", err.Error(),
		)
		return err
	}
	if item.InfluencerID != strings.TrimSpace(cmd.InfluencerID) {
		logger.Warn("distribution retry unauthorized",
			"event", "distribution_retry_unauthorized",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", strings.TrimSpace(cmd.InfluencerID),
		)
		return domainerrors.ErrUnauthorizedInfluencer
	}
	if item.Status != entities.DistributionStatusFailed {
		logger.Warn("distribution retry invalid state",
			"event", "distribution_retry_invalid_state",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"status", item.Status,
		)
		return domainerrors.ErrInvalidStateTransition
	}
	item.Status = entities.DistributionStatusPublishing
	item.RetryCount = 0
	item.LastError = ""
	now := uc.now()
	item.UpdatedAt = now
	if err := uc.Repository.UpdateItem(ctx, item); err != nil {
		logger.Error("distribution retry state update failed",
			"event", "distribution_retry_state_update_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}
	logger.Warn("distribution retry requested",
		"event", "distribution_retry_requested",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"influencer_id", item.InfluencerID,
	)
	if err := uc.publishItem(ctx, &item, item.InfluencerID, item.Platforms, item.Caption); err != nil {
		uc.markPublishFailed(ctx, &item, err, "retry")
		logger.Error("distribution retry publish failed",
			"event", "distribution_retry_publish_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}
	return nil
}

func (uc UseCase) ProcessDueScheduled(ctx context.Context, limit int) error {
	logger := application.ResolveLogger(uc.Logger)
	due, err := uc.Repository.ListDueScheduled(ctx, uc.now(), limit)
	if err != nil {
		logger.Error("distribution scheduler due list failed",
			"event", "distribution_scheduler_due_list_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"limit", limit,
			"error", err.Error(),
		)
		return err
	}
	var firstErr error
	for idx := range due {
		item := due[idx]
		if err := uc.publishItem(ctx, &item, item.InfluencerID, item.Platforms, item.Caption); err != nil {
			uc.markPublishFailed(ctx, &item, err, "scheduler")
			if firstErr == nil {
				firstErr = err
			}
			logger.Error("distribution scheduler publish failed",
				"event", "distribution_scheduler_publish_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "worker",
				"item_id", item.ID,
				"error", err.Error(),
			)
		}
	}
	if len(due) > 0 {
		logger.Info("distribution scheduler cycle completed",
			"event", "distribution_scheduler_cycle_completed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"due_count", len(due),
		)
	}
	return firstErr
}

type scheduleInput struct {
	itemID       string
	influencerID string
	platform     string
	scheduledFor time.Time
	timezone     string
	reschedule   bool
}

func (uc UseCase) schedule(ctx context.Context, input scheduleInput) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(input.itemID))
	if err != nil {
		logger.Warn("distribution schedule item lookup failed",
			"event", "distribution_schedule_item_lookup_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", strings.TrimSpace(input.itemID),
			"influencer_id", strings.TrimSpace(input.influencerID),
			"error", err.Error(),
		)
		return err
	}
	if item.InfluencerID != strings.TrimSpace(input.influencerID) {
		logger.Warn("distribution schedule unauthorized",
			"event", "distribution_schedule_unauthorized",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", strings.TrimSpace(input.influencerID),
		)
		return domainerrors.ErrUnauthorizedInfluencer
	}
	if input.reschedule {
		if item.Status != entities.DistributionStatusScheduled {
			logger.Warn("distribution reschedule invalid state",
				"event", "distribution_reschedule_invalid_state",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"status", item.Status,
			)
			return domainerrors.ErrInvalidStateTransition
		}
	} else if item.Status != entities.DistributionStatusClaimed && item.Status != entities.DistributionStatusFailed {
		logger.Warn("distribution schedule invalid state",
			"event", "distribution_schedule_invalid_state",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"status", item.Status,
		)
		return domainerrors.ErrInvalidStateTransition
	}

	platform, err := normalizePlatform(input.platform)
	if err != nil {
		logger.Warn("distribution schedule invalid platform",
			"event", "distribution_schedule_invalid_platform",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"platform", strings.TrimSpace(input.platform),
			"error", err.Error(),
		)
		return err
	}

	now := uc.now()
	// Canonical guardrail from M31 spec:
	// schedule must stay within [now+5m, now+30d].
	if input.scheduledFor.Before(now.Add(defaultScheduleBuffer)) || input.scheduledFor.After(now.Add(defaultScheduleWindow)) {
		logger.Warn("distribution schedule outside allowed window",
			"event", "distribution_schedule_outside_window",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"scheduled_for_utc", input.scheduledFor.UTC().Format(time.RFC3339),
		)
		return domainerrors.ErrInvalidScheduleWindow
	}
	item.Status = entities.DistributionStatusScheduled
	scheduled := input.scheduledFor.UTC()
	item.ScheduledForUTC = &scheduled
	item.Timezone = strings.TrimSpace(input.timezone)
	item.Platforms = []string{platform}
	item.UpdatedAt = now
	if err := uc.Repository.UpdateItem(ctx, item); err != nil {
		logger.Error("distribution schedule state update failed",
			"event", "distribution_schedule_state_update_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}

	action := "distribution_item_scheduled"
	if input.reschedule {
		action = "distribution_item_rescheduled"
	}
	logger.Info("distribution item schedule updated",
		"event", action,
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"scheduled_for_utc", scheduled.Format(time.RFC3339),
		"timezone", item.Timezone,
		"platform", platform,
	)
	return nil
}

func (uc UseCase) publishItem(
	ctx context.Context,
	item *entities.DistributionItem,
	influencerID string,
	requestedPlatforms []string,
	caption string,
) error {
	logger := application.ResolveLogger(uc.Logger)
	if item.InfluencerID != strings.TrimSpace(influencerID) {
		logger.Warn("distribution publish unauthorized",
			"event", "distribution_publish_unauthorized",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"influencer_id", strings.TrimSpace(influencerID),
		)
		return domainerrors.ErrUnauthorizedInfluencer
	}
	if item.Status != entities.DistributionStatusClaimed &&
		item.Status != entities.DistributionStatusScheduled &&
		item.Status != entities.DistributionStatusFailed &&
		item.Status != entities.DistributionStatusPublishing {
		logger.Warn("distribution publish invalid state",
			"event", "distribution_publish_invalid_state",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"status", item.Status,
		)
		return domainerrors.ErrInvalidStateTransition
	}

	platforms := requestedPlatforms
	if len(platforms) == 0 {
		platforms = item.Platforms
	}
	normalizedPlatforms, err := normalizePlatforms(platforms)
	if err != nil {
		logger.Warn("distribution publish platform validation failed",
			"event", "distribution_publish_platform_validation_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}

	now := uc.now()
	startedAt := now
	item.Status = entities.DistributionStatusPublishing
	item.PublishStartedAt = &startedAt
	item.UpdatedAt = now
	if err := uc.Repository.UpdateItem(ctx, *item); err != nil {
		logger.Error("distribution publish start state update failed",
			"event", "distribution_publish_start_state_update_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}

	completedAt := uc.now()
	item.Status = entities.DistributionStatusPublished
	item.Platforms = normalizedPlatforms
	item.Caption = strings.TrimSpace(caption)
	item.PublishCompletedAt = &completedAt
	item.PublishedAt = &completedAt
	item.UpdatedAt = completedAt
	item.LastError = ""
	if err := uc.Repository.UpdateItem(ctx, *item); err != nil {
		logger.Error("distribution publish complete state update failed",
			"event", "distribution_publish_complete_state_update_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}

	captionID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("distribution publish caption id generation failed",
			"event", "distribution_publish_caption_id_generation_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}
	if err := uc.Repository.UpsertCaption(ctx, entities.Caption{
		ID:                 captionID,
		DistributionItemID: item.ID,
		Platform:           "",
		CaptionText:        item.Caption,
		Hashtags:           append([]string(nil), item.Hashtags...),
		CreatedAt:          now,
		UpdatedAt:          completedAt,
	}); err != nil {
		logger.Error("distribution publish caption upsert failed",
			"event", "distribution_publish_caption_upsert_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"caption_id", captionID,
			"error", err.Error(),
		)
		return err
	}

	for _, platform := range normalizedPlatforms {
		statusID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("distribution publish platform status id generation failed",
				"event", "distribution_publish_platform_status_id_generation_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"platform", platform,
				"error", err.Error(),
			)
			return err
		}
		platformPublishedAt := completedAt
		if err := uc.Repository.UpsertPlatformStatus(ctx, entities.PlatformStatus{
			ID:                 statusID,
			DistributionItemID: item.ID,
			Platform:           platform,
			Status:             "published",
			PlatformPostID:     item.ID,
			PlatformPostURL:    "https://social.example/" + platform + "/post/" + item.ID,
			RetryCount:         item.RetryCount,
			MaxRetries:         3,
			PublishedAt:        &platformPublishedAt,
			UpdatedAt:          completedAt,
		}); err != nil {
			logger.Error("distribution publish platform status upsert failed",
				"event", "distribution_publish_platform_status_upsert_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"platform", platform,
				"status_id", statusID,
				"error", err.Error(),
			)
			return err
		}
		analyticsID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("distribution publish analytics id generation failed",
				"event", "distribution_publish_analytics_id_generation_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"platform", platform,
				"error", err.Error(),
			)
			return err
		}
		duration := int(defaultPublishLatency.Seconds())
		if item.PublishStartedAt != nil {
			duration = int(platformPublishedAt.Sub(item.PublishStartedAt.UTC()).Seconds())
			if duration < 0 {
				duration = 0
			}
		}
		if err := uc.Repository.AddPublishingAnalytics(ctx, entities.PublishingAnalytics{
			ID:                   analyticsID,
			DistributionItemID:   item.ID,
			InfluencerID:         item.InfluencerID,
			CampaignID:           item.CampaignID,
			Platform:             platform,
			Success:              true,
			Status:               "published",
			ClaimedAt:            &item.ClaimedAt,
			PublishStartedAt:     item.PublishStartedAt,
			PublishCompletedAt:   &platformPublishedAt,
			TimeToPublishSeconds: &duration,
			CreatedAt:            completedAt,
		}); err != nil {
			logger.Error("distribution publish analytics persistence failed",
				"event", "distribution_publish_analytics_persistence_failed",
				"module", "campaign-editorial/distribution-service",
				"layer", "application",
				"item_id", item.ID,
				"platform", platform,
				"analytics_id", analyticsID,
				"error", err.Error(),
			)
			return err
		}
	}

	// Publish outcomes are emitted through outbox to keep DB state and event side-effects decoupled.
	if err := uc.appendOutbox(ctx, "distribution.published", item.ID, map[string]any{
		"claim_id":             item.ID,
		"distribution_item_id": item.ID,
	}); err != nil {
		logger.Error("distribution publish outbox append failed",
			"event", "distribution_publish_outbox_append_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"error", err.Error(),
		)
		return err
	}

	logger.Info("distribution item published",
		"event", "distribution_item_published",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"influencer_id", item.InfluencerID,
		"platform_count", len(normalizedPlatforms),
	)
	return nil
}

func (uc UseCase) appendOutbox(
	ctx context.Context,
	eventType string,
	partitionKey string,
	data map[string]any,
) error {
	logger := application.ResolveLogger(uc.Logger)
	if uc.Outbox == nil {
		logger.Debug("distribution outbox disabled for module",
			"event", "distribution_outbox_disabled",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"event_type", eventType,
			"partition_key", partitionKey,
		)
		return nil
	}
	eventID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("distribution outbox event id generation failed",
			"event", "distribution_outbox_event_id_generation_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"event_type", eventType,
			"partition_key", partitionKey,
			"error", err.Error(),
		)
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		logger.Error("distribution outbox payload marshal failed",
			"event", "distribution_outbox_payload_marshal_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"event_type", eventType,
			"partition_key", partitionKey,
			"error", err.Error(),
		)
		return err
	}
	if err := uc.Outbox.AppendOutbox(ctx, ports.EventEnvelope{
		EventID:          eventID,
		EventType:        eventType,
		OccurredAt:       uc.now(),
		SourceService:    "distribution-service",
		TraceID:          eventID,
		SchemaVersion:    1,
		PartitionKeyPath: "claim_id",
		PartitionKey:     partitionKey,
		Data:             payload,
	}); err != nil {
		logger.Error("distribution outbox append failed",
			"event", "distribution_outbox_append_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"event_id", eventID,
			"event_type", eventType,
			"partition_key", partitionKey,
			"error", err.Error(),
		)
		return err
	}
	return nil
}

func (uc UseCase) now() time.Time {
	if uc.Clock == nil {
		return time.Now().UTC()
	}
	return uc.Clock.Now().UTC()
}

func normalizePlatforms(platforms []string) ([]string, error) {
	normalized := make([]string, 0, len(platforms))
	seen := map[string]struct{}{}
	for _, platform := range platforms {
		value, err := normalizePlatform(platform)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil, domainerrors.ErrInvalidDistributionInput
	}
	return normalized, nil
}

func normalizePlatform(value string) (string, error) {
	platform := strings.ToLower(strings.TrimSpace(value))
	switch platform {
	case "tiktok", "instagram", "youtube", "snapchat":
		return platform, nil
	default:
		return "", domainerrors.ErrUnsupportedPlatform
	}
}

func (uc UseCase) markPublishFailed(
	ctx context.Context,
	item *entities.DistributionItem,
	cause error,
	trigger string,
) {
	if item == nil {
		return
	}
	logger := application.ResolveLogger(uc.Logger)
	now := uc.now()
	item.Status = entities.DistributionStatusFailed
	item.RetryCount++
	item.LastError = strings.TrimSpace(cause.Error())
	item.UpdatedAt = now

	if err := uc.Repository.UpdateItem(ctx, *item); err != nil {
		logger.Error("distribution publish failure state update failed",
			"event", "distribution_publish_failure_state_update_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"trigger", strings.TrimSpace(trigger),
			"error", err.Error(),
		)
		return
	}
	if err := uc.appendOutbox(ctx, "distribution.failed", item.ID, map[string]any{
		"claim_id":             item.ID,
		"distribution_item_id": item.ID,
		"reason":               item.LastError,
	}); err != nil {
		logger.Error("distribution publish failure outbox append failed",
			"event", "distribution_publish_failure_outbox_append_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", item.ID,
			"trigger", strings.TrimSpace(trigger),
			"error", err.Error(),
		)
		return
	}
	logger.Warn("distribution item marked failed",
		"event", "distribution_item_marked_failed",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"trigger", strings.TrimSpace(trigger),
		"retry_count", item.RetryCount,
	)
}
