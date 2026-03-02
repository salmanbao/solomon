package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/influencer-dashboard-service/application"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/ports"
	httptransport "solomon/contexts/campaign-editorial/influencer-dashboard-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) SummaryHandler(ctx context.Context, userID string) (httptransport.SummaryResponse, error) {
	summary, err := h.Service.GetSummary(ctx, userID)
	if err != nil {
		return httptransport.SummaryResponse{}, err
	}
	resp := httptransport.SummaryResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.QuickStats.TotalViews = summary.QuickStats.TotalViews
	resp.Data.QuickStats.TotalEarnings = summary.QuickStats.TotalEarnings
	resp.Data.QuickStats.AverageCPV = summary.QuickStats.AverageCPV
	resp.Data.QuickStats.SuccessRate = summary.QuickStats.SuccessRate
	resp.Data.TopClips = make([]struct {
		ID             string  `json:"id"`
		Title          string  `json:"title"`
		ThumbnailURL   string  `json:"thumbnail_url"`
		Views          int     `json:"views"`
		Earnings       float64 `json:"earnings"`
		EngagementRate float64 `json:"engagement_rate"`
		PublishedAt    string  `json:"published_at"`
	}, 0, len(summary.TopClips))
	for _, item := range summary.TopClips {
		resp.Data.TopClips = append(resp.Data.TopClips, struct {
			ID             string  `json:"id"`
			Title          string  `json:"title"`
			ThumbnailURL   string  `json:"thumbnail_url"`
			Views          int     `json:"views"`
			Earnings       float64 `json:"earnings"`
			EngagementRate float64 `json:"engagement_rate"`
			PublishedAt    string  `json:"published_at"`
		}{
			ID:             item.ID,
			Title:          item.Title,
			ThumbnailURL:   item.ThumbnailURL,
			Views:          item.Views,
			Earnings:       item.Earnings,
			EngagementRate: item.EngagementRate,
			PublishedAt:    item.PublishedAt.UTC().Format(time.RFC3339),
		})
	}
	resp.Data.UpcomingPayouts = make([]struct {
		ID     string  `json:"id"`
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
		Status string  `json:"status"`
		Method string  `json:"method"`
	}, 0, len(summary.UpcomingPayouts))
	for _, item := range summary.UpcomingPayouts {
		resp.Data.UpcomingPayouts = append(resp.Data.UpcomingPayouts, struct {
			ID     string  `json:"id"`
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
			Status string  `json:"status"`
			Method string  `json:"method"`
		}{
			ID:     item.ID,
			Date:   item.Date.UTC().Format("2006-01-02"),
			Amount: item.Amount,
			Status: item.Status,
			Method: item.Method,
		})
	}
	resp.Data.Reward.Available = summary.RewardAvailable
	resp.Data.Reward.Pending = summary.RewardPending
	resp.Data.Reward.Currency = summary.RewardCurrency
	resp.Data.Gamification.Level = summary.GamificationLevel
	resp.Data.Gamification.Points = summary.GamificationPoints
	resp.Data.Gamification.Badges = append([]string(nil), summary.GamificationBadges...)
	resp.Data.DependencyStatus = summary.DependencyStatus
	return resp, nil
}

func (h Handler) ContentHandler(ctx context.Context, userID string, limitRaw string, offsetRaw string, view string, sortBy string, status string, dateFrom string, dateTo string) (httptransport.ContentResponse, error) {
	query := ports.ContentQuery{
		View:     strings.TrimSpace(view),
		SortBy:   strings.TrimSpace(sortBy),
		Status:   strings.TrimSpace(status),
		DateFrom: strings.TrimSpace(dateFrom),
		DateTo:   strings.TrimSpace(dateTo),
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(limitRaw)); err == nil {
		query.Limit = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(offsetRaw)); err == nil {
		query.Offset = parsed
	}
	page, err := h.Service.ListContent(ctx, userID, query)
	if err != nil {
		return httptransport.ContentResponse{}, err
	}
	resp := httptransport.ContentResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.TotalCount = page.TotalCount
	resp.Data.Items = make([]struct {
		ID             string  `json:"id"`
		Title          string  `json:"title"`
		ThumbnailURL   string  `json:"thumbnail_url"`
		Status         string  `json:"status"`
		Views          int     `json:"views"`
		Earnings       float64 `json:"earnings"`
		EngagementRate float64 `json:"engagement_rate"`
		ClaimedAt      string  `json:"claimed_at"`
		PublishedAt    string  `json:"published_at,omitempty"`
	}, 0, len(page.Items))
	for _, item := range page.Items {
		dto := struct {
			ID             string  `json:"id"`
			Title          string  `json:"title"`
			ThumbnailURL   string  `json:"thumbnail_url"`
			Status         string  `json:"status"`
			Views          int     `json:"views"`
			Earnings       float64 `json:"earnings"`
			EngagementRate float64 `json:"engagement_rate"`
			ClaimedAt      string  `json:"claimed_at"`
			PublishedAt    string  `json:"published_at,omitempty"`
		}{
			ID:             item.ID,
			Title:          item.Title,
			ThumbnailURL:   item.ThumbnailURL,
			Status:         item.Status,
			Views:          item.Views,
			Earnings:       item.Earnings,
			EngagementRate: item.EngagementRate,
			ClaimedAt:      item.ClaimedAt.UTC().Format(time.RFC3339),
		}
		if item.PublishedAt != nil {
			dto.PublishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
		}
		resp.Data.Items = append(resp.Data.Items, dto)
	}
	return resp, nil
}

func (h Handler) CreateGoalHandler(ctx context.Context, idempotencyKey string, userID string, req httptransport.CreateGoalRequest) (httptransport.CreateGoalResponse, error) {
	item, err := h.Service.CreateGoal(ctx, idempotencyKey, ports.GoalCreateInput{
		UserID:      strings.TrimSpace(userID),
		GoalType:    strings.TrimSpace(req.GoalType),
		GoalName:    strings.TrimSpace(req.GoalName),
		TargetValue: req.TargetValue,
		StartDate:   strings.TrimSpace(req.StartDate),
		EndDate:     strings.TrimSpace(req.EndDate),
	})
	if err != nil {
		return httptransport.CreateGoalResponse{}, err
	}
	resp := httptransport.CreateGoalResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.ID = item.ID
	resp.Data.GoalType = item.GoalType
	resp.Data.GoalName = item.GoalName
	resp.Data.TargetValue = item.TargetValue
	resp.Data.CurrentValue = item.CurrentValue
	resp.Data.ProgressPercent = item.ProgressPercent
	resp.Data.Status = item.Status
	resp.Data.StartDate = item.StartDate
	resp.Data.EndDate = item.EndDate
	resp.Data.CreatedAt = item.CreatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}
