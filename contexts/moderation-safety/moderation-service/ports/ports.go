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

type QueueFilter struct {
	Status string
	Limit  int
	Offset int
}

type QueueItem struct {
	SubmissionID        string
	CampaignID          string
	CreatorID           string
	Status              string
	RiskScore           float64
	ReportCount         int
	QueuedAt            time.Time
	AssignedModeratorID string
}

type ModerationActionInput struct {
	SubmissionID string
	CampaignID   string
	Reason       string
	Notes        string
	Severity     string
}

type DecisionRecord struct {
	DecisionID   string
	SubmissionID string
	CampaignID   string
	ModeratorID  string
	Action       string
	Reason       string
	Notes        string
	Severity     string
	CreatedAt    time.Time
	QueueStatus  string
}

type SubmissionDecisionClient interface {
	ApproveSubmission(ctx context.Context, submissionID string, moderatorID string, reason string) error
	RejectSubmission(ctx context.Context, submissionID string, moderatorID string, reason string, notes string) error
}

type Repository interface {
	ListQueue(ctx context.Context, filter QueueFilter) ([]QueueItem, error)
	RecordDecision(ctx context.Context, record DecisionRecord, now time.Time) (DecisionRecord, error)
}
