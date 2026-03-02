package httpadapter

import (
	"context"
	"log/slog"
	"time"

	"solomon/contexts/community-experience/gamification-service/application"
	"solomon/contexts/community-experience/gamification-service/ports"
	httptransport "solomon/contexts/community-experience/gamification-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) AwardPointsHandler(
	ctx context.Context,
	idempotencyKey string,
	req httptransport.AwardPointsRequest,
) (httptransport.AwardPointsResponse, error) {
	result, err := h.Service.AwardPoints(ctx, idempotencyKey, ports.AwardPointsInput{
		UserID:     req.UserID,
		ActionType: req.ActionType,
		Points:     req.Points,
		Reason:     req.Reason,
	})
	if err != nil {
		return httptransport.AwardPointsResponse{}, err
	}
	resp := httptransport.AwardPointsResponse{
		Status:   "success",
		Replayed: result.Replayed,
	}
	resp.Data.UserID = result.Points.UserID
	resp.Data.ActionType = result.Log.ActionType
	resp.Data.BasePoints = result.Log.BasePoints
	resp.Data.Multiplier = result.Log.Multiplier
	resp.Data.FinalPoints = result.Log.FinalPoints
	resp.Data.TotalPoints = result.Points.TotalPoints
	resp.Data.CurrentLevel = result.Points.CurrentLevel
	resp.Data.GrantedAt = result.Log.CreatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GrantBadgeHandler(
	ctx context.Context,
	idempotencyKey string,
	req httptransport.GrantBadgeRequest,
) (httptransport.GrantBadgeResponse, error) {
	result, err := h.Service.GrantBadge(ctx, idempotencyKey, ports.GrantBadgeInput{
		UserID:   req.UserID,
		BadgeKey: req.BadgeKey,
		Reason:   req.Reason,
	})
	if err != nil {
		return httptransport.GrantBadgeResponse{}, err
	}
	resp := httptransport.GrantBadgeResponse{
		Status:   "success",
		Replayed: result.Replayed,
	}
	resp.Data.BadgeID = result.Badge.BadgeID
	resp.Data.UserID = result.Badge.UserID
	resp.Data.BadgeKey = result.Badge.BadgeKey
	resp.Data.Reason = result.Badge.Reason
	resp.Data.GrantedAt = result.Badge.GrantedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GetUserSummaryHandler(ctx context.Context, userID string) (httptransport.UserSummaryResponse, error) {
	summary, err := h.Service.GetUserSummary(ctx, userID)
	if err != nil {
		return httptransport.UserSummaryResponse{}, err
	}
	resp := httptransport.UserSummaryResponse{Status: "success"}
	resp.Data.UserID = summary.UserID
	resp.Data.TotalPoints = summary.TotalPoints
	resp.Data.CurrentLevel = summary.CurrentLevel
	resp.Data.ReputationTier = summary.ReputationTier
	resp.Data.Badges = make([]string, 0, len(summary.Badges))
	for _, badge := range summary.Badges {
		resp.Data.Badges = append(resp.Data.Badges, badge.BadgeKey)
	}
	return resp, nil
}

func (h Handler) GetLeaderboardHandler(ctx context.Context, limit int, offset int) (httptransport.LeaderboardResponse, error) {
	items, err := h.Service.GetLeaderboard(ctx, limit, offset)
	if err != nil {
		return httptransport.LeaderboardResponse{}, err
	}
	resp := httptransport.LeaderboardResponse{
		Status: "success",
		Data:   make([]httptransport.LeaderboardEntryDTO, 0, len(items)),
	}
	for _, item := range items {
		resp.Data = append(resp.Data, httptransport.LeaderboardEntryDTO{
			Rank:         item.Rank,
			UserID:       item.UserID,
			TotalPoints:  item.TotalPoints,
			CurrentLevel: item.CurrentLevel,
		})
	}
	return resp, nil
}
