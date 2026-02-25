package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

type SubmissionFilter struct {
	CreatorID  string
	CampaignID string
	Status     entities.SubmissionStatus
}

type CampaignForSubmission struct {
	CampaignID       string
	Status           string
	AllowedPlatforms []string
	RatePer1KViews   float64
}

type Repository interface {
	CreateSubmission(ctx context.Context, submission entities.Submission) error
	UpdateSubmission(ctx context.Context, submission entities.Submission) error
	GetSubmission(ctx context.Context, submissionID string) (entities.Submission, error)
	ListSubmissions(ctx context.Context, filter SubmissionFilter) ([]entities.Submission, error)
	AddReport(ctx context.Context, report entities.SubmissionReport) error
	AddFlag(ctx context.Context, flag entities.SubmissionFlag) error
	AddAudit(ctx context.Context, audit entities.SubmissionAudit) error
	AddBulkOperation(ctx context.Context, operation entities.BulkSubmissionOperation) error
	AddViewSnapshot(ctx context.Context, snapshot entities.ViewSnapshot) error
}

type CampaignReadRepository interface {
	GetCampaignForSubmission(ctx context.Context, campaignID string) (CampaignForSubmission, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
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

type EventEnvelope = contractsv1.Envelope

type OutboxMessage struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
}

type OutboxWriter interface {
	AppendOutbox(ctx context.Context, envelope EventEnvelope) error
}

type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, event EventEnvelope) error
}

type EventSubscriber interface {
	Subscribe(
		ctx context.Context,
		topic string,
		consumerGroup string,
		handler func(context.Context, EventEnvelope) error,
	) error
}

type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}

type AutoApproveRepository interface {
	ListPendingAutoApprove(ctx context.Context, threshold time.Time, limit int) ([]entities.Submission, error)
}

type ViewLockRepository interface {
	ListDueViewLock(ctx context.Context, threshold time.Time, limit int) ([]entities.Submission, error)
}
