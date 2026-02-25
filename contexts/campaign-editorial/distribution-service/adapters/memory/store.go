package memory

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"

	"github.com/google/uuid"
)

type outboxRecord struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
	PublishedAt  *time.Time
}

type Store struct {
	mu sync.RWMutex

	items            map[string]entities.DistributionItem
	clipCampaign     map[string]string
	captions         map[string]entities.Caption
	overlays         map[string]entities.Overlay
	platformStatus   map[string]entities.PlatformStatus
	publishingMetric map[string]entities.PublishingAnalytics
	outbox           map[string]outboxRecord
}

func NewStore(seed []entities.DistributionItem) *Store {
	items := make(map[string]entities.DistributionItem, len(seed))
	clipCampaign := make(map[string]string, len(seed))
	for _, item := range seed {
		items[item.ID] = item
		if strings.TrimSpace(item.ClipID) != "" && strings.TrimSpace(item.CampaignID) != "" {
			clipCampaign[item.ClipID] = item.CampaignID
		}
	}
	return &Store{
		items:            items,
		clipCampaign:     clipCampaign,
		captions:         make(map[string]entities.Caption),
		overlays:         make(map[string]entities.Overlay),
		platformStatus:   make(map[string]entities.PlatformStatus),
		publishingMetric: make(map[string]entities.PublishingAnalytics),
		outbox:           make(map[string]outboxRecord),
	}
}

func (s *Store) CreateItem(_ context.Context, item entities.DistributionItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[item.ID]; exists {
		return domainerrors.ErrDistributionItemExists
	}
	for _, existing := range s.items {
		if existing.InfluencerID == item.InfluencerID &&
			existing.ClipID == item.ClipID &&
			existing.CampaignID == item.CampaignID {
			return domainerrors.ErrDistributionItemExists
		}
	}
	s.items[item.ID] = item
	if strings.TrimSpace(item.ClipID) != "" && strings.TrimSpace(item.CampaignID) != "" {
		s.clipCampaign[item.ClipID] = item.CampaignID
	}
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

func (s *Store) ListDueScheduled(
	_ context.Context,
	threshold time.Time,
	limit int,
) ([]entities.DistributionItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	items := make([]entities.DistributionItem, 0, limit)
	for _, item := range s.items {
		if item.Status != entities.DistributionStatusScheduled || item.ScheduledForUTC == nil {
			continue
		}
		if item.ScheduledForUTC.UTC().After(threshold.UTC()) {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ScheduledForUTC.Before(*items[j].ScheduledForUTC)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) GetCampaignIDByClip(_ context.Context, clipID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaignID, ok := s.clipCampaign[strings.TrimSpace(clipID)]
	if !ok {
		return "", domainerrors.ErrDistributionItemNotFound
	}
	return campaignID, nil
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

func (s *Store) UpsertCaption(_ context.Context, caption entities.Caption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[caption.DistributionItemID]; !exists {
		return domainerrors.ErrDistributionItemNotFound
	}
	key := caption.DistributionItemID + "|" + strings.TrimSpace(caption.Platform)
	s.captions[key] = caption
	return nil
}

func (s *Store) UpsertPlatformStatus(_ context.Context, status entities.PlatformStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[status.DistributionItemID]; !exists {
		return domainerrors.ErrDistributionItemNotFound
	}
	key := status.DistributionItemID + "|" + status.Platform
	s.platformStatus[key] = status
	return nil
}

func (s *Store) AddPublishingAnalytics(_ context.Context, analytics entities.PublishingAnalytics) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[analytics.DistributionItemID]; !exists {
		return domainerrors.ErrDistributionItemNotFound
	}
	s.publishingMetric[analytics.ID] = analytics
	return nil
}

func (s *Store) AppendOutbox(_ context.Context, envelope ports.EventEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	outboxID := strings.TrimSpace(envelope.EventID)
	if outboxID == "" {
		outboxID = uuid.NewString()
	}
	if _, exists := s.outbox[outboxID]; exists {
		return nil
	}
	s.outbox[outboxID] = outboxRecord{
		OutboxID:     outboxID,
		EventType:    strings.TrimSpace(envelope.EventType),
		PartitionKey: strings.TrimSpace(envelope.PartitionKey),
		Payload:      payload,
		CreatedAt:    envelope.OccurredAt.UTC(),
	}
	return nil
}

func (s *Store) ListPendingOutbox(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	rows := make([]outboxRecord, 0, len(s.outbox))
	for _, row := range s.outbox {
		if row.PublishedAt == nil {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	items := make([]ports.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, ports.OutboxMessage{
			OutboxID:     row.OutboxID,
			EventType:    row.EventType,
			PartitionKey: row.PartitionKey,
			Payload:      append([]byte(nil), row.Payload...),
			CreatedAt:    row.CreatedAt.UTC(),
		})
	}
	return items, nil
}

func (s *Store) MarkOutboxPublished(_ context.Context, outboxID string, publishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.outbox[strings.TrimSpace(outboxID)]
	if !ok {
		return domainerrors.ErrInvalidDistributionInput
	}
	timestamp := publishedAt.UTC()
	row.PublishedAt = &timestamp
	s.outbox[outboxID] = row
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
var _ ports.OutboxWriter = (*Store)(nil)
var _ ports.OutboxRepository = (*Store)(nil)
