package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/voting-engine/domain/errors"
	"solomon/contexts/campaign-editorial/voting-engine/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	votes       map[string]entities.Vote
	idempotency map[string]ports.IdempotencyRecord
}

func NewStore(seed []entities.Vote) *Store {
	votes := make(map[string]entities.Vote, len(seed))
	for _, vote := range seed {
		votes[vote.VoteID] = vote
	}
	return &Store{
		votes:       votes,
		idempotency: make(map[string]ports.IdempotencyRecord),
	}
}

func (s *Store) SaveVote(_ context.Context, vote entities.Vote) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.votes[vote.VoteID] = vote
	return nil
}

func (s *Store) GetVote(_ context.Context, voteID string) (entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vote, ok := s.votes[strings.TrimSpace(voteID)]
	if !ok {
		return entities.Vote{}, domainerrors.ErrVoteNotFound
	}
	return vote, nil
}

func (s *Store) GetVoteByIdentity(_ context.Context, submissionID string, userID string) (entities.Vote, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, vote := range s.votes {
		if vote.SubmissionID == strings.TrimSpace(submissionID) && vote.UserID == strings.TrimSpace(userID) {
			return vote, true, nil
		}
	}
	return entities.Vote{}, false, nil
}

func (s *Store) ListVotesBySubmission(_ context.Context, submissionID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Vote, 0)
	for _, vote := range s.votes {
		if vote.SubmissionID == strings.TrimSpace(submissionID) {
			items = append(items, vote)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) ListVotesByCampaign(_ context.Context, campaignID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Vote, 0)
	for _, vote := range s.votes {
		if strings.TrimSpace(campaignID) == "" || vote.CampaignID == strings.TrimSpace(campaignID) {
			items = append(items, vote)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) Get(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.idempotency[key]
	if !exists {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.After(now) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.idempotency[record.Key]
	if exists && existing.RequestHash != record.RequestHash {
		return domainerrors.ErrConflict
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
