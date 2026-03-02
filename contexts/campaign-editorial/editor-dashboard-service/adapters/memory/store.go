package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/editor-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/ports"
)

type Store struct {
	mu sync.RWMutex

	feedItems   map[string][]ports.FeedItem
	submissions map[string][]ports.SubmissionRecord
	earnings    map[string]ports.EarningsSummary
	performance map[string]ports.PerformanceSummary
	saved       map[string]map[string]ports.SaveCampaignResult
	idempotency map[string]ports.IdempotencyRecord
	eventDedup  map[string]time.Time
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	reviewed := now.Add(-3 * time.Hour)
	return &Store{
		feedItems: map[string][]ports.FeedItem{
			"editor-1": {
				{CampaignID: "camp-1", Title: "Fitness Sprint", Category: "fitness", RewardRate: 2.5, BudgetRemaining: 640, MatchScore: 0.91, SubmissionStatus: "pending"},
				{CampaignID: "camp-2", Title: "Tech Unbox", Category: "tech", RewardRate: 3.2, BudgetRemaining: 900, MatchScore: 0.83, SubmissionStatus: "approved"},
			},
		},
		submissions: map[string][]ports.SubmissionRecord{
			"editor-1": {
				{SubmissionID: "sub-1", CampaignID: "camp-1", CampaignTitle: "Fitness Sprint", Status: "pending", Views: 1200, Earnings: 3.00, SubmittedAt: now.Add(-6 * time.Hour)},
				{SubmissionID: "sub-2", CampaignID: "camp-2", CampaignTitle: "Tech Unbox", Status: "approved", Views: 15000, Earnings: 48.00, SubmittedAt: now.Add(-48 * time.Hour), ReviewedAt: &reviewed},
			},
		},
		earnings: map[string]ports.EarningsSummary{
			"editor-1": {Available: 48, Pending: 9, Lifetime: 132, Currency: "USD"},
		},
		performance: map[string]ports.PerformanceSummary{
			"editor-1": {ApprovalRate: 0.82, AvgViewsPerClip: 8100, ReputationScore: 74.5, BenchmarkPercentile: 0.67},
		},
		saved:       map[string]map[string]ports.SaveCampaignResult{},
		idempotency: map[string]ports.IdempotencyRecord{},
		eventDedup:  map[string]time.Time{},
		sequence:    1,
	}
}

func (s *Store) GetFeed(ctx context.Context, userID string, query ports.FeedQuery) ([]ports.FeedItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, ok := s.feedItems[userID]
	if !ok {
		return []ports.FeedItem{}, nil
	}
	out := make([]ports.FeedItem, 0, len(items))
	for _, item := range items {
		if userSaved, ok := s.saved[userID]; ok {
			_, item.Saved = userSaved[item.CampaignID]
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].MatchScore > out[j].MatchScore
	})
	if query.Offset >= len(out) {
		return []ports.FeedItem{}, nil
	}
	end := query.Offset + query.Limit
	if end > len(out) {
		end = len(out)
	}
	return append([]ports.FeedItem(nil), out[query.Offset:end]...), nil
}

func (s *Store) ListSubmissions(ctx context.Context, userID string, query ports.SubmissionQuery) ([]ports.SubmissionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := append([]ports.SubmissionRecord(nil), s.submissions[userID]...)
	if query.Status != "" {
		filtered := make([]ports.SubmissionRecord, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(item.Status, query.Status) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if query.Offset >= len(items) {
		return []ports.SubmissionRecord{}, nil
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	return append([]ports.SubmissionRecord(nil), items[query.Offset:end]...), nil
}

func (s *Store) GetEarnings(ctx context.Context, userID string) (ports.EarningsSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if summary, ok := s.earnings[userID]; ok {
		return summary, nil
	}
	return ports.EarningsSummary{Currency: "USD"}, nil
}

func (s *Store) GetPerformance(ctx context.Context, userID string) (ports.PerformanceSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if summary, ok := s.performance[userID]; ok {
		return summary, nil
	}
	return ports.PerformanceSummary{}, nil
}

func (s *Store) SaveCampaign(ctx context.Context, command ports.SaveCampaignCommand, now time.Time) (ports.SaveCampaignResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasCampaignLocked(command.UserID, command.CampaignID) {
		return ports.SaveCampaignResult{}, domainerrors.ErrNotFound
	}
	if _, ok := s.saved[command.UserID]; !ok {
		s.saved[command.UserID] = map[string]ports.SaveCampaignResult{}
	}
	result := ports.SaveCampaignResult{CampaignID: command.CampaignID, Saved: true}
	t := now.UTC()
	result.SavedAt = &t
	s.saved[command.UserID][command.CampaignID] = result
	return result, nil
}

func (s *Store) RemoveSavedCampaign(ctx context.Context, command ports.SaveCampaignCommand, now time.Time) (ports.SaveCampaignResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.saved[command.UserID]; ok {
		delete(s.saved[command.UserID], command.CampaignID)
	}
	return ports.SaveCampaignResult{CampaignID: command.CampaignID, Saved: false}, nil
}

func (s *Store) ExportSubmissionsCSV(ctx context.Context, userID string, query ports.SubmissionQuery) (string, error) {
	items, err := s.ListSubmissions(ctx, userID, query)
	if err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	builder.WriteString("submission_id,campaign_id,campaign_title,status,views,earnings,submitted_at\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("%s,%s,%s,%s,%d,%.2f,%s\n",
			item.SubmissionID,
			item.CampaignID,
			item.CampaignTitle,
			item.Status,
			item.Views,
			item.Earnings,
			item.SubmittedAt.UTC().Format(time.RFC3339),
		))
	}
	return builder.String(), nil
}

func (s *Store) ApplySubmissionLifecycleEvent(ctx context.Context, event ports.SubmissionLifecycleEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.submissions[event.UserID]
	for i := range items {
		if items[i].SubmissionID == event.SubmissionID {
			items[i].Status = event.Status
			t := event.OccurredAt.UTC()
			items[i].ReviewedAt = &t
			s.submissions[event.UserID] = items
			return nil
		}
	}
	s.submissions[event.UserID] = append(s.submissions[event.UserID], ports.SubmissionRecord{
		SubmissionID: event.SubmissionID,
		CampaignID:   "unknown",
		Status:       event.Status,
		SubmittedAt:  event.OccurredAt.UTC(),
	})
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

func (s *Store) HasProcessedEvent(ctx context.Context, eventID string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.eventDedup[eventID]
	if !ok {
		return false, nil
	}
	if now.UTC().After(expiry.UTC()) {
		delete(s.eventDedup, eventID)
		return false, nil
	}
	return true, nil
}

func (s *Store) MarkProcessedEvent(ctx context.Context, eventID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventDedup[eventID] = expiresAt.UTC()
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) hasCampaignLocked(userID string, campaignID string) bool {
	for _, item := range s.feedItems[userID] {
		if item.CampaignID == campaignID {
			return true
		}
	}
	return false
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%d", prefix, n)
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.EventDedupStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
