package commands

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
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
	Logger     *slog.Logger
}

func (uc UseCase) Claim(ctx context.Context, cmd ClaimItemCommand) (entities.DistributionItem, error) {
	now := uc.Clock.Now().UTC()
	itemID := strings.TrimSpace(cmd.ItemID)
	if itemID == "" {
		var err error
		itemID, err = uc.IDGen.NewID(ctx)
		if err != nil {
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
		return entities.DistributionItem{}, domainerrors.ErrInvalidDistributionInput
	}
	if err := uc.Repository.CreateItem(ctx, item); err != nil {
		return entities.DistributionItem{}, err
	}
	application.ResolveLogger(uc.Logger).Info("distribution item claimed",
		"event", "distribution_item_claimed",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
	)
	return item, nil
}

func (uc UseCase) AddOverlay(ctx context.Context, cmd AddOverlayCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		return err
	}
	if item.Status == entities.DistributionStatusPublished {
		return domainerrors.ErrInvalidStateTransition
	}
	if cmd.DurationSeconds <= 0 || cmd.DurationSeconds > 3 {
		return domainerrors.ErrInvalidDistributionInput
	}
	overlayID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	if err := uc.Repository.AddOverlay(ctx, entities.Overlay{
		ID:                 overlayID,
		DistributionItemID: item.ID,
		OverlayType:        strings.TrimSpace(cmd.OverlayType),
		AssetPath:          strings.TrimSpace(cmd.AssetPath),
		DurationSeconds:    cmd.DurationSeconds,
		CreatedAt:          uc.Clock.Now().UTC(),
	}); err != nil {
		return err
	}
	logger.Info("distribution overlay added",
		"event", "distribution_overlay_added",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
	)
	return nil
}

func (uc UseCase) Schedule(ctx context.Context, cmd ScheduleCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		return err
	}
	if item.InfluencerID != strings.TrimSpace(cmd.InfluencerID) {
		return domainerrors.ErrInvalidDistributionInput
	}
	now := uc.Clock.Now().UTC()
	if cmd.ScheduledFor.Before(now.Add(5*time.Minute)) || cmd.ScheduledFor.After(now.Add(30*24*time.Hour)) {
		return domainerrors.ErrInvalidScheduleWindow
	}
	item.Status = entities.DistributionStatusScheduled
	scheduled := cmd.ScheduledFor.UTC()
	item.ScheduledForUTC = &scheduled
	item.Timezone = strings.TrimSpace(cmd.Timezone)
	item.Platforms = []string{strings.TrimSpace(cmd.Platform)}
	item.UpdatedAt = now
	if err := uc.Repository.UpdateItem(ctx, item); err != nil {
		return err
	}
	logger.Info("distribution item scheduled",
		"event", "distribution_item_scheduled",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"scheduled_for_utc", scheduled.Format(time.RFC3339),
	)
	return nil
}

func (uc UseCase) PublishMulti(ctx context.Context, cmd PublishMultiCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		return err
	}
	if item.InfluencerID != strings.TrimSpace(cmd.InfluencerID) {
		return domainerrors.ErrInvalidDistributionInput
	}
	if item.Status != entities.DistributionStatusClaimed && item.Status != entities.DistributionStatusScheduled && item.Status != entities.DistributionStatusFailed {
		return domainerrors.ErrInvalidStateTransition
	}

	item.Status = entities.DistributionStatusPublished
	item.Platforms = append([]string(nil), cmd.Platforms...)
	item.Caption = strings.TrimSpace(cmd.Caption)
	now := uc.Clock.Now().UTC()
	item.PublishedAt = &now
	item.UpdatedAt = now
	if err := uc.Repository.UpdateItem(ctx, item); err != nil {
		return err
	}
	for _, platform := range item.Platforms {
		statusID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			return err
		}
		if err := uc.Repository.UpsertPlatformStatus(ctx, entities.PlatformStatus{
			ID:                 statusID,
			DistributionItemID: item.ID,
			Platform:           platform,
			Status:             "published",
			PlatformPostURL:    "https://social.example/" + platform + "/post/" + item.ID,
			UpdatedAt:          now,
		}); err != nil {
			return err
		}
	}
	logger.Info("distribution item published",
		"event", "distribution_item_published",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"platform_count", len(item.Platforms),
	)
	return nil
}

func (uc UseCase) Retry(ctx context.Context, cmd RetryCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(cmd.ItemID))
	if err != nil {
		return err
	}
	if item.InfluencerID != strings.TrimSpace(cmd.InfluencerID) {
		return domainerrors.ErrInvalidDistributionInput
	}
	if item.Status != entities.DistributionStatusFailed {
		return domainerrors.ErrInvalidStateTransition
	}
	item.Status = entities.DistributionStatusPublishing
	item.RetryCount++
	item.LastError = ""
	item.UpdatedAt = uc.Clock.Now().UTC()
	if err := uc.Repository.UpdateItem(ctx, item); err != nil {
		return err
	}
	logger.Warn("distribution retry requested",
		"event", "distribution_retry_requested",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"retry_count", item.RetryCount,
	)
	return nil
}
