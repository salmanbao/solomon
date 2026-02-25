package commands

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type UpdateCampaignCommand struct {
	CampaignID        string
	ActorID           string
	Title             *string
	Description       *string
	Instructions      *string
	Niche             *string
	AllowedPlatforms  *[]string
	RequiredHashtags  *[]string
	RequiredTags      *[]string
	OptionalHashtags  *[]string
	UsageGuidelines   *string
	DosAndDonts       *string
	CampaignType      *string
	DeadlineAt        *time.Time
	TargetSubmissions *int
	BannerImageURL    *string
	ExternalURL       *string
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
		if campaign.Status == entities.CampaignStatusActive {
			if !isOnlyDeadlineUpdate(cmd) {
				return domainerrors.ErrCampaignEditRestricted
			}
			if cmd.DeadlineAt == nil {
				return domainerrors.ErrCampaignEditRestricted
			}
			if campaign.DeadlineAt != nil && !cmd.DeadlineAt.After(*campaign.DeadlineAt) {
				return domainerrors.ErrCampaignEditRestricted
			}
			if !entities.DeadlineAtLeastSevenDays(cmd.DeadlineAt, uc.Clock.Now().UTC()) {
				return domainerrors.ErrDeadlineTooSoon
			}
			campaign.DeadlineAt = cmd.DeadlineAt
			campaign.UpdatedAt = uc.Clock.Now().UTC()
			if err := uc.Campaigns.UpdateCampaign(ctx, campaign); err != nil {
				return err
			}
			return nil
		}
		return domainerrors.ErrCampaignNotEditable
	}

	if campaign.Status == entities.CampaignStatusPaused {
		if cmd.Niche != nil || cmd.AllowedPlatforms != nil || cmd.CampaignType != nil {
			return domainerrors.ErrCampaignEditRestricted
		}
	}

	if cmd.Title != nil {
		campaign.Title = strings.TrimSpace(*cmd.Title)
	}
	if cmd.Description != nil {
		campaign.Description = strings.TrimSpace(*cmd.Description)
	}
	if cmd.Instructions != nil {
		campaign.Instructions = strings.TrimSpace(*cmd.Instructions)
	}
	if cmd.Niche != nil {
		campaign.Niche = strings.TrimSpace(*cmd.Niche)
	}
	if cmd.AllowedPlatforms != nil {
		campaign.AllowedPlatforms = append([]string(nil), (*cmd.AllowedPlatforms)...)
	}
	if cmd.RequiredHashtags != nil {
		campaign.RequiredHashtags = append([]string(nil), (*cmd.RequiredHashtags)...)
	}
	if cmd.RequiredTags != nil {
		campaign.RequiredTags = append([]string(nil), (*cmd.RequiredTags)...)
	}
	if cmd.OptionalHashtags != nil {
		campaign.OptionalHashtags = append([]string(nil), (*cmd.OptionalHashtags)...)
	}
	if cmd.UsageGuidelines != nil {
		campaign.UsageGuidelines = strings.TrimSpace(*cmd.UsageGuidelines)
	}
	if cmd.DosAndDonts != nil {
		campaign.DosAndDonts = strings.TrimSpace(*cmd.DosAndDonts)
	}
	if cmd.CampaignType != nil {
		campaign.CampaignType = entities.CampaignType(strings.TrimSpace(*cmd.CampaignType))
	}
	if cmd.DeadlineAt != nil {
		campaign.DeadlineAt = cmd.DeadlineAt
	}
	if cmd.TargetSubmissions != nil {
		campaign.TargetSubmissions = cmd.TargetSubmissions
	}
	if cmd.BannerImageURL != nil {
		campaign.BannerImageURL = strings.TrimSpace(*cmd.BannerImageURL)
	}
	if cmd.ExternalURL != nil {
		campaign.ExternalURL = strings.TrimSpace(*cmd.ExternalURL)
	}
	campaign.UpdatedAt = uc.Clock.Now().UTC()

	if !campaign.ValidateBasics(campaign.UpdatedAt) {
		return domainerrors.ErrInvalidCampaignInput
	}
	if !entities.DeadlineAtLeastSevenDays(campaign.DeadlineAt, campaign.UpdatedAt) {
		return domainerrors.ErrDeadlineTooSoon
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

func isOnlyDeadlineUpdate(cmd UpdateCampaignCommand) bool {
	return cmd.Title == nil &&
		cmd.Description == nil &&
		cmd.Instructions == nil &&
		cmd.Niche == nil &&
		cmd.AllowedPlatforms == nil &&
		cmd.RequiredHashtags == nil &&
		cmd.RequiredTags == nil &&
		cmd.OptionalHashtags == nil &&
		cmd.UsageGuidelines == nil &&
		cmd.DosAndDonts == nil &&
		cmd.CampaignType == nil &&
		cmd.TargetSubmissions == nil &&
		cmd.BannerImageURL == nil &&
		cmd.ExternalURL == nil &&
		cmd.DeadlineAt != nil
}
