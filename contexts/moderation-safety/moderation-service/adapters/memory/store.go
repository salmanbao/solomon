package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/moderation-safety/moderation-service/domain/errors"
	"solomon/contexts/moderation-safety/moderation-service/ports"
)

type Store struct {
	mu sync.RWMutex

	queue       map[string]ports.QueueItem
	decisions   map[string]ports.DecisionRecord
	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	return &Store{
		queue: map[string]ports.QueueItem{
			"sub-1": {
				SubmissionID: "sub-1",
				CampaignID:   "camp-1",
				CreatorID:    "creator-1",
				Status:       "pending",
				RiskScore:    0.78,
				ReportCount:  2,
				QueuedAt:     now.Add(-2 * time.Hour),
			},
			"sub-2": {
				SubmissionID: "sub-2",
				CampaignID:   "camp-2",
				CreatorID:    "creator-2",
				Status:       "pending",
				RiskScore:    0.55,
				ReportCount:  0,
				QueuedAt:     now.Add(-4 * time.Hour),
			},
		},
		decisions:   map[string]ports.DecisionRecord{},
		idempotency: map[string]ports.IdempotencyRecord{},
		sequence:    1,
	}
}

func (s *Store) ListQueue(ctx context.Context, filter ports.QueueFilter) ([]ports.QueueItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ports.QueueItem, 0, len(s.queue))
	for _, item := range s.queue {
		if filter.Status != "" && !strings.EqualFold(item.Status, filter.Status) {
			continue
		}
		items = append(items, item)
	}
	if filter.Offset >= len(items) {
		return []ports.QueueItem{}, nil
	}
	end := filter.Offset + filter.Limit
	if end > len(items) {
		end = len(items)
	}
	return append([]ports.QueueItem(nil), items[filter.Offset:end]...), nil
}

func (s *Store) RecordDecision(ctx context.Context, record ports.DecisionRecord, now time.Time) (ports.DecisionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.queue[record.SubmissionID]
	if !ok {
		return ports.DecisionRecord{}, domainerrors.ErrNotFound
	}
	record.DecisionID = s.nextID("decision")
	record.CreatedAt = now.UTC()
	switch record.Action {
	case "approved":
		item.Status = "approved"
	case "rejected":
		item.Status = "rejected"
	case "flagged":
		item.Status = "flagged"
	default:
		return ports.DecisionRecord{}, domainerrors.ErrInvalidRequest
	}
	item.AssignedModeratorID = record.ModeratorID
	s.queue[record.SubmissionID] = item
	record.QueueStatus = item.Status
	s.decisions[record.DecisionID] = record
	return record, nil
}

func (s *Store) ApproveSubmission(ctx context.Context, submissionID string, moderatorID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.queue[submissionID]
	if !ok {
		return domainerrors.ErrNotFound
	}
	item.Status = "approved"
	item.AssignedModeratorID = moderatorID
	s.queue[submissionID] = item
	return nil
}

func (s *Store) RejectSubmission(ctx context.Context, submissionID string, moderatorID string, reason string, notes string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.queue[submissionID]
	if !ok {
		return domainerrors.ErrNotFound
	}
	item.Status = "rejected"
	item.AssignedModeratorID = moderatorID
	s.queue[submissionID] = item
	return nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%d", prefix, n)
}

var _ ports.Repository = (*Store)(nil)
var _ ports.SubmissionDecisionClient = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
