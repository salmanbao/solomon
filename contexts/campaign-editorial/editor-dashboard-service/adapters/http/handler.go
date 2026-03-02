package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/editor-dashboard-service/application"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/ports"
	httptransport "solomon/contexts/campaign-editorial/editor-dashboard-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) FeedHandler(ctx context.Context, userID string, limitRaw string, offsetRaw string, sortBy string) (httptransport.FeedResponse, error) {
	query := ports.FeedQuery{SortBy: strings.TrimSpace(sortBy)}
	if parsed, err := strconv.Atoi(strings.TrimSpace(limitRaw)); err == nil {
		query.Limit = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(offsetRaw)); err == nil {
		query.Offset = parsed
	}
	items, err := h.Service.GetFeed(ctx, userID, query)
	if err != nil {
		return httptransport.FeedResponse{}, err
	}
	resp := httptransport.FeedResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.Items = make([]httptransport.FeedItemDTO, 0, len(items))
	for _, item := range items {
		resp.Data.Items = append(resp.Data.Items, httptransport.FeedItemDTO{
			CampaignID:       item.CampaignID,
			Title:            item.Title,
			Category:         item.Category,
			RewardRate:       item.RewardRate,
			BudgetRemaining:  item.BudgetRemaining,
			MatchScore:       item.MatchScore,
			SubmissionStatus: item.SubmissionStatus,
			Saved:            item.Saved,
		})
	}
	return resp, nil
}

func (h Handler) SubmissionsHandler(ctx context.Context, userID string, statusRaw string, limitRaw string, offsetRaw string) (httptransport.SubmissionsResponse, error) {
	query := ports.SubmissionQuery{Status: strings.TrimSpace(statusRaw)}
	if parsed, err := strconv.Atoi(strings.TrimSpace(limitRaw)); err == nil {
		query.Limit = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(offsetRaw)); err == nil {
		query.Offset = parsed
	}
	items, err := h.Service.ListSubmissions(ctx, userID, query)
	if err != nil {
		return httptransport.SubmissionsResponse{}, err
	}
	resp := httptransport.SubmissionsResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.Items = make([]httptransport.SubmissionDTO, 0, len(items))
	for _, item := range items {
		dto := httptransport.SubmissionDTO{
			SubmissionID:   item.SubmissionID,
			CampaignID:     item.CampaignID,
			CampaignTitle:  item.CampaignTitle,
			Status:         item.Status,
			Views:          item.Views,
			Earnings:       item.Earnings,
			SubmittedAt:    item.SubmittedAt.UTC().Format(time.RFC3339),
			Feedback:       item.Feedback,
			RejectionCode:  item.RejectionCode,
			ModerationNote: item.ModerationNote,
		}
		if item.ReviewedAt != nil {
			dto.ReviewedAt = item.ReviewedAt.UTC().Format(time.RFC3339)
		}
		resp.Data.Items = append(resp.Data.Items, dto)
	}
	return resp, nil
}

func (h Handler) EarningsHandler(ctx context.Context, userID string) (httptransport.EarningsResponse, error) {
	summary, err := h.Service.GetEarnings(ctx, userID)
	if err != nil {
		return httptransport.EarningsResponse{}, err
	}
	resp := httptransport.EarningsResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.Available = summary.Available
	resp.Data.Pending = summary.Pending
	resp.Data.Lifetime = summary.Lifetime
	resp.Data.Currency = summary.Currency
	return resp, nil
}

func (h Handler) PerformanceHandler(ctx context.Context, userID string) (httptransport.PerformanceResponse, error) {
	summary, err := h.Service.GetPerformance(ctx, userID)
	if err != nil {
		return httptransport.PerformanceResponse{}, err
	}
	resp := httptransport.PerformanceResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.ApprovalRate = summary.ApprovalRate
	resp.Data.AvgViewsPerClip = summary.AvgViewsPerClip
	resp.Data.ReputationScore = summary.ReputationScore
	resp.Data.BenchmarkPercentile = summary.BenchmarkPercentile
	return resp, nil
}

func (h Handler) SaveCampaignHandler(ctx context.Context, idempotencyKey string, userID string, campaignID string) (httptransport.SaveCampaignResponse, error) {
	result, err := h.Service.SaveCampaign(ctx, idempotencyKey, userID, campaignID)
	if err != nil {
		return httptransport.SaveCampaignResponse{}, err
	}
	resp := httptransport.SaveCampaignResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.CampaignID = result.CampaignID
	resp.Data.Saved = result.Saved
	if result.SavedAt != nil {
		resp.Data.SavedAt = result.SavedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) RemoveSavedCampaignHandler(ctx context.Context, idempotencyKey string, userID string, campaignID string) (httptransport.SaveCampaignResponse, error) {
	result, err := h.Service.RemoveSavedCampaign(ctx, idempotencyKey, userID, campaignID)
	if err != nil {
		return httptransport.SaveCampaignResponse{}, err
	}
	resp := httptransport.SaveCampaignResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.CampaignID = result.CampaignID
	resp.Data.Saved = result.Saved
	if result.SavedAt != nil {
		resp.Data.SavedAt = result.SavedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) ExportSubmissionsHandler(ctx context.Context, userID string, statusRaw string) (httptransport.ExportSubmissionsResponse, error) {
	csvContent, err := h.Service.ExportSubmissionsCSV(ctx, userID, ports.SubmissionQuery{
		Status: strings.TrimSpace(statusRaw),
		Limit:  50,
	})
	if err != nil {
		return httptransport.ExportSubmissionsResponse{}, err
	}
	resp := httptransport.ExportSubmissionsResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.Filename = "editor-submissions.csv"
	resp.Data.Content = csvContent
	return resp, nil
}
