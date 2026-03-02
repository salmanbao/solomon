package memory

import (
	"bytes"
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	domainerrors "solomon/contexts/community-experience/gamification-service/domain/errors"
	"solomon/contexts/community-experience/gamification-service/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	projections map[string]ports.UserProjection
	points      map[string]ports.UserPoints
	pointsLog   []ports.PointsLog
	badges      map[string]map[string]ports.BadgeGrant
	idempotency map[string]ports.IdempotencyRecord
}

func NewStore(seed []ports.UserProjection) *Store {
	projections := make(map[string]ports.UserProjection, len(seed))
	for _, item := range seed {
		projections[strings.TrimSpace(item.UserID)] = item
	}
	return &Store{
		projections: projections,
		points:      make(map[string]ports.UserPoints),
		pointsLog:   make([]ports.PointsLog, 0),
		badges:      make(map[string]map[string]ports.BadgeGrant),
		idempotency: make(map[string]ports.IdempotencyRecord),
	}
}

func (s *Store) SeedUserProjection(item ports.UserProjection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projections[strings.TrimSpace(item.UserID)] = item
}

func (s *Store) GetUserProjection(_ context.Context, userID string) (ports.UserProjection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.projections[strings.TrimSpace(userID)]
	if !ok {
		return ports.UserProjection{}, domainerrors.ErrDependencyUnavailable
	}
	return item, nil
}

func (s *Store) AppendPointsLog(_ context.Context, log ports.PointsLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pointsLog = append(s.pointsLog, log)
	return nil
}

func (s *Store) IncrementUserPoints(_ context.Context, userID string, delta int, updatedAt time.Time) (ports.UserPoints, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(userID)
	item := s.points[key]
	item.UserID = key
	item.TotalPoints += delta
	if item.TotalPoints < 0 {
		item.TotalPoints = 0
	}
	item.UpdatedAt = updatedAt.UTC()
	s.points[key] = item
	return item, nil
}

func (s *Store) UpsertBadge(_ context.Context, grant ports.BadgeGrant) (ports.BadgeGrant, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID := strings.TrimSpace(grant.UserID)
	badgeKey := strings.TrimSpace(grant.BadgeKey)
	if userID == "" || badgeKey == "" {
		return ports.BadgeGrant{}, false, domainerrors.ErrInvalidInput
	}
	if _, ok := s.badges[userID]; !ok {
		s.badges[userID] = make(map[string]ports.BadgeGrant)
	}
	existing, ok := s.badges[userID][badgeKey]
	if ok {
		return existing, true, nil
	}
	s.badges[userID][badgeKey] = grant
	return grant, false, nil
}

func (s *Store) ListUserBadges(_ context.Context, userID string) ([]ports.BadgeGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID = strings.TrimSpace(userID)
	badges := s.badges[userID]
	items := make([]ports.BadgeGrant, 0, len(badges))
	for _, item := range badges {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GrantedAt.After(items[j].GrantedAt)
	})
	return items, nil
}

func (s *Store) GetUserPoints(_ context.Context, userID string) (ports.UserPoints, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item := s.points[strings.TrimSpace(userID)]
	item.UserID = strings.TrimSpace(userID)
	item.CurrentLevel = levelForPoints(item.TotalPoints)
	return item, nil
}

func (s *Store) ListLeaderboard(_ context.Context, limit int, offset int) ([]ports.LeaderboardEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	items := make([]ports.LeaderboardEntry, 0, len(s.points))
	for _, point := range s.points {
		items = append(items, ports.LeaderboardEntry{
			UserID:       point.UserID,
			TotalPoints:  point.TotalPoints,
			CurrentLevel: levelForPoints(point.TotalPoints),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalPoints == items[j].TotalPoints {
			return items[i].UserID < items[j].UserID
		}
		return items[i].TotalPoints > items[j].TotalPoints
	})
	for i := range items {
		items[i].Rank = i + 1
	}
	if offset >= len(items) {
		return []ports.LeaderboardEntry{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]ports.LeaderboardEntry(nil), items[offset:end]...), nil
}

func (s *Store) GetRecord(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[strings.TrimSpace(key)]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.After(now.UTC()) {
		delete(s.idempotency, strings.TrimSpace(key))
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) PutRecord(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(record.Key)
	if key == "" {
		return domainerrors.ErrInvalidInput
	}
	if existing, ok := s.idempotency[key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		if !bytes.Equal(existing.ResponsePayload, record.ResponsePayload) {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[key] = record
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}

func levelForPoints(totalPoints int) int {
	if totalPoints <= 0 {
		return 1
	}
	level := 1
	for level < 100 {
		next := 100*level + (50*level*(level-1))/2
		if totalPoints < next {
			break
		}
		level++
	}
	return level
}
