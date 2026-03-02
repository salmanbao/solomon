package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/influencer-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/ports"
)

type Store struct {
	mu sync.RWMutex

	FailRewardProvider       bool
	FailGamificationProvider bool

	summaries   map[string]ports.DashboardSummary
	content     map[string][]ports.ContentItem
	goals       map[string][]ports.Goal
	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	payoutDate := now.AddDate(0, 0, 10)
	published := now.Add(-24 * time.Hour)
	return &Store{
		summaries: map[string]ports.DashboardSummary{
			"creator-1": {
				QuickStats: ports.QuickStats{
					TotalViews:    847300,
					TotalEarnings: 2486.50,
					AverageCPV:    4.95,
					SuccessRate:   94.2,
				},
				TopClips: []ports.TopClip{
					{
						ID:             "clip-1",
						Title:          "Trending Fitness Routine",
						ThumbnailURL:   "https://cdn.whop.dev/thumbs/clip-1.jpg",
						Views:          145200,
						Earnings:       726,
						EngagementRate: 5.2,
						PublishedAt:    published,
					},
				},
				UpcomingPayouts: []ports.UpcomingPayout{
					{
						ID:     "pay-1",
						Date:   payoutDate,
						Amount: 1234.56,
						Status: "scheduled",
						Method: "stripe_connect",
					},
				},
			},
		},
		content: map[string][]ports.ContentItem{
			"creator-1": {
				{
					ID:             "cnt-1",
					Title:          "Fitness Challenge",
					ThumbnailURL:   "https://cdn.whop.dev/thumbs/cnt-1.jpg",
					Status:         "published",
					Views:          45200,
					Earnings:       226.00,
					EngagementRate: 3.2,
					ClaimedAt:      now.Add(-48 * time.Hour),
					PublishedAt:    &published,
				},
				{
					ID:             "cnt-2",
					Title:          "Tech Review",
					ThumbnailURL:   "https://cdn.whop.dev/thumbs/cnt-2.jpg",
					Status:         "scheduled",
					Views:          0,
					Earnings:       0,
					EngagementRate: 0,
					ClaimedAt:      now.Add(-8 * time.Hour),
				},
			},
		},
		goals:       map[string][]ports.Goal{},
		idempotency: map[string]ports.IdempotencyRecord{},
		sequence:    1,
	}
}

func (s *Store) GetSummary(ctx context.Context, userID string) (ports.DashboardSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if summary, ok := s.summaries[userID]; ok {
		return summary, nil
	}
	return ports.DashboardSummary{}, domainerrors.ErrNotFound
}

func (s *Store) ListContent(ctx context.Context, userID string, query ports.ContentQuery) (ports.ContentPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := append([]ports.ContentItem(nil), s.content[userID]...)
	if query.Status != "" {
		filtered := make([]ports.ContentItem, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(item.Status, query.Status) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	switch query.SortBy {
	case "views":
		sort.SliceStable(items, func(i, j int) bool { return items[i].Views > items[j].Views })
	case "earnings":
		sort.SliceStable(items, func(i, j int) bool { return items[i].Earnings > items[j].Earnings })
	case "date_claimed":
		sort.SliceStable(items, func(i, j int) bool { return items[i].ClaimedAt.After(items[j].ClaimedAt) })
	case "status":
		sort.SliceStable(items, func(i, j int) bool { return items[i].Status < items[j].Status })
	}
	total := len(items)
	if query.Offset >= total {
		return ports.ContentPage{TotalCount: total, Items: []ports.ContentItem{}}, nil
	}
	end := query.Offset + query.Limit
	if end > total {
		end = total
	}
	return ports.ContentPage{
		TotalCount: total,
		Items:      append([]ports.ContentItem(nil), items[query.Offset:end]...),
	}, nil
}

func (s *Store) CreateGoal(ctx context.Context, input ports.GoalCreateInput, now time.Time) (ports.Goal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	goal := ports.Goal{
		ID:              s.nextID("goal"),
		UserID:          input.UserID,
		GoalType:        input.GoalType,
		GoalName:        input.GoalName,
		TargetValue:     input.TargetValue,
		CurrentValue:    0,
		ProgressPercent: 0,
		Status:          "active",
		StartDate:       input.StartDate,
		EndDate:         input.EndDate,
		CreatedAt:       now.UTC(),
	}
	s.goals[input.UserID] = append(s.goals[input.UserID], goal)
	return goal, nil
}

func (s *Store) GetRewardSnapshot(ctx context.Context, userID string) (ports.RewardSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.FailRewardProvider {
		return ports.RewardSnapshot{}, domainerrors.ErrDependencyUnavailable
	}
	summary, ok := s.summaries[userID]
	if !ok {
		return ports.RewardSnapshot{}, domainerrors.ErrNotFound
	}
	return ports.RewardSnapshot{
		Available: summary.QuickStats.TotalEarnings * 0.7,
		Pending:   summary.QuickStats.TotalEarnings * 0.3,
		Currency:  "USD",
	}, nil
}

func (s *Store) GetGamificationSnapshot(ctx context.Context, userID string) (ports.GamificationSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.FailGamificationProvider {
		return ports.GamificationSnapshot{}, domainerrors.ErrDependencyUnavailable
	}
	if _, ok := s.summaries[userID]; !ok {
		return ports.GamificationSnapshot{}, domainerrors.ErrNotFound
	}
	return ports.GamificationSnapshot{
		Level:  8,
		Points: 2420,
		Badges: []string{"consistency", "viral"},
	}, nil
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
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.RewardProvider = (*Store)(nil)
var _ ports.GamificationProvider = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
