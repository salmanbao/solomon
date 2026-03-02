package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/influencer-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/ports"
)

type Service struct {
	Repo                 ports.Repository
	Idempotency          ports.IdempotencyStore
	RewardProvider       ports.RewardProvider
	GamificationProvider ports.GamificationProvider
	Clock                ports.Clock
	IdempotencyTTL       time.Duration
	Logger               *slog.Logger
}

func (s Service) GetSummary(ctx context.Context, userID string) (ports.DashboardSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ports.DashboardSummary{}, domainerrors.ErrInvalidRequest
	}
	summary, err := s.Repo.GetSummary(ctx, userID)
	if err != nil {
		return ports.DashboardSummary{}, err
	}
	if summary.DependencyStatus == nil {
		summary.DependencyStatus = map[string]string{}
	}

	// Explicit readiness checks required for M34 dependencies on M41 and M47.
	rewardReady := "ready"
	if s.RewardProvider != nil {
		snapshot, err := s.RewardProvider.GetRewardSnapshot(ctx, userID)
		if err != nil {
			rewardReady = "degraded"
		} else {
			summary.RewardAvailable = snapshot.Available
			summary.RewardPending = snapshot.Pending
			summary.RewardCurrency = snapshot.Currency
		}
	} else {
		rewardReady = "degraded"
	}
	gamificationReady := "ready"
	if s.GamificationProvider != nil {
		snapshot, err := s.GamificationProvider.GetGamificationSnapshot(ctx, userID)
		if err != nil {
			gamificationReady = "degraded"
		} else {
			summary.GamificationLevel = snapshot.Level
			summary.GamificationPoints = snapshot.Points
			summary.GamificationBadges = append([]string(nil), snapshot.Badges...)
		}
	} else {
		gamificationReady = "degraded"
	}

	summary.DependencyStatus["m41_reward_engine"] = rewardReady
	summary.DependencyStatus["m47_gamification_service"] = gamificationReady
	return summary, nil
}

func (s Service) ListContent(ctx context.Context, userID string, query ports.ContentQuery) (ports.ContentPage, error) {
	userID = strings.TrimSpace(userID)
	query.View = strings.TrimSpace(strings.ToLower(query.View))
	query.SortBy = strings.TrimSpace(strings.ToLower(query.SortBy))
	query.Status = strings.TrimSpace(strings.ToLower(query.Status))
	if userID == "" {
		return ports.ContentPage{}, domainerrors.ErrInvalidRequest
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		return ports.ContentPage{}, domainerrors.ErrInvalidRequest
	}
	if query.View != "" && query.View != "grid" && query.View != "list" {
		return ports.ContentPage{}, domainerrors.ErrInvalidRequest
	}
	if query.SortBy != "" {
		switch query.SortBy {
		case "views", "earnings", "date_claimed", "status":
		default:
			return ports.ContentPage{}, domainerrors.ErrInvalidRequest
		}
	}
	return s.Repo.ListContent(ctx, userID, query)
}

func (s Service) CreateGoal(ctx context.Context, idempotencyKey string, input ports.GoalCreateInput) (ports.Goal, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.UserID = strings.TrimSpace(input.UserID)
	input.GoalType = strings.TrimSpace(strings.ToLower(input.GoalType))
	input.GoalName = strings.TrimSpace(input.GoalName)
	input.StartDate = strings.TrimSpace(input.StartDate)
	input.EndDate = strings.TrimSpace(input.EndDate)
	if idempotencyKey == "" {
		return ports.Goal{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if input.UserID == "" || input.GoalType == "" || input.GoalName == "" || input.TargetValue <= 0 {
		return ports.Goal{}, domainerrors.ErrInvalidRequest
	}
	requestHash := hashStrings(input.UserID, input.GoalType, input.GoalName, input.StartDate, input.EndDate, formatFloat(input.TargetValue))
	var output ports.Goal
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			item, err := s.Repo.CreateGoal(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return output, err
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}
	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}
	resolveLogger(s.Logger).Debug("influencer dashboard idempotent mutation committed",
		"event", "influencer_dashboard_idempotent_mutation_committed",
		"module", "campaign-editorial/influencer-dashboard-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
