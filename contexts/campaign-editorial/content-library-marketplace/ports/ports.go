package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

// ClipListFilter defines read-side filtering/pagination for the clip catalog.
type ClipListFilter struct {
	Niches         []string
	DurationBucket string
	Status         entities.ClipStatus
	Cursor         string
	Limit          int
	Popularity     string
}

// ClipRepository encapsulates read-only access to clip data owned by M09.
type ClipRepository interface {
	ListClips(ctx context.Context, filter ClipListFilter) ([]entities.Clip, string, error)
	GetClip(ctx context.Context, clipID string) (entities.Clip, error)
}

// ClaimedEvent is the outbound integration payload persisted to outbox.
type ClaimedEvent struct {
	EventID      string
	EventType    string
	ClaimID      string
	ClipID       string
	UserID       string
	ClaimType    string
	PartitionKey string
	OccurredAt   time.Time
}

// ClaimRepository owns claim persistence and transaction boundaries for claim writes.
type ClaimRepository interface {
	ListClaimsByUser(ctx context.Context, userID string) ([]entities.Claim, error)
	ListClaimsByClip(ctx context.Context, clipID string) ([]entities.Claim, error)
	GetClaim(ctx context.Context, claimID string) (entities.Claim, error)
	GetClaimByRequestID(ctx context.Context, requestID string) (entities.Claim, bool, error)
	// CreateClaimWithOutbox must atomically persist the claim and outbox event.
	CreateClaimWithOutbox(ctx context.Context, claim entities.Claim, event ClaimedEvent) error
	// UpdateClaimStatus is used by async consumers to apply distribution outcomes.
	UpdateClaimStatus(ctx context.Context, claimID string, status entities.ClaimStatus, updatedAt time.Time) error
	// ExpireActiveClaims transitions active claims that passed expiry.
	ExpireActiveClaims(ctx context.Context, now time.Time) (int, error)
}

// IdempotencyRecord captures dedupe metadata for mutating requests.
type IdempotencyRecord struct {
	Key         string
	RequestHash string
	ClaimID     string
	ExpiresAt   time.Time
}

// IdempotencyStore abstracts idempotency persistence with TTL handling.
type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

// ClipDownload captures a successful download issuance audit record.
type ClipDownload struct {
	DownloadID   string
	ClipID       string
	UserID       string
	IPAddress    string
	UserAgent    string
	DownloadedAt time.Time
}

// DownloadRepository persists and queries download history rows.
type DownloadRepository interface {
	CountUserClipDownloadsSince(ctx context.Context, userID string, clipID string, since time.Time) (int, error)
	CreateDownload(ctx context.Context, download ClipDownload) error
}

// Clock allows deterministic testing of TTL/expiry rules.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts claim/event identifier generation.
type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

// OutboxMessage is a row ready to relay from the module outbox.
type OutboxMessage struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
}

// OutboxRepository models worker-side outbox polling/acknowledgement.
type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxSent(ctx context.Context, outboxID string, sentAt time.Time) error
}

// EventDedupStore provides idempotent processing guarantees for consumed events.
type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}

// EventEnvelope reuses the canonical cross-runtime envelope contract.
type EventEnvelope = contractsv1.Envelope

// EventPublisher publishes canonical envelopes to a topic.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event EventEnvelope) error
}

// EventSubscriber registers a topic consumer callback.
type EventSubscriber interface {
	Subscribe(
		ctx context.Context,
		topic string,
		consumerGroup string,
		handler func(context.Context, EventEnvelope) error,
	) error
}
