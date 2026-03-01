package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/community-experience/reputation-service/application"
	domainerrors "solomon/contexts/community-experience/reputation-service/domain/errors"
	"solomon/contexts/community-experience/reputation-service/ports"
	httptransport "solomon/contexts/community-experience/reputation-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) GetUserReputationHandler(
	ctx context.Context,
	userID string,
) (httptransport.UserReputationResponse, error) {
	item, err := h.Service.GetUserReputation(ctx, userID)
	if err != nil {
		return httptransport.UserReputationResponse{}, err
	}

	resp := httptransport.UserReputationResponse{Status: "success"}
	resp.Data.UserID = item.UserID
	resp.Data.ReputationScore = item.ReputationScore
	resp.Data.Tier = string(item.Tier)
	resp.Data.TierProgress.CurrentPoints = item.TierProgress.CurrentPoints
	resp.Data.TierProgress.NextTierPoints = item.TierProgress.NextTierPoints
	resp.Data.TierProgress.PointsToNextTier = item.TierProgress.PointsToNextTier
	resp.Data.PreviousScore = item.PreviousScore
	resp.Data.ScoreTrend.WeekOverWeek = item.ScoreTrend.WeekOverWeek
	resp.Data.ScoreTrend.MonthOverMonth = item.ScoreTrend.MonthOverMonth
	resp.Data.ScoreTrend.Direction = item.ScoreTrend.Direction
	resp.Data.ScoreBreakdown.ApprovalRate = toScoreComponentDTO(item.ScoreBreakdown.ApprovalRate)
	resp.Data.ScoreBreakdown.ViewVelocity = toScoreComponentDTO(item.ScoreBreakdown.ViewVelocity)
	resp.Data.ScoreBreakdown.EarningsConsistency = toScoreComponentDTO(item.ScoreBreakdown.EarningsConsistency)
	resp.Data.ScoreBreakdown.SupportSatisfaction = toScoreComponentDTO(item.ScoreBreakdown.SupportSatisfaction)
	resp.Data.ScoreBreakdown.ModerationRecord = toScoreComponentDTO(item.ScoreBreakdown.ModerationRecord)
	resp.Data.ScoreBreakdown.CommunitySentiment = toScoreComponentDTO(item.ScoreBreakdown.CommunitySentiment)
	resp.Data.Badges = make([]struct {
		BadgeID   string `json:"badge_id"`
		BadgeName string `json:"badge_name"`
		EarnedAt  string `json:"earned_at"`
		Category  string `json:"category"`
		Rarity    string `json:"rarity"`
		IconURL   string `json:"icon_url"`
	}, 0, len(item.Badges))
	for _, badge := range item.Badges {
		resp.Data.Badges = append(resp.Data.Badges, struct {
			BadgeID   string `json:"badge_id"`
			BadgeName string `json:"badge_name"`
			EarnedAt  string `json:"earned_at"`
			Category  string `json:"category"`
			Rarity    string `json:"rarity"`
			IconURL   string `json:"icon_url"`
		}{
			BadgeID:   badge.BadgeID,
			BadgeName: badge.BadgeName,
			EarnedAt:  badge.EarnedAt,
			Category:  badge.Category,
			Rarity:    badge.Rarity,
			IconURL:   badge.IconURL,
		})
	}
	resp.Data.CalculatedAt = item.CalculatedAt.UTC().Format(time.RFC3339)
	resp.Data.NextRecalculationAt = item.NextRecalculationAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GetLeaderboardHandler(
	ctx context.Context,
	req httptransport.LeaderboardRequest,
	viewerUserID string,
) (httptransport.LeaderboardResponse, error) {
	filter := ports.LeaderboardFilter{
		ViewerUserID: strings.TrimSpace(viewerUserID),
	}
	if strings.TrimSpace(req.Tier) != "" {
		tier, ok := ports.ParseTier(req.Tier)
		if !ok {
			return httptransport.LeaderboardResponse{}, domainerrors.ErrInvalidRequest
		}
		filter.Tier = tier
	}
	if strings.TrimSpace(req.Limit) != "" {
		limit, err := strconv.Atoi(strings.TrimSpace(req.Limit))
		if err != nil {
			return httptransport.LeaderboardResponse{}, domainerrors.ErrInvalidRequest
		}
		filter.Limit = limit
	}
	if strings.TrimSpace(req.Offset) != "" {
		offset, err := strconv.Atoi(strings.TrimSpace(req.Offset))
		if err != nil {
			return httptransport.LeaderboardResponse{}, domainerrors.ErrInvalidRequest
		}
		filter.Offset = offset
	}

	board, err := h.Service.GetLeaderboard(ctx, filter)
	if err != nil {
		return httptransport.LeaderboardResponse{}, err
	}

	resp := httptransport.LeaderboardResponse{Status: "success"}
	resp.Data.Leaderboard = make([]struct {
		Rank     int      `json:"rank"`
		UserID   string   `json:"user_id"`
		Username string   `json:"username"`
		Tier     string   `json:"tier"`
		Score    int      `json:"score"`
		Badges   []string `json:"badges"`
		Trend    string   `json:"trend"`
	}, 0, len(board.Entries))
	for _, item := range board.Entries {
		resp.Data.Leaderboard = append(resp.Data.Leaderboard, struct {
			Rank     int      `json:"rank"`
			UserID   string   `json:"user_id"`
			Username string   `json:"username"`
			Tier     string   `json:"tier"`
			Score    int      `json:"score"`
			Badges   []string `json:"badges"`
			Trend    string   `json:"trend"`
		}{
			Rank:     item.Rank,
			UserID:   item.UserID,
			Username: item.Username,
			Tier:     string(item.Tier),
			Score:    item.Score,
			Badges:   append([]string(nil), item.Badges...),
			Trend:    item.Trend,
		})
	}
	resp.Data.TotalCreators = board.TotalCreators
	resp.Data.YourRank = board.YourRank
	return resp, nil
}

func toScoreComponentDTO(item ports.ScoreComponent) httptransport.ScoreComponentDTO {
	return httptransport.ScoreComponentDTO{
		Value:        item.Value,
		Weight:       item.Weight,
		Contribution: item.Contribution,
	}
}
