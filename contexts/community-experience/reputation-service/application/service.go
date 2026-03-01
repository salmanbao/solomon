package application

import (
	"context"
	"log/slog"
	"strings"

	domainerrors "solomon/contexts/community-experience/reputation-service/domain/errors"
	"solomon/contexts/community-experience/reputation-service/ports"
)

type Service struct {
	Repo   ports.Repository
	Logger *slog.Logger
}

func (s Service) GetUserReputation(ctx context.Context, userID string) (ports.UserReputation, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ports.UserReputation{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetUserReputation(ctx, userID)
}

func (s Service) GetLeaderboard(ctx context.Context, filter ports.LeaderboardFilter) (ports.Leaderboard, error) {
	if filter.Tier != "" && !ports.IsValidTier(filter.Tier) {
		return ports.Leaderboard{}, domainerrors.ErrInvalidRequest
	}
	if filter.Offset < 0 || filter.Limit < 0 {
		return ports.Leaderboard{}, domainerrors.ErrInvalidRequest
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	filter.ViewerUserID = strings.TrimSpace(filter.ViewerUserID)

	board, err := s.Repo.GetLeaderboard(ctx, filter)
	if err != nil {
		return ports.Leaderboard{}, err
	}

	resolveLogger(s.Logger).Debug("reputation leaderboard served",
		"event", "reputation_leaderboard_served",
		"module", "community-experience/reputation-service",
		"layer", "application",
		"tier", string(filter.Tier),
		"limit", filter.Limit,
		"offset", filter.Offset,
		"total_creators", board.TotalCreators,
	)

	return board, nil
}
