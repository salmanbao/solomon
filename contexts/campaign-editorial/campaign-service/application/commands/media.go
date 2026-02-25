package commands

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type GenerateUploadURLCommand struct {
	CampaignID  string
	ActorID     string
	FileName    string
	FileSize    int64
	ContentType string
}

type GenerateUploadURLResult struct {
	MediaID   string
	UploadURL string
	ExpiresAt time.Time
	AssetPath string
}

type GenerateUploadURLUseCase struct {
	Campaigns ports.CampaignRepository
	Clock     ports.Clock
	IDGen     ports.IDGenerator
	Logger    *slog.Logger
}

func (uc GenerateUploadURLUseCase) Execute(ctx context.Context, cmd GenerateUploadURLCommand) (GenerateUploadURLResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(cmd.CampaignID))
	if err != nil {
		return GenerateUploadURLResult{}, err
	}
	if strings.TrimSpace(cmd.ActorID) == "" || campaign.BrandID != strings.TrimSpace(cmd.ActorID) {
		return GenerateUploadURLResult{}, domainerrors.ErrInvalidCampaignInput
	}
	if campaign.Status == entities.CampaignStatusCompleted {
		return GenerateUploadURLResult{}, domainerrors.ErrInvalidStateTransition
	}
	if cmd.FileSize <= 0 {
		return GenerateUploadURLResult{}, domainerrors.ErrInvalidCampaignInput
	}
	if cmd.FileSize > 500*1024*1024 {
		return GenerateUploadURLResult{}, domainerrors.ErrMediaFileTooLarge
	}
	if !isSupportedMediaContentType(cmd.ContentType) {
		return GenerateUploadURLResult{}, domainerrors.ErrUnsupportedMediaType
	}

	mediaID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return GenerateUploadURLResult{}, err
	}
	now := uc.Clock.Now().UTC()
	assetPath := fmt.Sprintf("campaigns/%s/source/%s-%s", campaign.CampaignID, mediaID, sanitizeFileName(cmd.FileName))
	result := GenerateUploadURLResult{
		MediaID:   mediaID,
		UploadURL: fmt.Sprintf("https://uploads.viralforge.local/%s", assetPath),
		ExpiresAt: now.Add(15 * time.Minute),
		AssetPath: assetPath,
	}

	logger.Info("campaign media upload url generated",
		"event", "campaign_media_upload_url_generated",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
		"media_id", mediaID,
	)
	return result, nil
}

type ConfirmMediaCommand struct {
	CampaignID  string
	ActorID     string
	MediaID     string
	AssetPath   string
	ContentType string
}

type ConfirmMediaUseCase struct {
	Campaigns ports.CampaignRepository
	Media     ports.MediaRepository
	Clock     ports.Clock
	Logger    *slog.Logger
}

func (uc ConfirmMediaUseCase) Execute(ctx context.Context, cmd ConfirmMediaCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(cmd.CampaignID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" || campaign.BrandID != strings.TrimSpace(cmd.ActorID) {
		return domainerrors.ErrInvalidCampaignInput
	}
	if strings.TrimSpace(cmd.MediaID) == "" || strings.TrimSpace(cmd.AssetPath) == "" {
		return domainerrors.ErrInvalidCampaignInput
	}
	if !isSupportedMediaContentType(cmd.ContentType) {
		return domainerrors.ErrUnsupportedMediaType
	}

	now := uc.Clock.Now().UTC()
	if existing, err := uc.Media.GetMedia(ctx, strings.TrimSpace(cmd.MediaID)); err == nil {
		if existing.CampaignID == campaign.CampaignID &&
			existing.AssetPath == strings.TrimSpace(cmd.AssetPath) &&
			existing.ContentType == strings.TrimSpace(cmd.ContentType) {
			return nil
		}
		return domainerrors.ErrMediaAlreadyConfirmed
	}
	items, err := uc.Media.ListMediaByCampaign(ctx, campaign.CampaignID)
	if err != nil {
		return err
	}
	if len(items) >= 10 {
		return domainerrors.ErrMediaLimitReached
	}
	media := entities.Media{
		MediaID:     strings.TrimSpace(cmd.MediaID),
		CampaignID:  campaign.CampaignID,
		AssetPath:   strings.TrimSpace(cmd.AssetPath),
		ContentType: strings.TrimSpace(cmd.ContentType),
		Status:      entities.MediaStatusReady,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.Media.AddMedia(ctx, media); err != nil {
		return err
	}

	logger.Info("campaign media confirmed",
		"event", "campaign_media_confirmed",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
		"media_id", media.MediaID,
	)
	return nil
}

func sanitizeFileName(fileName string) string {
	value := strings.TrimSpace(fileName)
	if value == "" {
		return "asset.bin"
	}
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	return value
}

func isSupportedMediaContentType(contentType string) bool {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "video/mp4", "video/quicktime", "video/x-msvideo",
		"image/jpeg", "image/png", "image/gif",
		"audio/mpeg", "audio/wav", "audio/aac":
		return true
	default:
		return false
	}
}
