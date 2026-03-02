package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/gamification-service/domain/errors"
	"solomon/contexts/community-experience/gamification-service/ports"
)

type Service struct {
	Repo                  ports.Repository
	Idempotency           ports.IdempotencyStore
	Clock                 ports.Clock
	IDGen                 ports.IDGenerator
	IdempotencyTTL        time.Duration
	DisableTierMultiplier bool
	Logger                *slog.Logger
}

type AwardPointsResult struct {
	Points   ports.UserPoints
	Log      ports.PointsLog
	Replayed bool
}

type GrantBadgeResult struct {
	Badge    ports.BadgeGrant
	Replayed bool
}

func (s Service) AwardPoints(
	ctx context.Context,
	idempotencyKey string,
	input ports.AwardPointsInput,
) (AwardPointsResult, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return AwardPointsResult{}, domainerrors.ErrIdempotencyKeyMissing
	}
	if !isValidAwardInput(input) {
		return AwardPointsResult{}, domainerrors.ErrInvalidInput
	}
	projection, err := s.Repo.GetUserProjection(ctx, strings.TrimSpace(input.UserID))
	if err != nil {
		return AwardPointsResult{}, err
	}
	if !projection.AuthActive || !projection.ProfileExists {
		return AwardPointsResult{}, domainerrors.ErrDependencyUnavailable
	}

	now := s.now()
	requestHash := hashPayload(map[string]any{
		"user_id":      strings.TrimSpace(input.UserID),
		"action_type":  strings.TrimSpace(input.ActionType),
		"points":       input.Points,
		"reason":       strings.TrimSpace(input.Reason),
		"tier":         strings.ToLower(strings.TrimSpace(projection.ReputationTier)),
		"request_type": "award_points",
	})

	record, found, err := s.Idempotency.GetRecord(ctx, strings.TrimSpace(idempotencyKey), now)
	if err != nil {
		return AwardPointsResult{}, err
	}
	if found {
		if record.RequestHash != requestHash {
			return AwardPointsResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replayed AwardPointsResult
		if err := json.Unmarshal(record.ResponsePayload, &replayed); err != nil {
			return AwardPointsResult{}, err
		}
		replayed.Replayed = true
		return replayed, nil
	}

	multiplier := multiplierForTier(projection.ReputationTier)
	if s.DisableTierMultiplier {
		multiplier = 1.0
	}
	finalPoints := int(math.Round(float64(input.Points) * multiplier))
	if finalPoints < 1 {
		finalPoints = 1
	}

	logID, err := s.IDGen.NewID(ctx)
	if err != nil {
		return AwardPointsResult{}, err
	}
	log := ports.PointsLog{
		LogID:       strings.TrimSpace(logID),
		UserID:      strings.TrimSpace(input.UserID),
		ActionType:  strings.TrimSpace(input.ActionType),
		BasePoints:  input.Points,
		Multiplier:  multiplier,
		FinalPoints: finalPoints,
		Reason:      strings.TrimSpace(input.Reason),
		CreatedAt:   now,
	}
	if err := s.Repo.AppendPointsLog(ctx, log); err != nil {
		return AwardPointsResult{}, err
	}

	points, err := s.Repo.IncrementUserPoints(ctx, log.UserID, finalPoints, now)
	if err != nil {
		return AwardPointsResult{}, err
	}
	points.CurrentLevel = levelForPoints(points.TotalPoints)

	result := AwardPointsResult{
		Points: points,
		Log:    log,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return AwardPointsResult{}, err
	}
	if err := s.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             strings.TrimSpace(idempotencyKey),
		RequestHash:     requestHash,
		ResponsePayload: payload,
		ExpiresAt:       now.Add(s.idempotencyTTL()),
	}); err != nil {
		return AwardPointsResult{}, err
	}

	resolveLogger(s.Logger).Info("gamification points awarded",
		"event", "gamification_points_awarded",
		"module", "community-experience/gamification-service",
		"layer", "application",
		"user_id", points.UserID,
		"action_type", log.ActionType,
		"final_points", log.FinalPoints,
		"total_points", points.TotalPoints,
	)
	return result, nil
}

