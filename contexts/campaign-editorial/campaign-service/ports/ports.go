package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

type CampaignFilter struct {
	BrandID string
	Status  entities.CampaignStatus
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign entities.Campaign) error
	UpdateCampaign(ctx context.Context, campaign entities.Campaign) error
	GetCampaign(ctx context.Context, campaignID string) (entities.Campaign, error)
	ListCampaigns(ctx context.Context, filter CampaignFilter) ([]entities.Campaign, error)
}

type MediaRepository interface {
	AddMedia(ctx context.Context, media entities.Media) error
	GetMedia(ctx context.Context, mediaID string) (entities.Media, error)
	UpdateMedia(ctx context.Context, media entities.Media) error
	ListMediaByCampaign(ctx context.Context, campaignID string) ([]entities.Media, error)
}

type HistoryRepository interface {
	AppendState(ctx context.Context, item entities.StateHistory) error
	AppendBudget(ctx context.Context, item entities.BudgetLog) error
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

type SubmissionCreatedResult struct {
	CampaignID           string
	BudgetReservedDelta  float64
	BudgetRemaining      float64
	AutoPaused           bool
	NewStatus            entities.CampaignStatus
}

type SubmissionCreatedRepository interface {
	ApplySubmissionCreated(
		ctx context.Context,
		campaignID string,
		eventID string,
		occurredAt time.Time,
	) (SubmissionCreatedResult, error)
}

type DeadlineCompletionResult struct {
	CampaignID string
}

type DeadlineRepository interface {
	CompleteCampaignsPastDeadline(
		ctx context.Context,
		now time.Time,
		limit int,
	) ([]DeadlineCompletionResult, error)
}
