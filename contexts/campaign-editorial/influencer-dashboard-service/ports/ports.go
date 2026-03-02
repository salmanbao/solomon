package ports

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type RewardSnapshot struct {
	Available float64
	Pending   float64
	Currency  string
}

type RewardProvider interface {
	GetRewardSnapshot(ctx context.Context, userID string) (RewardSnapshot, error)
}

type GamificationSnapshot struct {
	Level  int
	Points int
	Badges []string
}

type GamificationProvider interface {
	GetGamificationSnapshot(ctx context.Context, userID string) (GamificationSnapshot, error)
}

type QuickStats struct {
	TotalViews    int
	TotalEarnings float64
	AverageCPV    float64
	SuccessRate   float64
}

type TopClip struct {
	ID             string
	Title          string
	ThumbnailURL   string
	Views          int
	Earnings       float64
	EngagementRate float64
	PublishedAt    time.Time
}

type UpcomingPayout struct {
	ID     string
	Date   time.Time
	Amount float64
	Status string
	Method string
}

type DashboardSummary struct {
	QuickStats         QuickStats
	TopClips           []TopClip
	UpcomingPayouts    []UpcomingPayout
	RewardAvailable    float64
	RewardPending      float64
	RewardCurrency     string
	GamificationLevel  int
	GamificationPoints int
	GamificationBadges []string
	DependencyStatus   map[string]string
}

type ContentQuery struct {
	Limit    int
	Offset   int
	View     string
	SortBy   string
	Status   string
	DateFrom string
	DateTo   string
}

type ContentItem struct {
	ID             string
	Title          string
	ThumbnailURL   string
	Status         string
	Views          int
	Earnings       float64
	EngagementRate float64
	ClaimedAt      time.Time
	PublishedAt    *time.Time
}

type ContentPage struct {
	TotalCount int
	Items      []ContentItem
}

type GoalCreateInput struct {
	UserID      string
	GoalType    string
	GoalName    string
	TargetValue float64
	StartDate   string
	EndDate     string
}

type Goal struct {
	ID              string
	UserID          string
	GoalType        string
	GoalName        string
	TargetValue     float64
	CurrentValue    float64
	ProgressPercent float64
	Status          string
	StartDate       string
	EndDate         string
	CreatedAt       time.Time
}

type Repository interface {
	GetSummary(ctx context.Context, userID string) (DashboardSummary, error)
	ListContent(ctx context.Context, userID string, query ContentQuery) (ContentPage, error)
	CreateGoal(ctx context.Context, input GoalCreateInput, now time.Time) (Goal, error)
}
