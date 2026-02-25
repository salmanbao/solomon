package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
)

type VoteRepository interface {
	SaveVote(ctx context.Context, vote entities.Vote) error
	GetVote(ctx context.Context, voteID string) (entities.Vote, error)
	GetVoteByIdentity(ctx context.Context, submissionID string, userID string) (entities.Vote, bool, error)
	ListVotesBySubmission(ctx context.Context, submissionID string) ([]entities.Vote, error)
	ListVotesByCampaign(ctx context.Context, campaignID string) ([]entities.Vote, error)
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	VoteID      string
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
