package httpadapter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/application/commands"
	"solomon/contexts/campaign-editorial/campaign-service/application/queries"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/campaign-service/transport/http"
)

type Handler struct {
	CreateCampaign    commands.CreateCampaignUseCase
	UpdateCampaign    commands.UpdateCampaignUseCase
	ChangeStatus      commands.ChangeStatusUseCase
	IncreaseBudget    commands.IncreaseBudgetUseCase
	GenerateUploadURL commands.GenerateUploadURLUseCase
	ConfirmMedia      commands.ConfirmMediaUseCase
	ListCampaigns     queries.ListCampaignsUseCase
	GetCampaign       queries.GetCampaignUseCase
	ListMedia         queries.ListMediaUseCase
	GetAnalytics      queries.GetAnalyticsUseCase
	ExportAnalytics   queries.ExportAnalyticsUseCase
	Logger            *slog.Logger
}

func (h Handler) CreateCampaignHandler(
	ctx context.Context,
	userID string,
	idempotencyKey string,
	req httptransport.CreateCampaignRequest,
) (httptransport.CreateCampaignResponse, error) {
	deadline, err := parseDeadline(req.Deadline)
	if err != nil {
		return httptransport.CreateCampaignResponse{}, domainerrors.ErrInvalidCampaignInput
	}
	result, err := h.CreateCampaign.Execute(ctx, commands.CreateCampaignCommand{
		BrandID:           userID,
		IdempotencyKey:    idempotencyKey,
		Title:             req.Title,
		Description:       req.Description,
		Instructions:      req.Instructions,
		Niche:             req.Niche,
		AllowedPlatforms:  append([]string(nil), req.AllowedPlatforms...),
		RequiredHashtags:  append([]string(nil), req.RequiredHashtags...),
		RequiredTags:      append([]string(nil), req.RequiredTags...),
		OptionalHashtags:  append([]string(nil), req.OptionalHashtags...),
		UsageGuidelines:   req.UsageGuidelines,
		DosAndDonts:       req.DosAndDonts,
		CampaignType:      req.CampaignType,
		DeadlineAt:        deadline,
		TargetSubmissions: req.TargetSubmissions,
		BannerImageURL:    req.BannerImageURL,
		ExternalURL:       req.ExternalURL,
		BudgetTotal:       req.BudgetTotal,
		RatePer1KViews:    req.RatePer1KViews,
	})
	if err != nil {
		return httptransport.CreateCampaignResponse{}, err
	}
	return httptransport.CreateCampaignResponse{
		Campaign: mapCampaign(result.Campaign),
		Replayed: result.Replayed,
	}, nil
}

func (h Handler) ListCampaignsHandler(ctx context.Context, userID string, status string) (httptransport.ListCampaignsResponse, error) {
	items, err := h.ListCampaigns.Execute(ctx, queries.ListCampaignsQuery{
		BrandID: userID,
		Status:  status,
	})
	if err != nil {
		return httptransport.ListCampaignsResponse{}, err
	}
	result := make([]httptransport.CampaignDTO, 0, len(items))
	for _, item := range items {
		result = append(result, mapCampaign(item))
	}
	return httptransport.ListCampaignsResponse{Items: result}, nil
}

func (h Handler) GetCampaignHandler(ctx context.Context, campaignID string) (httptransport.GetCampaignResponse, error) {
	item, err := h.GetCampaign.Execute(ctx, campaignID)
	if err != nil {
		return httptransport.GetCampaignResponse{}, err
	}
	return httptransport.GetCampaignResponse{Campaign: mapCampaign(item)}, nil
}

