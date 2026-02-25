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
	BrandID           string
	IdempotencyKey    string
	Title             string
	Description       string
	Instructions      string
	Niche             string
	AllowedPlatforms  []string
	RequiredHashtags  []string
	RequiredTags      []string
	OptionalHashtags  []string
	UsageGuidelines   string
	DosAndDonts       string
	CampaignType      string
	DeadlineAt        *time.Time
	TargetSubmissions *int
	BannerImageURL    string
	ExternalURL       string
	BudgetTotal       float64
	RatePer1KViews    float64
}

type CreateCampaignUseCase struct {
	Campaigns      ports.CampaignRepository
	Idempotency    ports.IdempotencyStore
	Outbox         ports.OutboxWriter
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
	CampaignID              string                  `json:"campaign_id"`
	BrandID                 string                  `json:"brand_id"`
	Title                   string                  `json:"title"`
	Description             string                  `json:"description"`
	Instructions            string                  `json:"instructions"`
	Niche                   string                  `json:"niche"`
	AllowedPlatforms        []string                `json:"allowed_platforms"`
	RequiredHashtags        []string                `json:"required_hashtags"`
	RequiredTags            []string                `json:"required_tags"`
	OptionalHashtags        []string                `json:"optional_hashtags"`
	UsageGuidelines         string                  `json:"usage_guidelines"`
	DosAndDonts             string                  `json:"dos_and_donts"`
	CampaignType            entities.CampaignType   `json:"campaign_type"`
	DeadlineAt              *time.Time              `json:"deadline_at"`
	TargetSubmissions       *int                    `json:"target_submissions"`
	BannerImageURL          string                  `json:"banner_image_url"`
	ExternalURL             string                  `json:"external_url"`
	BudgetTotal             float64                 `json:"budget_total"`
	BudgetSpent             float64                 `json:"budget_spent"`
	BudgetReserved          float64                 `json:"budget_reserved"`
	BudgetRemaining         float64                 `json:"budget_remaining"`
	RatePer1KViews          float64                 `json:"rate_per_1k_views"`
	SubmissionCount         int                     `json:"submission_count"`
	ApprovedSubmissionCount int                     `json:"approved_submission_count"`
	TotalViews              int64                   `json:"total_views"`
	Status                  entities.CampaignStatus `json:"status"`
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
				CampaignID:              payload.CampaignID,
				BrandID:                 payload.BrandID,
				Title:                   payload.Title,
				Description:             payload.Description,
				Instructions:            payload.Instructions,
				Niche:                   payload.Niche,
				AllowedPlatforms:        append([]string(nil), payload.AllowedPlatforms...),
				RequiredHashtags:        append([]string(nil), payload.RequiredHashtags...),
				RequiredTags:            append([]string(nil), payload.RequiredTags...),
				OptionalHashtags:        append([]string(nil), payload.OptionalHashtags...),
				UsageGuidelines:         payload.UsageGuidelines,
				DosAndDonts:             payload.DosAndDonts,
				CampaignType:            payload.CampaignType,
				DeadlineAt:              payload.DeadlineAt,
				TargetSubmissions:       payload.TargetSubmissions,
				BannerImageURL:          payload.BannerImageURL,
				ExternalURL:             payload.ExternalURL,
				BudgetTotal:             payload.BudgetTotal,
				BudgetSpent:             payload.BudgetSpent,
				BudgetReserved:          payload.BudgetReserved,
				BudgetRemaining:         payload.BudgetRemaining,
				RatePer1KViews:          payload.RatePer1KViews,
				SubmissionCount:         payload.SubmissionCount,
				ApprovedSubmissionCount: payload.ApprovedSubmissionCount,
				TotalViews:              payload.TotalViews,
				Status:                  payload.Status,
			},
			Replayed: true,
		}, nil
	}

	campaignID, err := uc.IDGenerator.NewID(ctx)
	if err != nil {
		return CreateCampaignResult{}, err
	}

	campaign := entities.Campaign{
		CampaignID:              campaignID,
		BrandID:                 strings.TrimSpace(cmd.BrandID),
		Title:                   strings.TrimSpace(cmd.Title),
		Description:             strings.TrimSpace(cmd.Description),
		Instructions:            strings.TrimSpace(cmd.Instructions),
		Niche:                   strings.TrimSpace(cmd.Niche),
		AllowedPlatforms:        append([]string(nil), cmd.AllowedPlatforms...),
		RequiredHashtags:        append([]string(nil), cmd.RequiredHashtags...),
		RequiredTags:            append([]string(nil), cmd.RequiredTags...),
		OptionalHashtags:        append([]string(nil), cmd.OptionalHashtags...),
		UsageGuidelines:         strings.TrimSpace(cmd.UsageGuidelines),
		DosAndDonts:             strings.TrimSpace(cmd.DosAndDonts),
		CampaignType:            entities.CampaignType(strings.TrimSpace(cmd.CampaignType)),
		DeadlineAt:              cmd.DeadlineAt,
		TargetSubmissions:       cmd.TargetSubmissions,
		BannerImageURL:          strings.TrimSpace(cmd.BannerImageURL),
		ExternalURL:             strings.TrimSpace(cmd.ExternalURL),
		BudgetTotal:             cmd.BudgetTotal,
		BudgetSpent:             0,
		BudgetReserved:          0,
		BudgetRemaining:         cmd.BudgetTotal,
		RatePer1KViews:          cmd.RatePer1KViews,
		SubmissionCount:         0,
		ApprovedSubmissionCount: 0,
		TotalViews:              0,
		Status:                  entities.CampaignStatusDraft,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if campaign.CampaignType == "" {
		campaign.CampaignType = entities.CampaignTypeUGCCreation
	}
	if !campaign.ValidateBasics(now) || campaign.BrandID == "" {
		return CreateCampaignResult{}, domainerrors.ErrInvalidCampaignInput
	}
	if !entities.DeadlineAtLeastSevenDays(campaign.DeadlineAt, now) {
		return CreateCampaignResult{}, domainerrors.ErrDeadlineTooSoon
	}

	if err := uc.Campaigns.CreateCampaign(ctx, campaign); err != nil {
		return CreateCampaignResult{}, err
	}

	payload := createCampaignReplayPayload{
		CampaignID:              campaign.CampaignID,
		BrandID:                 campaign.BrandID,
		Title:                   campaign.Title,
		Description:             campaign.Description,
		Instructions:            campaign.Instructions,
		Niche:                   campaign.Niche,
		AllowedPlatforms:        append([]string(nil), campaign.AllowedPlatforms...),
		RequiredHashtags:        append([]string(nil), campaign.RequiredHashtags...),
		RequiredTags:            append([]string(nil), campaign.RequiredTags...),
		OptionalHashtags:        append([]string(nil), campaign.OptionalHashtags...),
		UsageGuidelines:         campaign.UsageGuidelines,
		DosAndDonts:             campaign.DosAndDonts,
		CampaignType:            campaign.CampaignType,
		DeadlineAt:              campaign.DeadlineAt,
		TargetSubmissions:       campaign.TargetSubmissions,
		BannerImageURL:          campaign.BannerImageURL,
		ExternalURL:             campaign.ExternalURL,
		BudgetTotal:             campaign.BudgetTotal,
		BudgetSpent:             campaign.BudgetSpent,
		BudgetReserved:          campaign.BudgetReserved,
		BudgetRemaining:         campaign.BudgetRemaining,
		RatePer1KViews:          campaign.RatePer1KViews,
		SubmissionCount:         campaign.SubmissionCount,
		ApprovedSubmissionCount: campaign.ApprovedSubmissionCount,
		TotalViews:              campaign.TotalViews,
		Status:                  campaign.Status,
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
	if uc.Outbox != nil {
		eventID, err := uc.IDGenerator.NewID(ctx)
		if err != nil {
			return CreateCampaignResult{}, err
		}
		envelope, err := newCampaignEnvelope(
			eventID,
			"campaign.created",
			campaign.CampaignID,
			now,
			map[string]any{
				"campaign_id": campaign.CampaignID,
				"brand_id":    campaign.BrandID,
				"status":      string(campaign.Status),
			},
		)
		if err != nil {
			return CreateCampaignResult{}, err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			return CreateCampaignResult{}, err
		}
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
		"brand_id":           strings.TrimSpace(cmd.BrandID),
		"title":              strings.TrimSpace(cmd.Title),
		"description":        strings.TrimSpace(cmd.Description),
		"instructions":       strings.TrimSpace(cmd.Instructions),
		"niche":              strings.TrimSpace(cmd.Niche),
		"allowed_platforms":  append([]string(nil), cmd.AllowedPlatforms...),
		"required_hashtags":  append([]string(nil), cmd.RequiredHashtags...),
		"required_tags":      append([]string(nil), cmd.RequiredTags...),
		"optional_hashtags":  append([]string(nil), cmd.OptionalHashtags...),
		"usage_guidelines":   strings.TrimSpace(cmd.UsageGuidelines),
		"dos_and_donts":      strings.TrimSpace(cmd.DosAndDonts),
		"campaign_type":      strings.TrimSpace(cmd.CampaignType),
		"target_submissions": cmd.TargetSubmissions,
		"banner_image_url":   strings.TrimSpace(cmd.BannerImageURL),
		"external_url":       strings.TrimSpace(cmd.ExternalURL),
		"budget_total":       cmd.BudgetTotal,
		"rate_per_1k_views":  cmd.RatePer1KViews,
	}
	if cmd.DeadlineAt != nil {
		payload["deadline_at"] = cmd.DeadlineAt.UTC().Format(time.RFC3339Nano)
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
