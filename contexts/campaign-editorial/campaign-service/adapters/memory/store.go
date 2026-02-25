package memory

import (
	"bytes"
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	campaigns map[string]entities.Campaign
	media     map[string]entities.Media
	stateLog  []entities.StateHistory
	budgetLog []entities.BudgetLog

	idempotency map[string]ports.IdempotencyRecord
}

func NewStore(seed []entities.Campaign) *Store {
	campaigns := make(map[string]entities.Campaign, len(seed))
	for _, item := range seed {
		campaigns[item.CampaignID] = item
	}
	return &Store{
		campaigns:   campaigns,
		media:       make(map[string]entities.Media),
		stateLog:    make([]entities.StateHistory, 0),
		budgetLog:   make([]entities.BudgetLog, 0),
		idempotency: make(map[string]ports.IdempotencyRecord),
	}
}

func (s *Store) CreateCampaign(_ context.Context, campaign entities.Campaign) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.campaigns[campaign.CampaignID]; exists {
		return domainerrors.ErrInvalidCampaignInput
	}
	s.campaigns[campaign.CampaignID] = campaign
	return nil
}

func (s *Store) UpdateCampaign(_ context.Context, campaign entities.Campaign) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.campaigns[campaign.CampaignID]; !exists {
		return domainerrors.ErrCampaignNotFound
	}
	s.campaigns[campaign.CampaignID] = campaign
	return nil
}

func (s *Store) GetCampaign(_ context.Context, campaignID string) (entities.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.campaigns[strings.TrimSpace(campaignID)]
	if !exists {
		return entities.Campaign{}, domainerrors.ErrCampaignNotFound
	}
	return item, nil
}

func (s *Store) ListCampaigns(_ context.Context, filter ports.CampaignFilter) ([]entities.Campaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		if strings.TrimSpace(filter.BrandID) != "" && campaign.BrandID != strings.TrimSpace(filter.BrandID) {
			continue
		}
		if filter.Status != "" && campaign.Status != filter.Status {
			continue
		}
		items = append(items, campaign)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) AddMedia(_ context.Context, media entities.Media) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.media[media.MediaID]; exists {
		return domainerrors.ErrMediaAlreadyConfirmed
	}
	if _, exists := s.campaigns[media.CampaignID]; !exists {
		return domainerrors.ErrCampaignNotFound
	}
	s.media[media.MediaID] = media
	return nil
}

func (s *Store) GetMedia(_ context.Context, mediaID string) (entities.Media, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.media[strings.TrimSpace(mediaID)]
	if !exists {
		return entities.Media{}, domainerrors.ErrMediaNotFound
	}
	return item, nil
}

func (s *Store) UpdateMedia(_ context.Context, media entities.Media) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.media[media.MediaID]; !exists {
		return domainerrors.ErrMediaNotFound
	}
	s.media[media.MediaID] = media
	return nil
}

func (s *Store) ListMediaByCampaign(_ context.Context, campaignID string) ([]entities.Media, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Media, 0)
	for _, item := range s.media {
		if item.CampaignID == strings.TrimSpace(campaignID) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) AppendState(_ context.Context, item entities.StateHistory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateLog = append(s.stateLog, item)
	return nil
}

func (s *Store) AppendBudget(_ context.Context, item entities.BudgetLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.budgetLog = append(s.budgetLog, item)
	return nil
}

func (s *Store) GetRecord(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
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

func (s *Store) PutRecord(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.idempotency[record.Key]
	if exists {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyKeyConflict
		}
		if !bytes.Equal(existing.ResponsePayload, record.ResponsePayload) {
			return domainerrors.ErrIdempotencyKeyConflict
		}
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