func (h Handler) UpdateCampaignHandler(
	ctx context.Context,
	userID string,
	campaignID string,
	req httptransport.UpdateCampaignRequest,
) error {
	deadline, err := parseOptionalDeadline(req.Deadline)
	if err != nil {
		return domainerrors.ErrInvalidCampaignInput
	}
	return h.UpdateCampaign.Execute(ctx, commands.UpdateCampaignCommand{
		CampaignID:        campaignID,
		ActorID:           userID,
		Title:             req.Title,
		Description:       req.Description,
		Instructions:      req.Instructions,
		Niche:             req.Niche,
		AllowedPlatforms:  req.AllowedPlatforms,
		RequiredHashtags:  req.RequiredHashtags,
		RequiredTags:      req.RequiredTags,
		OptionalHashtags:  req.OptionalHashtags,
		UsageGuidelines:   req.UsageGuidelines,
		DosAndDonts:       req.DosAndDonts,
		CampaignType:      req.CampaignType,
		DeadlineAt:        deadline,
		TargetSubmissions: req.TargetSubmissions,
		BannerImageURL:    req.BannerImageURL,
		ExternalURL:       req.ExternalURL,
	})
}

func (h Handler) LaunchCampaignHandler(ctx context.Context, userID string, campaignID string, reason string) error {
	return h.ChangeStatus.Execute(ctx, commands.ChangeStatusCommand{
		CampaignID: campaignID,
		ActorID:    userID,
		Action:     commands.StatusActionLaunch,
		Reason:     reason,
	})
}

func (h Handler) PauseCampaignHandler(ctx context.Context, userID string, campaignID string, reason string) error {
	return h.ChangeStatus.Execute(ctx, commands.ChangeStatusCommand{
		CampaignID: campaignID,
		ActorID:    userID,
		Action:     commands.StatusActionPause,
		Reason:     reason,
	})
}

func (h Handler) ResumeCampaignHandler(ctx context.Context, userID string, campaignID string, reason string) error {
	return h.ChangeStatus.Execute(ctx, commands.ChangeStatusCommand{
		CampaignID: campaignID,
		ActorID:    userID,
		Action:     commands.StatusActionResume,
		Reason:     reason,
	})
}

func (h Handler) CompleteCampaignHandler(ctx context.Context, userID string, campaignID string, reason string) error {
	return h.ChangeStatus.Execute(ctx, commands.ChangeStatusCommand{
		CampaignID: campaignID,
		ActorID:    userID,
		Action:     commands.StatusActionComplete,
		Reason:     reason,
	})
}

func (h Handler) IncreaseBudgetHandler(
	ctx context.Context,
	userID string,
	campaignID string,
	req httptransport.IncreaseBudgetRequest,
) error {
	return h.IncreaseBudget.Execute(ctx, commands.IncreaseBudgetCommand{
		CampaignID: campaignID,
		ActorID:    userID,
		Amount:     req.Amount,
		Reason:     req.Reason,
	})
}

func (h Handler) GenerateUploadURLHandler(
	ctx context.Context,
	userID string,
	campaignID string,
	req httptransport.GenerateUploadURLRequest,
) (httptransport.GenerateUploadURLResponse, error) {
	result, err := h.GenerateUploadURL.Execute(ctx, commands.GenerateUploadURLCommand{
		CampaignID:  campaignID,
		ActorID:     userID,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
	})
	if err != nil {
		return httptransport.GenerateUploadURLResponse{}, err
	}
	return httptransport.GenerateUploadURLResponse{
		MediaID:   result.MediaID,
		UploadURL: result.UploadURL,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
		AssetPath: result.AssetPath,
	}, nil
}

func (h Handler) ConfirmMediaHandler(
	ctx context.Context,
	userID string,
	campaignID string,
	mediaID string,
	req httptransport.ConfirmMediaRequest,
) error {
	return h.ConfirmMedia.Execute(ctx, commands.ConfirmMediaCommand{
		CampaignID:  campaignID,
		ActorID:     userID,
		MediaID:     mediaID,
		AssetPath:   req.AssetPath,
		ContentType: req.ContentType,
	})
}

