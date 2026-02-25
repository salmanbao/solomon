package queries

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type ListCampaignsQuery struct {
	BrandID string
	Status  string
}

type ListCampaignsUseCase struct {
	Campaigns ports.CampaignRepository
	Logger    *slog.Logger
}

func (uc ListCampaignsUseCase) Execute(ctx context.Context, query ListCampaignsQuery) ([]entities.Campaign, error) {
	logger := application.ResolveLogger(uc.Logger)
	filter := ports.CampaignFilter{
		BrandID: strings.TrimSpace(query.BrandID),
	}
	if strings.TrimSpace(query.Status) != "" {
		filter.Status = entities.CampaignStatus(strings.TrimSpace(query.Status))
	}
	items, err := uc.Campaigns.ListCampaigns(ctx, filter)
	if err != nil {
		return nil, err
	}
	logger.Info("campaigns listed",
		"event", "campaigns_listed",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"count", len(items),
	)
	return items, nil
}
