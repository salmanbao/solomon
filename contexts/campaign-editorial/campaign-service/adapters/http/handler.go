package httpadapter

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/application/commands"
	"solomon/contexts/campaign-editorial/campaign-service/application/queries"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
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
	result, err := h.CreateCampaign.Execute(ctx, commands.CreateCampaignCommand{
		BrandID:          userID,
		IdempotencyKey:   idempotencyKey,
		Title:            req.Title,
		Description:      req.Description,
		Instructions:     req.Instructions,
		Niche:            req.Niche,
		AllowedPlatforms: append([]string(nil), req.AllowedPlatforms...),
		RequiredHashtags: append([]string(nil), req.RequiredHashtags...),
		BudgetTotal:      req.BudgetTotal,
		RatePer1KViews:   req.RatePer1KViews,
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
	return h.UpdateCampaign.Execute(ctx, commands.UpdateCampaignCommand{
		CampaignID:       campaignID,
		ActorID:          userID,
		Title:            req.Title,
		Description:      req.Description,
		Instructions:     req.Instructions,
		Niche:            req.Niche,
		AllowedPlatforms: append([]string(nil), req.AllowedPlatforms...),
		RequiredHashtags: append([]string(nil), req.RequiredHashtags...),
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
	return httptransport.CampaignDTO{
		CampaignID:       item.CampaignID,
		BrandID:          item.BrandID,
		Title:            item.Title,
		Description:      item.Description,
		Instructions:     item.Instructions,
		Niche:            item.Niche,
		AllowedPlatforms: append([]string(nil), item.AllowedPlatforms...),
		RequiredHashtags: append([]string(nil), item.RequiredHashtags...),
		BudgetTotal:      item.BudgetTotal,
		BudgetSpent:      item.BudgetSpent,
		BudgetRemaining:  item.BudgetRemaining,
		RatePer1KViews:   item.RatePer1KViews,
		Status:           string(item.Status),
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.Format(time.RFC3339),
	}
}
