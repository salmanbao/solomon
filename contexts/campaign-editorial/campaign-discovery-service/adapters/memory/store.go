package memory

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/campaign-discovery-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/ports"
)

type Store struct {
	mu sync.RWMutex

	campaigns   map[string]ports.CampaignSummary
	bookmarks   map[string]map[string]ports.BookmarkRecord
	idempotency map[string]ports.IdempotencyRecord
	creatorTier map[string]string
	sequence    uint64
}

func NewStore() *Store {
	return &Store{
		campaigns: map[string]ports.CampaignSummary{
			"c-12345678-9abc-def0-1234-56789abcdef0": {
				CampaignID:       "c-12345678-9abc-def0-1234-56789abcdef0",
				Title:            "Summer Fitness Challenge",
				Description:      "Promote our new fitness app with your unique spin.",
				CreatorName:      "FitBrand Co",
				CreatorTier:      "gold",
				BudgetTotal:      2500,
				BudgetSpent:      1250,
				BudgetCurrency:   "USD",
				RatePer1KViews:   2.5,
				EstimatedViews:   500000,
				EstimatedEarning: 1250,
				SubmissionCount:  48,
				ApprovalRate:     0.92,
				Deadline:         "2026-03-15",
				Category:         "fitness",
				Platforms:        []string{"tiktok", "instagram", "youtube"},
				State:            "active",
				IsFeatured:       true,
				FeaturedUntil:    "2026-03-12",
				MatchScore:       0.88,
				IsEligible:       true,
				TrendingStatus:   "trending",
				CreatedAt:        "2026-01-20",
				CombinedScore:    95.5,
			},
			"c-11111111-2222-3333-4444-555555555555": {
				CampaignID:       "c-11111111-2222-3333-4444-555555555555",
				Title:            "Tech Product Launch",
				Description:      "Create unboxing content for our new tech product.",
				CreatorName:      "TechCorp",
				CreatorTier:      "silver",
				BudgetTotal:      5000,
				BudgetSpent:      500,
				BudgetCurrency:   "USD",
				RatePer1KViews:   3.0,
				EstimatedViews:   1500000,
				EstimatedEarning: 4500,
				SubmissionCount:  12,
				ApprovalRate:     0.95,
				Deadline:         "2026-04-01",
				Category:         "tech",
				Platforms:        []string{"youtube", "instagram"},
				State:            "active",
				IsFeatured:       false,
				MatchScore:       0.76,
				IsEligible:       true,
				CreatedAt:        "2026-02-01",
				CombinedScore:    79.5,
			},
			"c-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee": {
				CampaignID:       "c-aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Title:            "Wellness Reset Week",
				Description:      "Share your reset routine and daily healthy habits.",
				CreatorName:      "Wellness Labs",
				CreatorTier:      "platinum",
				BudgetTotal:      3200,
				BudgetSpent:      800,
				BudgetCurrency:   "USD",
				RatePer1KViews:   1.9,
				EstimatedViews:   900000,
				EstimatedEarning: 2400,
				SubmissionCount:  31,
				ApprovalRate:     0.9,
				Deadline:         "2026-03-22",
				Category:         "wellness",
				Platforms:        []string{"instagram", "tiktok"},
				State:            "active",
				IsFeatured:       false,
				MatchScore:       0.82,
				IsEligible:       true,
				CreatedAt:        "2026-01-28",
				CombinedScore:    83.4,
			},
		},
		bookmarks:   map[string]map[string]ports.BookmarkRecord{},
		idempotency: map[string]ports.IdempotencyRecord{},
		creatorTier: map[string]string{
			"FitBrand Co":   "gold",
			"TechCorp":      "silver",
			"Wellness Labs": "platinum",
		},
		sequence: 1,
	}
}

func (s *Store) BrowseCampaigns(ctx context.Context, query ports.BrowseQuery) (ports.BrowseResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ports.CampaignSummary, 0, len(s.campaigns))
	for _, item := range s.campaigns {
		if !matchesBrowseFilters(item, query.Filters) {
			continue
		}
		if bookmarks, ok := s.bookmarks[query.UserID]; ok {
			_, item.UserSaved = bookmarks[item.CampaignID]
		}
		items = append(items, item)
	}
	sortCampaigns(items, query.SortBy)

	start := decodeCursor(query.Cursor)
	if start < 0 {
		start = 0
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + query.PageSize
	if end > len(items) {
		end = len(items)
	}
	page := append([]ports.CampaignSummary(nil), items[start:end]...)

	next := ""
	prev := ""
	if end < len(items) {
		next = encodeCursor(end)
	}
	if start > 0 {
		prev = encodeCursor(max(start-query.PageSize, 0))
	}

	return ports.BrowseResult{
		Campaigns: page,
		Pagination: ports.Pagination{
			NextCursor:     next,
			PrevCursor:     prev,
			HasNext:        end < len(items),
			HasPrev:        start > 0,
			TotalEstimated: len(items),
			PageSize:       query.PageSize,
		},
		Summary: ports.BrowseSummary{
			ResultCount:  len(page),
			SearchTimeMS: 20,
			CacheHit:     true,
		},
	}, nil
}

