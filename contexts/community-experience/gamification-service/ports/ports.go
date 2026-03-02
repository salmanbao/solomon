package ports

import (
	"context"
	"time"
)

type UserProjection struct {
	UserID         string
	ProfileExists  bool
	AuthActive     bool
	ReputationTier string
}

type UserPoints struct {
	UserID       string
	TotalPoints  int
	CurrentLevel int
	UpdatedAt    time.Time
}

type PointsLog struct {
	LogID       string
	UserID      string
	ActionType  string
	BasePoints  int
	Multiplier  float64
	FinalPoints int
	Reason      string
	CreatedAt   time.Time
}

type BadgeGrant struct {
	BadgeID    string
	UserID     string
	BadgeKey   string
	Reason     string
	GrantedAt  time.Time
	SourceType string
}

type UserSummary struct {
	UserID         string
	TotalPoints    int
	CurrentLevel   int
	ReputationTier string
	Badges         []BadgeGrant
}

type LeaderboardEntry struct {
	UserID       string
	TotalPoints  int
	CurrentLevel int
	Rank         int
}

type AwardPointsInput struct {
	UserID     string
	ActionType string
	Points     int
	Reason     string
}

type GrantBadgeInput struct {
	UserID   string
	BadgeKey string
	Reason   string
}

type Repository interface {
	GetUserProjection(ctx context.Context, userID string) (UserProjection, error)
	AppendPointsLog(ctx context.Context, log PointsLog) error
	IncrementUserPoints(ctx context.Context, userID string, delta int, updatedAt time.Time) (UserPoints, error)
	UpsertBadge(ctx context.Context, grant BadgeGrant) (BadgeGrant, bool, error)
	ListUserBadges(ctx context.Context, userID string) ([]BadgeGrant, error)
	GetUserPoints(ctx context.Context, userID string) (UserPoints, error)
	ListLeaderboard(ctx context.Context, limit int, offset int) ([]LeaderboardEntry, error)
}

type IdempotencyRecord struct {
	Key             string
	RequestHash     string
	ResponsePayload []byte
	ExpiresAt       time.Time
}

type IdempotencyStore interface {
	GetRecord(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	PutRecord(ctx context.Context, record IdempotencyRecord) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
