package ports

import (
	"context"
	"strings"
	"time"
)

type Tier string

const (
	TierBronze   Tier = "bronze"
	TierSilver   Tier = "silver"
	TierGold     Tier = "gold"
	TierPlatinum Tier = "platinum"
)

func ParseTier(raw string) (Tier, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(TierBronze):
		return TierBronze, true
	case string(TierSilver):
		return TierSilver, true
	case string(TierGold):
		return TierGold, true
	case string(TierPlatinum):
		return TierPlatinum, true
	default:
		return "", false
	}
}

func IsValidTier(tier Tier) bool {
	switch tier {
	case TierBronze, TierSilver, TierGold, TierPlatinum:
		return true
	default:
		return false
	}
}

type TierProgress struct {
	CurrentPoints    int
	NextTierPoints   int
	PointsToNextTier int
}

type ScoreTrend struct {
	WeekOverWeek   int
	MonthOverMonth int
	Direction      string
}

type ScoreComponent struct {
	Value        any
	Weight       float64
	Contribution float64
}

type ScoreBreakdown struct {
	ApprovalRate        ScoreComponent
	ViewVelocity        ScoreComponent
	EarningsConsistency ScoreComponent
	SupportSatisfaction ScoreComponent
	ModerationRecord    ScoreComponent
	CommunitySentiment  ScoreComponent
}

type Badge struct {
	BadgeID   string
	BadgeName string
	EarnedAt  string
	Category  string
	Rarity    string
	IconURL   string
}

type UserReputation struct {
	UserID              string
	ReputationScore     int
	Tier                Tier
	TierProgress        TierProgress
	PreviousScore       int
	ScoreTrend          ScoreTrend
	ScoreBreakdown      ScoreBreakdown
	Badges              []Badge
	CalculatedAt        time.Time
	NextRecalculationAt time.Time
}

type LeaderboardEntry struct {
	Rank     int
	UserID   string
	Username string
	Tier     Tier
	Score    int
	Badges   []string
	Trend    string
}

type Leaderboard struct {
	Entries       []LeaderboardEntry
	TotalCreators int
	YourRank      int
}

type LeaderboardFilter struct {
	Tier         Tier
	Limit        int
	Offset       int
	ViewerUserID string
}

type Repository interface {
	GetUserReputation(ctx context.Context, userID string) (UserReputation, error)
	GetLeaderboard(ctx context.Context, filter LeaderboardFilter) (Leaderboard, error)
}