func (s Service) GrantBadge(
	ctx context.Context,
	idempotencyKey string,
	input ports.GrantBadgeInput,
) (GrantBadgeResult, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return GrantBadgeResult{}, domainerrors.ErrIdempotencyKeyMissing
	}
	if strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.BadgeKey) == "" {
		return GrantBadgeResult{}, domainerrors.ErrInvalidInput
	}
	projection, err := s.Repo.GetUserProjection(ctx, strings.TrimSpace(input.UserID))
	if err != nil {
		return GrantBadgeResult{}, err
	}
	if !projection.AuthActive || !projection.ProfileExists {
		return GrantBadgeResult{}, domainerrors.ErrDependencyUnavailable
	}

	now := s.now()
	requestHash := hashPayload(map[string]any{
		"user_id":      strings.TrimSpace(input.UserID),
		"badge_key":    strings.TrimSpace(input.BadgeKey),
		"reason":       strings.TrimSpace(input.Reason),
		"request_type": "grant_badge",
	})
	record, found, err := s.Idempotency.GetRecord(ctx, strings.TrimSpace(idempotencyKey), now)
	if err != nil {
		return GrantBadgeResult{}, err
	}
	if found {
		if record.RequestHash != requestHash {
			return GrantBadgeResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replayed GrantBadgeResult
		if err := json.Unmarshal(record.ResponsePayload, &replayed); err != nil {
			return GrantBadgeResult{}, err
		}
		replayed.Replayed = true
		return replayed, nil
	}

	badgeID, err := s.IDGen.NewID(ctx)
	if err != nil {
		return GrantBadgeResult{}, err
	}
	grant, _, err := s.Repo.UpsertBadge(ctx, ports.BadgeGrant{
		BadgeID:    strings.TrimSpace(badgeID),
		UserID:     strings.TrimSpace(input.UserID),
		BadgeKey:   strings.TrimSpace(input.BadgeKey),
		Reason:     strings.TrimSpace(input.Reason),
		GrantedAt:  now,
		SourceType: "manual",
	})
	if err != nil {
		return GrantBadgeResult{}, err
	}
	result := GrantBadgeResult{Badge: grant}

	payload, err := json.Marshal(result)
	if err != nil {
		return GrantBadgeResult{}, err
	}
	if err := s.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             strings.TrimSpace(idempotencyKey),
		RequestHash:     requestHash,
		ResponsePayload: payload,
		ExpiresAt:       now.Add(s.idempotencyTTL()),
	}); err != nil {
		return GrantBadgeResult{}, err
	}
	return result, nil
}

func (s Service) GetUserSummary(ctx context.Context, userID string) (ports.UserSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ports.UserSummary{}, domainerrors.ErrInvalidInput
	}
	projection, err := s.Repo.GetUserProjection(ctx, userID)
	if err != nil {
		return ports.UserSummary{}, err
	}
	if !projection.AuthActive || !projection.ProfileExists {
		return ports.UserSummary{}, domainerrors.ErrDependencyUnavailable
	}

	points, err := s.Repo.GetUserPoints(ctx, userID)
	if err != nil {
		return ports.UserSummary{}, err
	}
	badges, err := s.Repo.ListUserBadges(ctx, userID)
	if err != nil {
		return ports.UserSummary{}, err
	}
	sort.Slice(badges, func(i, j int) bool {
		return badges[i].GrantedAt.After(badges[j].GrantedAt)
	})

	return ports.UserSummary{
		UserID:         userID,
		TotalPoints:    points.TotalPoints,
		CurrentLevel:   levelForPoints(points.TotalPoints),
		ReputationTier: projection.ReputationTier,
		Badges:         badges,
	}, nil
}

func (s Service) GetLeaderboard(ctx context.Context, limit int, offset int) ([]ports.LeaderboardEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.Repo.ListLeaderboard(ctx, limit, offset)
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func isValidAwardInput(input ports.AwardPointsInput) bool {
	return strings.TrimSpace(input.UserID) != "" &&
		strings.TrimSpace(input.ActionType) != "" &&
		input.Points > 0
}

func multiplierForTier(tier string) float64 {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "silver":
		return 1.1
	case "gold":
		return 1.2
	case "platinum":
		return 1.25
	default:
		return 1.0
	}
}

func levelForPoints(totalPoints int) int {
	if totalPoints <= 0 {
		return 1
	}
	level := 1
	for level < 100 {
		next := 100*level + (50*level*(level-1))/2
		if totalPoints < next {
			break
		}
		level++
	}
	return level
}

func hashPayload(payload map[string]any) string {
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
