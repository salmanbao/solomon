package commands

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type UpdateCampaignCommand struct {
	CampaignID       string
	ActorID          string
	Title            string
	Description      string
	Instructions     string
	Niche            string
	AllowedPlatforms []string
	RequiredHashtags []string
}

type UpdateCampaignUseCase struct {
	Campaigns ports.CampaignRepository
	Clock     ports.Clock
	Logger    *slog.Logger
}

func (uc UpdateCampaignUseCase) Execute(ctx context.Context, cmd UpdateCampaignCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(cmd.CampaignID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" || campaign.BrandID != strings.TrimSpace(cmd.ActorID) {
		return domainerrors.ErrInvalidCampaignInput
	}
	if !campaign.CanEdit() {
		return domainerrors.ErrCampaignNotEditable
	}

	campaign.Title = strings.TrimSpace(cmd.Title)
	campaign.Description = strings.TrimSpace(cmd.Description)
	campaign.Instructions = strings.TrimSpace(cmd.Instructions)
	campaign.Niche = strings.TrimSpace(cmd.Niche)
	campaign.AllowedPlatforms = append([]string(nil), cmd.AllowedPlatforms...)
	campaign.RequiredHashtags = append([]string(nil), cmd.RequiredHashtags...)
	campaign.UpdatedAt = uc.Clock.Now().UTC()

	if !campaign.ValidateBasics() {
		return domainerrors.ErrInvalidCampaignInput
	}
	if len(campaign.AllowedPlatforms) == 0 {
		return domainerrors.ErrInvalidCampaignInput
	}
	if err := uc.Campaigns.UpdateCampaign(ctx, campaign); err != nil {
		return err
	}

	logger.Info("campaign updated",
		"event", "campaign_updated",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
	)
	return nil
}
