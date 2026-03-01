package httpadapter

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"solomon/contexts/community-experience/community-health-service/application"
	domainerrors "solomon/contexts/community-experience/community-health-service/domain/errors"
	"solomon/contexts/community-experience/community-health-service/ports"
	httptransport "solomon/contexts/community-experience/community-health-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) IngestWebhookHandler(
	ctx context.Context,
	idempotencyKey string,
	req httptransport.WebhookIngestRequest,
) (httptransport.WebhookIngestResponse, error) {
	input := ports.WebhookIngestInput{
		EventID:         strings.TrimSpace(req.EventID),
		EventType:       strings.TrimSpace(req.EventType),
		MessageID:       strings.TrimSpace(req.MessageID),
		ServerID:        strings.TrimSpace(req.ServerID),
		ChannelID:       strings.TrimSpace(req.ChannelID),
		UserID:          strings.TrimSpace(req.UserID),
		Content:         req.Content,
		ThreadID:        strings.TrimSpace(req.ThreadID),
		OldContent:      req.OldContent,
		NewContent:      req.NewContent,
		DeletedByUserID: strings.TrimSpace(req.DeletedByUserID),
	}
	if ts, ok, err := parseOptionalTime(req.CreatedAt); err != nil {
		return httptransport.WebhookIngestResponse{}, domainerrors.ErrInvalidRequest
	} else if ok {
		input.CreatedAt = &ts
	}
	if ts, ok, err := parseOptionalTime(req.EditedAt); err != nil {
		return httptransport.WebhookIngestResponse{}, domainerrors.ErrInvalidRequest
	} else if ok {
		input.EditedAt = &ts
	}
	if ts, ok, err := parseOptionalTime(req.DeletedAt); err != nil {
		return httptransport.WebhookIngestResponse{}, domainerrors.ErrInvalidRequest
	} else if ok {
		input.DeletedAt = &ts
	}

	item, err := h.Service.IngestWebhook(ctx, idempotencyKey, input)
	if err != nil {
		return httptransport.WebhookIngestResponse{}, err
	}
	resp := httptransport.WebhookIngestResponse{Status: "success"}
	resp.Data.MessageID = item.MessageID
	resp.Data.EventType = item.EventType
	resp.Data.SentimentScore = item.SentimentScore
	resp.Data.ToxicityCategory = item.ToxicityCategory
	resp.Data.MaxSeverity = item.MaxSeverity
	resp.Data.RiskLevel = item.RiskLevel
	resp.Data.AlertsGenerated = item.AlertsGenerated
	resp.Data.ProcessedAt = item.ProcessedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GetCommunityHealthScoreHandler(
	ctx context.Context,
	serverID string,
) (httptransport.CommunityHealthScoreResponse, error) {
	item, err := h.Service.GetCommunityHealthScore(ctx, serverID)
	if err != nil {
		return httptransport.CommunityHealthScoreResponse{}, err
	}
	resp := httptransport.CommunityHealthScoreResponse{Status: "success"}
	resp.Data.ServerID = item.ServerID
	resp.Data.HealthScore = item.HealthScore
	resp.Data.Category = item.Category
	resp.Data.Trend = item.Trend
	resp.Data.WeekStartDate = item.WeekStartDate
	resp.Data.Breakdown.SentimentHealth = item.SentimentHealth
	resp.Data.Breakdown.ToxicityHealth = item.ToxicityHealth
	resp.Data.Breakdown.EngagementHealth = item.EngagementHealth
	resp.Data.Breakdown.LatencyHealth = item.LatencyHealth
	resp.Data.Breakdown.TrendBonus = item.TrendBonus
	resp.Data.Metrics.TotalMessages = item.TotalMessages
	resp.Data.Metrics.PositivePct = math.Round(item.PositivePct*10000) / 100
	resp.Data.Metrics.ToxicityPct = math.Round(item.ToxicityPct*10000) / 100
	resp.Data.Metrics.EngagementGini = item.EngagementGini
	resp.Data.Metrics.AvgModerationLatencyHours = item.AvgModerationLatencyHr
	resp.Data.Alerts = item.Alerts
	resp.Data.CalculatedAt = item.CalculatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GetUserRiskScoreHandler(
	ctx context.Context,
	serverID string,
	userID string,
) (httptransport.UserRiskScoreResponse, error) {
	item, err := h.Service.GetUserRiskScore(ctx, serverID, userID)
	if err != nil {
		return httptransport.UserRiskScoreResponse{}, err
	}
	resp := httptransport.UserRiskScoreResponse{Status: "success"}
	resp.Data.UserID = item.UserID
	resp.Data.ServerID = item.ServerID
	resp.Data.RiskScore = item.RiskScore
	resp.Data.RiskLevel = item.RiskLevel
	resp.Data.ToxicMessageCount = item.ToxicMessageCount
	resp.Data.WarningCount = item.WarningCount
	resp.Data.BanCount = item.BanCount
	if item.LastToxicAt != nil {
		resp.Data.LastToxicMessageAt = item.LastToxicAt.UTC().Format(time.RFC3339)
	}
	resp.Data.Recommendations = append([]string(nil), item.Recommendations...)
	return resp, nil
}

func parseOptionalTime(raw string) (time.Time, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false, err
	}
	return ts.UTC(), true, nil
}
