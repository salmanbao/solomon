package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

type Repository interface {
	CreateItem(ctx context.Context, item entities.DistributionItem) error
	UpdateItem(ctx context.Context, item entities.DistributionItem) error
	GetItem(ctx context.Context, itemID string) (entities.DistributionItem, error)
	ListItemsByInfluencer(ctx context.Context, influencerID string) ([]entities.DistributionItem, error)
	ListDueScheduled(ctx context.Context, threshold time.Time, limit int) ([]entities.DistributionItem, error)
	GetCampaignIDByClip(ctx context.Context, clipID string) (string, error)
	AddOverlay(ctx context.Context, overlay entities.Overlay) error
	UpsertCaption(ctx context.Context, caption entities.Caption) error
	UpsertPlatformStatus(ctx context.Context, status entities.PlatformStatus) error
	AddPublishingAnalytics(ctx context.Context, analytics entities.PublishingAnalytics) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type EventEnvelope = contractsv1.Envelope

type OutboxWriter interface {
	AppendOutbox(ctx context.Context, envelope EventEnvelope) error
}

type OutboxMessage struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
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
