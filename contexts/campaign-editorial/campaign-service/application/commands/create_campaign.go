package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type CreateCampaignCommand struct {
	BrandID          string
	IdempotencyKey   string
	Title            string
	Description      string
	Instructions     string
	Niche            string
	AllowedPlatforms []string
	RequiredHashtags []string
	BudgetTotal      float64
	RatePer1KViews   float64
}

type CreateCampaignUseCase struct {
	Campaigns      ports.CampaignRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

type CreateCampaignResult struct {
	Campaign entities.Campaign
	Replayed bool
}

type createCampaignReplayPayload struct {
	CampaignID       string                  `json:"campaign_id"`
	BrandID          string                  `json:"brand_id"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	Instructions     string                  `json:"instructions"`
	Niche            string                  `json:"niche"`
	AllowedPlatforms []string                `json:"allowed_platforms"`
	RequiredHashtags []string                `json:"required_hashtags"`
	BudgetTotal      float64                 `json:"budget_total"`
	BudgetSpent      float64                 `json:"budget_spent"`
	BudgetRemaining  float64                 `json:"budget_remaining"`
	RatePer1KViews   float64                 `json:"rate_per_1k_views"`
	Status           entities.CampaignStatus `json:"status"`
}

func (uc CreateCampaignUseCase) Execute(ctx context.Context, cmd CreateCampaignCommand) (CreateCampaignResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return CreateCampaignResult{}, domainerrors.ErrIdempotencyKeyRequired
	}

	now := uc.Clock.Now().UTC()
	requestHash := hashCreateCampaignCommand(cmd)
	if record, found, err := uc.Idempotency.GetRecord(ctx, cmd.IdempotencyKey, now); err != nil {
		return CreateCampaignResult{}, err
	} else if found {
		if record.RequestHash != requestHash {
			return CreateCampaignResult{}, domainerrors.ErrIdempotencyKeyConflict
		}
		var payload createCampaignReplayPayload
		if err := json.Unmarshal(record.ResponsePayload, &payload); err != nil {
			return CreateCampaignResult{}, err
		}
		return CreateCampaignResult{
			Campaign: entities.Campaign{
				CampaignID:       payload.CampaignID,
				BrandID:          payload.BrandID,
				Title:            payload.Title,
				Description:      payload.Description,
				Instructions:     payload.Instructions,
				Niche:            payload.Niche,
				AllowedPlatforms: append([]string(nil), payload.AllowedPlatforms...),
				RequiredHashtags: append([]string(nil), payload.RequiredHashtags...),
				BudgetTotal:      payload.BudgetTotal,
				BudgetSpent:      payload.BudgetSpent,
				BudgetRemaining:  payload.BudgetRemaining,
				RatePer1KViews:   payload.RatePer1KViews,
				Status:           payload.Status,
			},
			Replayed: true,
		}, nil
	}

	campaignID, err := uc.IDGenerator.NewID(ctx)
	if err != nil {
		return CreateCampaignResult{}, err
	}

	campaign := entities.Campaign{
		CampaignID:       campaignID,
		BrandID:          strings.TrimSpace(cmd.BrandID),
		Title:            strings.TrimSpace(cmd.Title),
		Description:      strings.TrimSpace(cmd.Description),
		Instructions:     strings.TrimSpace(cmd.Instructions),
		Niche:            strings.TrimSpace(cmd.Niche),
		AllowedPlatforms: append([]string(nil), cmd.AllowedPlatforms...),
		RequiredHashtags: append([]string(nil), cmd.RequiredHashtags...),
		BudgetTotal:      cmd.BudgetTotal,
		BudgetSpent:      0,
		BudgetRemaining:  cmd.BudgetTotal,
		RatePer1KViews:   cmd.RatePer1KViews,
		Status:           entities.CampaignStatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if !campaign.ValidateBasics() || campaign.BrandID == "" {
		return CreateCampaignResult{}, domainerrors.ErrInvalidCampaignInput
	}
	if len(campaign.AllowedPlatforms) == 0 {
		return CreateCampaignResult{}, domainerrors.ErrInvalidCampaignInput
	}

	if err := uc.Campaigns.CreateCampaign(ctx, campaign); err != nil {
		return CreateCampaignResult{}, err
	}

	payload := createCampaignReplayPayload{
		CampaignID:       campaign.CampaignID,
		BrandID:          campaign.BrandID,
		Title:            campaign.Title,
		Description:      campaign.Description,
		Instructions:     campaign.Instructions,
		Niche:            campaign.Niche,
		AllowedPlatforms: append([]string(nil), campaign.AllowedPlatforms...),
		RequiredHashtags: append([]string(nil), campaign.RequiredHashtags...),
		BudgetTotal:      campaign.BudgetTotal,
		BudgetSpent:      campaign.BudgetSpent,
		BudgetRemaining:  campaign.BudgetRemaining,
		RatePer1KViews:   campaign.RatePer1KViews,
		Status:           campaign.Status,
	}
	serialized, err := json.Marshal(payload)
	if err != nil {
		return CreateCampaignResult{}, err
	}
	if err := uc.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             cmd.IdempotencyKey,
		RequestHash:     requestHash,
		ResponsePayload: serialized,
		ExpiresAt:       now.Add(uc.IdempotencyTTL),
	}); err != nil {
		return CreateCampaignResult{}, err
	}

	logger.Info("campaign created",
		"event", "campaign_created",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
		"brand_id", campaign.BrandID,
	)
	return CreateCampaignResult{Campaign: campaign}, nil
}

func hashCreateCampaignCommand(cmd CreateCampaignCommand) string {
	payload := map[string]any{
		"brand_id":          strings.TrimSpace(cmd.BrandID),
		"title":             strings.TrimSpace(cmd.Title),
		"description":       strings.TrimSpace(cmd.Description),
		"instructions":      strings.TrimSpace(cmd.Instructions),
		"niche":             strings.TrimSpace(cmd.Niche),
		"allowed_platforms": append([]string(nil), cmd.AllowedPlatforms...),
		"required_hashtags": append([]string(nil), cmd.RequiredHashtags...),
		"budget_total":      cmd.BudgetTotal,
		"rate_per_1k_views": cmd.RatePer1KViews,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