func (h Handler) ListMediaHandler(ctx context.Context, campaignID string) (httptransport.ListMediaResponse, error) {
	items, err := h.ListMedia.Execute(ctx, campaignID)
	if err != nil {
		return httptransport.ListMediaResponse{}, err
	}
	result := make([]httptransport.CampaignMediaDTO, 0, len(items))
	for _, item := range items {
		result = append(result, httptransport.CampaignMediaDTO{
			MediaID:     item.MediaID,
			CampaignID:  item.CampaignID,
			AssetPath:   item.AssetPath,
			ContentType: item.ContentType,
			Status:      string(item.Status),
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return httptransport.ListMediaResponse{Items: result}, nil
}

func (h Handler) GetAnalyticsHandler(ctx context.Context, campaignID string) (httptransport.AnalyticsResponse, error) {
	result, err := h.GetAnalytics.Execute(ctx, campaignID)
	if err != nil {
		return httptransport.AnalyticsResponse{}, err
	}
	return httptransport.AnalyticsResponse{
		CampaignID:      campaignID,
		SubmissionCount: result.SubmissionCount,
		TotalViews:      result.TotalViews,
		BudgetSpent:     result.BudgetSpent,
		BudgetRemaining: result.BudgetRemaining,
	}, nil
}

func (h Handler) ExportAnalyticsHandler(ctx context.Context, campaignID string) (httptransport.ExportAnalyticsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	url, expiresAt := h.ExportAnalytics.Execute(ctx, campaignID)
	logger.Info("campaign analytics export generated",
		"event", "campaign_analytics_export_generated",
		"module", "campaign-editorial/campaign-service",
		"layer", "transport",
		"campaign_id", campaignID,
	)
	return httptransport.ExportAnalyticsResponse{
		DownloadURL: url,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}

func mapCampaign(item entities.Campaign) httptransport.CampaignDTO {
	result := httptransport.CampaignDTO{
		CampaignID:              item.CampaignID,
		BrandID:                 item.BrandID,
		Title:                   item.Title,
		Description:             item.Description,
		Instructions:            item.Instructions,
		Niche:                   item.Niche,
		CampaignType:            string(item.CampaignType),
		AllowedPlatforms:        append([]string(nil), item.AllowedPlatforms...),
		RequiredHashtags:        append([]string(nil), item.RequiredHashtags...),
		RequiredTags:            append([]string(nil), item.RequiredTags...),
		OptionalHashtags:        append([]string(nil), item.OptionalHashtags...),
		UsageGuidelines:         item.UsageGuidelines,
		DosAndDonts:             item.DosAndDonts,
		TargetSubmissions:       item.TargetSubmissions,
		BannerImageURL:          item.BannerImageURL,
		ExternalURL:             item.ExternalURL,
		BudgetTotal:             item.BudgetTotal,
		BudgetSpent:             item.BudgetSpent,
		BudgetReserved:          item.BudgetReserved,
		BudgetRemaining:         item.BudgetRemaining,
		RatePer1KViews:          item.RatePer1KViews,
		SubmissionCount:         item.SubmissionCount,
		ApprovedSubmissionCount: item.ApprovedSubmissionCount,
		TotalViews:              item.TotalViews,
		Status:                  string(item.Status),
		CreatedAt:               item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               item.UpdatedAt.Format(time.RFC3339),
	}
	if item.DeadlineAt != nil {
		result.Deadline = item.DeadlineAt.UTC().Format(time.RFC3339)
	}
	if item.LaunchedAt != nil {
		result.LaunchedAt = item.LaunchedAt.UTC().Format(time.RFC3339)
	}
	if item.CompletedAt != nil {
		result.CompletedAt = item.CompletedAt.UTC().Format(time.RFC3339)
	}
	return result
}

func parseDeadline(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("parse deadline: %w", err)
	}
	utc := parsed.UTC()
	return &utc, nil
}

func parseOptionalDeadline(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("parse deadline: %w", err)
	}
	utc := parsed.UTC()
	return &utc, nil
}
