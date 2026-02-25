package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	items          map[string]entities.DistributionItem
	overlays       map[string]entities.Overlay
	platformStatus map[string]entities.PlatformStatus
}

func NewStore(seed []entities.DistributionItem) *Store {
	items := make(map[string]entities.DistributionItem, len(seed))
	for _, item := range seed {
		items[item.ID] = item
	}
	return &Store{
		items:          items,
		overlays:       make(map[string]entities.Overlay),
		platformStatus: make(map[string]entities.PlatformStatus),
	}
}

func (s *Store) CreateItem(_ context.Context, item entities.DistributionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[item.ID]; exists {
		return domainerrors.ErrInvalidDistributionInput
	}
	s.items[item.ID] = item
	return nil
}

func (s *Store) UpdateItem(_ context.Context, item entities.DistributionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[item.ID]; !exists {
		return domainerrors.ErrDistributionItemNotFound
	}
	s.items[item.ID] = item
	return nil
}

func (s *Store) GetItem(_ context.Context, itemID string) (entities.DistributionItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.items[strings.TrimSpace(itemID)]
	if !exists {
		return entities.DistributionItem{}, domainerrors.ErrDistributionItemNotFound
	}
	return item, nil
}

func (s *Store) ListItemsByInfluencer(_ context.Context, influencerID string) ([]entities.DistributionItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.DistributionItem, 0)
	for _, item := range s.items {
		if item.InfluencerID == strings.TrimSpace(influencerID) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ClaimedAt.After(items[j].ClaimedAt)
	})
	return items, nil
}

func (s *Store) AddOverlay(_ context.Context, overlay entities.Overlay) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[overlay.DistributionItemID]; !exists {
		return domainerrors.ErrDistributionItemNotFound
	}
	s.overlays[overlay.ID] = overlay
	return nil
}

func (s *Store) UpsertPlatformStatus(_ context.Context, status entities.PlatformStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := status.DistributionItemID + "|" + status.Platform
	s.platformStatus[key] = status
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}

var _ ports.Repository = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
