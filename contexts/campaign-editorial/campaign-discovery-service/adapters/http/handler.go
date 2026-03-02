package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/campaign-discovery-service/application"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/ports"
	httptransport "solomon/contexts/campaign-editorial/campaign-discovery-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) BrowseCampaignsHandler(
	ctx context.Context,
	userID string,
	req httptransport.BrowseCampaignsRequest,
) (httptransport.BrowseCampaignsResponse, error) {
	query := ports.BrowseQuery{
		UserID: strings.TrimSpace(userID),
		Cursor: strings.TrimSpace(req.Cursor),
		SortBy: strings.TrimSpace(req.SortBy),
		Filters: ports.BrowseFilters{
			Category: strings.TrimSpace(req.Category),
			State:    strings.TrimSpace(req.State),
		},
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(req.PageSize)); err == nil {
		query.PageSize = parsed
	}
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(req.BudgetMin), 64); err == nil {
		query.Filters.BudgetMin = parsed
	}
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(req.BudgetMax), 64); err == nil {
		query.Filters.BudgetMax = parsed
	}
	if parsed, ok := parseDate(req.DeadlineAfter); ok {
		query.Filters.DeadlineAfter = &parsed
	}
	if parsed, ok := parseDate(req.DeadlineBefore); ok {
		query.Filters.DeadlineBefore = &parsed
	}
	if raw := strings.TrimSpace(req.Platforms); raw != "" {
		parts := strings.Split(raw, ",")
		query.Filters.Platforms = make([]string, 0, len(parts))
		for _, part := range parts {
			if value := strings.TrimSpace(part); value != "" {
				query.Filters.Platforms = append(query.Filters.Platforms, value)
			}
		}
	}
	if parsed, err := strconv.ParseBool(strings.TrimSpace(req.ExcludeFeatured)); err == nil {
		query.Filters.ExcludeFeatured = parsed
	}

	result, err := h.Service.BrowseCampaigns(ctx, query)
	if err != nil {
		return httptransport.BrowseCampaignsResponse{}, err
	}
	resp := httptransport.BrowseCampaignsResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Campaigns = make([]httptransport.CampaignDTO, 0, len(result.Campaigns))
	for _, item := range result.Campaigns {
		resp.Data.Campaigns = append(resp.Data.Campaigns, mapCampaignDTO(item))
	}
	resp.Data.Pagination.NextCursor = result.Pagination.NextCursor
	resp.Data.Pagination.PrevCursor = result.Pagination.PrevCursor
	resp.Data.Pagination.HasNext = result.Pagination.HasNext
	resp.Data.Pagination.HasPrev = result.Pagination.HasPrev
	resp.Data.Pagination.TotalEstimated = result.Pagination.TotalEstimated
	resp.Data.Pagination.PageSize = result.Pagination.PageSize
	resp.Data.Summary.ResultCount = result.Summary.ResultCount
	resp.Data.Summary.SearchTimeMS = result.Summary.SearchTimeMS
	resp.Data.Summary.CacheHit = result.Summary.CacheHit
	return resp, nil
}

func (h Handler) SearchCampaignsHandler(
	ctx context.Context,
	userID string,
	req httptransport.SearchCampaignsRequest,
) (httptransport.SearchCampaignsResponse, error) {
	query := ports.SearchQuery{
		UserID:   strings.TrimSpace(userID),
		Query:    strings.TrimSpace(req.Query),
		Category: strings.TrimSpace(req.Category),
	}
	if parsed, err := strconv.ParseFloat(strings.TrimSpace(req.BudgetMin), 64); err == nil {
		query.BudgetMin = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(req.Limit)); err == nil {
		query.Limit = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(req.Offset)); err == nil {
		query.Offset = parsed
	}

	result, err := h.Service.SearchCampaigns(ctx, query)
	if err != nil {
		return httptransport.SearchCampaignsResponse{}, err
	}
	resp := httptransport.SearchCampaignsResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Results = make([]httptransport.SearchResultDTO, 0, len(result.Items))
	for _, item := range result.Items {
		resp.Data.Results = append(resp.Data.Results, httptransport.SearchResultDTO{
			CampaignID:      item.CampaignID,
			Title:           item.Title,
			Description:     item.Description,
			CreatorName:     item.CreatorName,
			MatchScore:      item.MatchScore,
			Budget:          item.Budget,
			Deadline:        item.Deadline,
			Category:        item.Category,
			SubmissionCount: item.SubmissionCount,
			IsFeatured:      item.IsFeatured,
		})
	}
	resp.Data.Pagination.Total = result.Total
	resp.Data.Pagination.Limit = result.Limit
	resp.Data.Pagination.Offset = result.Offset
	resp.Data.Pagination.HasNext = result.HasNext
	resp.Data.SearchStats.ExecutionTimeMS = result.ExecutionTime
	resp.Data.SearchStats.IndexVersion = result.IndexVersion
	return resp, nil
}

func (h Handler) GetCampaignDetailsHandler(
	ctx context.Context,
	userID string,
	campaignID string,
) (httptransport.CampaignDetailsResponse, error) {
	result, err := h.Service.GetCampaignDetails(ctx, userID, campaignID)
	if err != nil {
		return httptransport.CampaignDetailsResponse{}, err
	}
	resp := httptransport.CampaignDetailsResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Campaign = mapCampaignDTO(result.Campaign)
	return resp, nil
}

func (h Handler) SaveBookmarkHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	campaignID string,
	req httptransport.SaveBookmarkRequest,
) (httptransport.SaveBookmarkResponse, error) {
	record, err := h.Service.SaveBookmark(ctx, idempotencyKey, ports.BookmarkCommand{
		UserID:     strings.TrimSpace(userID),
		CampaignID: strings.TrimSpace(campaignID),
		Tag:        strings.TrimSpace(req.Tag),
		Note:       strings.TrimSpace(req.Note),
	})
	if err != nil {
		return httptransport.SaveBookmarkResponse{}, err
	}
	resp := httptransport.SaveBookmarkResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.BookmarkID = record.BookmarkID
	resp.Data.UserID = record.UserID
	resp.Data.CampaignID = record.CampaignID
	resp.Data.Tag = record.Tag
	resp.Data.Note = record.Note
	resp.Data.CreatedAt = record.CreatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func mapCampaignDTO(item ports.CampaignSummary) httptransport.CampaignDTO {
	return httptransport.CampaignDTO{
		CampaignID:       item.CampaignID,
		Title:            item.Title,
		Description:      item.Description,
		CreatorName:      item.CreatorName,
		CreatorTier:      item.CreatorTier,
		BudgetTotal:      item.BudgetTotal,
		BudgetSpent:      item.BudgetSpent,
		BudgetRemaining:  item.BudgetTotal - item.BudgetSpent,
		BudgetCurrency:   item.BudgetCurrency,
		RatePer1KViews:   item.RatePer1KViews,
		EstimatedViews:   item.EstimatedViews,
		EstimatedEarning: item.EstimatedEarning,
		SubmissionCount:  item.SubmissionCount,
		ApprovalRate:     item.ApprovalRate,
		Deadline:         item.Deadline,
		Category:         item.Category,
		Platforms:        append([]string(nil), item.Platforms...),
		State:            item.State,
		IsFeatured:       item.IsFeatured,
		FeaturedUntil:    item.FeaturedUntil,
		MatchScore:       item.MatchScore,
		IsEligible:       item.IsEligible,
		Eligibility:      item.Eligibility,
		UserSaved:        item.UserSaved,
		TrendingStatus:   item.TrendingStatus,
		CreatedAt:        item.CreatedAt,
	}
}

func parseDate(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}
