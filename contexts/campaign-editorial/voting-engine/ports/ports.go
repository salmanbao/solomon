package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

type SubmissionProjection struct {
	SubmissionID string
	CampaignID   string
	CreatorID    string
	Status       string
}

// CampaignProjection is the DBR read model used to validate vote eligibility
// against current campaign status without taking ownership of campaign writes.
type CampaignProjection struct {
	CampaignID string
	Status     string
}

// VoteRepository is the primary persistence boundary for M08. Implementations
// must enforce single-writer behavior for vote-owned records and only readonly
// access for foreign projections.
type VoteRepository interface {
	SaveVote(ctx context.Context, vote entities.Vote) error
	GetVote(ctx context.Context, voteID string) (entities.Vote, error)
	GetVoteByIdentity(ctx context.Context, submissionID string, userID string, roundID string) (entities.Vote, bool, error)
	ListVotesBySubmission(ctx context.Context, submissionID string) ([]entities.Vote, error)
	ListVotesByCampaign(ctx context.Context, campaignID string) ([]entities.Vote, error)
	ListVotesByRound(ctx context.Context, roundID string) ([]entities.Vote, error)
	ListVotesByCreator(ctx context.Context, creatorID string) ([]entities.Vote, error)
	ListVotes(ctx context.Context) ([]entities.Vote, error)

	GetSubmission(ctx context.Context, submissionID string) (SubmissionProjection, error)
	GetCampaign(ctx context.Context, campaignID string) (CampaignProjection, error)
	GetReputationScore(ctx context.Context, userID string) (float64, bool, error)

	GetRound(ctx context.Context, roundID string) (entities.VotingRound, error)
	GetActiveRoundByCampaign(ctx context.Context, campaignID string) (entities.VotingRound, bool, error)
	TransitionRoundsForCampaign(
		ctx context.Context,
		campaignID string,
		toStatus entities.RoundStatus,
		updatedAt time.Time,
	) ([]entities.VotingRound, error)

	GetQuarantine(ctx context.Context, quarantineID string) (entities.VoteQuarantine, error)
	SaveQuarantine(ctx context.Context, quarantine entities.VoteQuarantine) error
	ListQuarantines(ctx context.Context) ([]entities.VoteQuarantine, error)

	RetractVotesBySubmission(ctx context.Context, submissionID string, updatedAt time.Time) ([]entities.Vote, error)
}

// IdempotencyRecord persists request fingerprinting for replay-safe commands.
type IdempotencyRecord struct {
	Key         string
	RequestHash string
	VoteID      string
	ExpiresAt   time.Time
}

// IdempotencyStore prevents duplicate side effects for command handlers.
type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

// OutboxMessage is the durable outbox row shape consumed by relay workers.
type OutboxMessage struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
}

// EventEnvelope aliases the canonical v1 event contract used across modules.
type EventEnvelope = contractsv1.Envelope

// OutboxWriter appends canonical envelopes to durable storage in the same
// transactional boundary as state changes where possible.
type OutboxWriter interface {
	AppendOutbox(ctx context.Context, envelope EventEnvelope) error
}

// OutboxRepository exposes relay operations for unpublished outbox records.
type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error
}

// EventPublisher publishes envelopes to the broker/topic abstraction.
type EventPublisher interface {
	Publish(ctx context.Context, topic string, event EventEnvelope) error
}

// EventSubscriber subscribes to external lifecycle events consumed by M08.
type EventSubscriber interface {
	Subscribe(
		ctx context.Context,
		topic string,
		consumerGroup string,
		handler func(context.Context, EventEnvelope) error,
	) error
}

// EventDedupStore tracks consumed event fingerprints to keep consumers
// idempotent under at-least-once delivery.
type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}

// Clock allows deterministic time control in tests and workers.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts vote/outbox identifier generation.
type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
