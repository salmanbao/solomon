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

type EventDedupStore interface {
	HasProcessedEvent(ctx context.Context, eventID string, now time.Time) (bool, error)
	MarkProcessedEvent(ctx context.Context, eventID string, expiresAt time.Time) error
}

type FeedQuery struct {
	Limit  int
	Offset int
	SortBy string
}

type FeedItem struct {
	CampaignID       string
	Title            string
	Category         string
	RewardRate       float64
	BudgetRemaining  float64
	MatchScore       float64
	SubmissionStatus string
	Saved            bool
}

type SubmissionQuery struct {
	Status string
	Limit  int
	Offset int
}

type SubmissionRecord struct {
	SubmissionID   string
	CampaignID     string
	CampaignTitle  string
	Status         string
	Views          int
	Earnings       float64
	SubmittedAt    time.Time
	ReviewedAt     *time.Time
	Feedback       string
	RejectionCode  string
	ModerationNote string
}

type EarningsSummary struct {
	Available float64
	Pending   float64
	Lifetime  float64
	Currency  string
}

type PerformanceSummary struct {
	ApprovalRate        float64
	AvgViewsPerClip     float64
	ReputationScore     float64
	BenchmarkPercentile float64
}

type SaveCampaignCommand struct {
	UserID     string
	CampaignID string
}

type SaveCampaignResult struct {
	CampaignID string
	Saved      bool
	SavedAt    *time.Time
}

type SubmissionLifecycleEvent struct {
	EventID      string
	EventType    string
	SubmissionID string
	UserID       string
	Status       string
	OccurredAt   time.Time
}

type Repository interface {
	GetFeed(ctx context.Context, userID string, query FeedQuery) ([]FeedItem, error)
	ListSubmissions(ctx context.Context, userID string, query SubmissionQuery) ([]SubmissionRecord, error)
	GetEarnings(ctx context.Context, userID string) (EarningsSummary, error)
	GetPerformance(ctx context.Context, userID string) (PerformanceSummary, error)
	SaveCampaign(ctx context.Context, command SaveCampaignCommand, now time.Time) (SaveCampaignResult, error)
	RemoveSavedCampaign(ctx context.Context, command SaveCampaignCommand, now time.Time) (SaveCampaignResult, error)
	ExportSubmissionsCSV(ctx context.Context, userID string, query SubmissionQuery) (string, error)
	ApplySubmissionLifecycleEvent(ctx context.Context, event SubmissionLifecycleEvent) error
}
