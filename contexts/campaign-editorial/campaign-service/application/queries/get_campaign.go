package queries

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type GetCampaignUseCase struct {
	Campaigns ports.CampaignRepository
	Logger    *slog.Logger
}

func (uc GetCampaignUseCase) Execute(ctx context.Context, campaignID string) (entities.Campaign, error) {
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		return entities.Campaign{}, err
	}
	return campaign, nil
}

type ListMediaUseCase struct {
	Media  ports.MediaRepository
	Logger *slog.Logger
}

func (uc ListMediaUseCase) Execute(ctx context.Context, campaignID string) ([]entities.Media, error) {
	return uc.Media.ListMediaByCampaign(ctx, strings.TrimSpace(campaignID))
}

type GetAnalyticsResult struct {
	SubmissionCount int
	TotalViews      int64
	BudgetSpent     float64
	BudgetRemaining float64
}

type GetAnalyticsUseCase struct {
	Campaigns ports.CampaignRepository
	Clock     ports.Clock
	Logger    *slog.Logger
}

func (uc GetAnalyticsUseCase) Execute(ctx context.Context, campaignID string) (GetAnalyticsResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		return GetAnalyticsResult{}, err
	}
	viewsApprox := int64((campaign.BudgetSpent / campaign.RatePer1KViews) * 1000)
	logger.Debug("campaign analytics fetched",
		"event", "campaign_analytics_fetched",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaignID,
	)
	return GetAnalyticsResult{
		SubmissionCount: 0,
		TotalViews:      viewsApprox,
		BudgetSpent:     campaign.BudgetSpent,
		BudgetRemaining: campaign.BudgetRemaining,
	}, nil
}

type ExportAnalyticsUseCase struct {
	Clock  ports.Clock
	Logger *slog.Logger
}

func (uc ExportAnalyticsUseCase) Execute(_ context.Context, campaignID string) (string, time.Time) {
	now := uc.Clock.Now().UTC()
	expiresAt := now.Add(30 * time.Minute)
	return "https://exports.viralforge.local/campaigns/" + strings.TrimSpace(campaignID) + "/analytics.csv", expiresAt
}