func (s *Store) SearchCampaigns(ctx context.Context, query ports.SearchQuery) (ports.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	needle := strings.ToLower(strings.TrimSpace(query.Query))
	items := make([]ports.SearchResultItem, 0, len(s.campaigns))
	for _, item := range s.campaigns {
		if query.Category != "" && !strings.EqualFold(item.Category, query.Category) {
			continue
		}
		if query.BudgetMin > 0 && item.BudgetTotal < query.BudgetMin {
			continue
		}
		if !strings.Contains(strings.ToLower(item.Title), needle) &&
			!strings.Contains(strings.ToLower(item.Description), needle) {
			continue
		}
		items = append(items, ports.SearchResultItem{
			CampaignID:      item.CampaignID,
			Title:           item.Title,
			Description:     item.Description,
			CreatorName:     item.CreatorName,
			MatchScore:      item.MatchScore,
			Budget:          item.BudgetTotal,
			Deadline:        item.Deadline,
			Category:        item.Category,
			SubmissionCount: item.SubmissionCount,
			IsFeatured:      item.IsFeatured,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].MatchScore > items[j].MatchScore
	})

	if query.Offset > len(items) {
		query.Offset = len(items)
	}
	end := query.Offset + query.Limit
	if end > len(items) {
		end = len(items)
	}
	page := append([]ports.SearchResultItem(nil), items[query.Offset:end]...)

	return ports.SearchResult{
		Items:         page,
		Total:         len(items),
		Limit:         query.Limit,
		Offset:        query.Offset,
		HasNext:       end < len(items),
		ExecutionTime: 18,
		IndexVersion:  "2026-03-02T00:00:00Z",
	}, nil
}

func (s *Store) GetCampaignDetails(ctx context.Context, userID string, campaignID string) (ports.CampaignDetails, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.campaigns[campaignID]
	if !ok {
		return ports.CampaignDetails{}, domainerrors.ErrNotFound
	}
	if bookmarks, ok := s.bookmarks[userID]; ok {
		_, item.UserSaved = bookmarks[campaignID]
	}
	return ports.CampaignDetails{Campaign: item}, nil
}

func (s *Store) SaveBookmark(ctx context.Context, command ports.BookmarkCommand, now time.Time) (ports.BookmarkRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.campaigns[command.CampaignID]; !ok {
		return ports.BookmarkRecord{}, domainerrors.ErrNotFound
	}
	if _, ok := s.bookmarks[command.UserID]; !ok {
		s.bookmarks[command.UserID] = map[string]ports.BookmarkRecord{}
	}
	record := ports.BookmarkRecord{
		BookmarkID: "bm_" + s.nextID(),
		UserID:     command.UserID,
		CampaignID: command.CampaignID,
		Tag:        command.Tag,
		Note:       command.Note,
		CreatedAt:  now.UTC(),
	}
	if existing, ok := s.bookmarks[command.UserID][command.CampaignID]; ok {
		record.BookmarkID = existing.BookmarkID
	}
	s.bookmarks[command.UserID][command.CampaignID] = record
	return record, nil
}

func (s *Store) GetCampaignProjections(ctx context.Context, campaignIDs []string) (map[string]ports.CampaignProjection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]ports.CampaignProjection, len(campaignIDs))
	for _, campaignID := range campaignIDs {
		item, ok := s.campaigns[campaignID]
		if !ok {
			continue
		}
		out[campaignID] = ports.CampaignProjection{
			CampaignID:      campaignID,
			State:           item.State,
			BudgetRemaining: item.BudgetTotal - item.BudgetSpent,
			SubmissionCount: item.SubmissionCount,
		}
	}
	return out, nil
}

func (s *Store) GetCreatorTiers(ctx context.Context, creatorIDs []string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]string, len(creatorIDs))
	for _, creatorID := range creatorIDs {
		if tier, ok := s.creatorTier[creatorID]; ok {
			out[creatorID] = tier
		}
	}
	return out, nil
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

func (s *Store) nextID() string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%d", n)
}

func matchesBrowseFilters(item ports.CampaignSummary, filters ports.BrowseFilters) bool {
	if filters.Category != "" && !strings.EqualFold(item.Category, filters.Category) {
		return false
	}
	if filters.State != "" && !strings.EqualFold(item.State, filters.State) {
		return false
	}
	if filters.ExcludeFeatured && item.IsFeatured {
		return false
	}
	if filters.BudgetMin > 0 && item.BudgetTotal < filters.BudgetMin {
		return false
	}
	if filters.BudgetMax > 0 && item.BudgetTotal > filters.BudgetMax {
		return false
	}
	if len(filters.Platforms) > 0 {
		ok := false
		for _, expected := range filters.Platforms {
			for _, actual := range item.Platforms {
				if strings.EqualFold(strings.TrimSpace(expected), actual) {
					ok = true
					break
				}
			}
		}
		if !ok {
			return false
		}
	}
	deadline, err := time.Parse("2006-01-02", item.Deadline)
	if err == nil {
		if filters.DeadlineAfter != nil && deadline.Before(filters.DeadlineAfter.UTC()) {
			return false
		}
		if filters.DeadlineBefore != nil && deadline.After(filters.DeadlineBefore.UTC()) {
			return false
		}
	}
	return true
}

func sortCampaigns(items []ports.CampaignSummary, sortBy string) {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "budget":
		sort.SliceStable(items, func(i, j int) bool {
			iRemaining := items[i].BudgetTotal - items[i].BudgetSpent
			jRemaining := items[j].BudgetTotal - items[j].BudgetSpent
			return iRemaining > jRemaining
		})
	case "deadline":
		sort.SliceStable(items, func(i, j int) bool {
			iDeadline, iErr := time.Parse("2006-01-02", items[i].Deadline)
			jDeadline, jErr := time.Parse("2006-01-02", items[j].Deadline)
			if iErr != nil || jErr != nil {
				return items[i].CombinedScore > items[j].CombinedScore
			}
			return iDeadline.Before(jDeadline)
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].CombinedScore > items[j].CombinedScore
		})
	}
}

func decodeCursor(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func encodeCursor(offset int) string {
	if offset < 0 {
		return ""
	}
	return strconv.Itoa(offset)
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.CampaignProjectionProvider = (*Store)(nil)
var _ ports.ReputationProjectionProvider = (*Store)(nil)
