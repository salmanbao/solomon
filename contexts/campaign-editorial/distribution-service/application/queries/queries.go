package queries

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

type UseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	Logger     *slog.Logger
}

func (uc UseCase) GetItem(ctx context.Context, itemID string) (entities.DistributionItem, error) {
	logger := application.ResolveLogger(uc.Logger)
	normalizedItemID := strings.TrimSpace(itemID)
	item, err := uc.Repository.GetItem(ctx, normalizedItemID)
	if err != nil {
		logger.Warn("distribution query get item failed",
			"event", "distribution_query_get_item_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", normalizedItemID,
			"error", err.Error(),
		)
		return entities.DistributionItem{}, err
	}
	return item, nil
}

func (uc UseCase) Preview(ctx context.Context, itemID string) (string, time.Time, error) {
	logger := application.ResolveLogger(uc.Logger)
	normalizedItemID := strings.TrimSpace(itemID)
	item, err := uc.Repository.GetItem(ctx, normalizedItemID)
	if err != nil {
		logger.Warn("distribution query preview failed",
			"event", "distribution_query_preview_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"item_id", normalizedItemID,
			"error", err.Error(),
		)
		return "", time.Time{}, err
	}
	expiresAt := uc.Clock.Now().UTC().Add(5 * time.Minute)
	url := "https://preview.viralforge.local/distribution/" + item.ID
	logger.Info("distribution query preview generated",
		"event", "distribution_query_preview_generated",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"item_id", item.ID,
		"expires_at", expiresAt.Format(time.RFC3339),
	)
	return url, expiresAt, nil
}

func (uc UseCase) Queue(ctx context.Context, influencerID string) ([]entities.DistributionItem, error) {
	logger := application.ResolveLogger(uc.Logger)
	normalizedInfluencerID := strings.TrimSpace(influencerID)
	items, err := uc.Repository.ListItemsByInfluencer(ctx, normalizedInfluencerID)
	if err != nil {
		logger.Warn("distribution query queue failed",
			"event", "distribution_query_queue_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "application",
			"influencer_id", normalizedInfluencerID,
			"error", err.Error(),
		)
		return nil, err
	}
	logger.Info("distribution query queue listed",
		"event", "distribution_query_queue_listed",
		"module", "campaign-editorial/distribution-service",
		"layer", "application",
		"influencer_id", normalizedInfluencerID,
		"item_count", len(items),
	)
	return items, nil
}
