package memory

import (
	"bytes"
	"context"
	"encoding/json"
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
	outbox      map[string]memoryOutboxRecord
	eventDedup  map[string]memoryDedupRecord
}

type memoryOutboxRecord struct {
	Message     ports.OutboxMessage
	Status      string
	PublishedAt *time.Time
}

type memoryDedupRecord struct {
	PayloadHash string
	ExpiresAt   time.Time
}

const (
	outboxStatusPending   = "pending"
	outboxStatusPublished = "published"
)

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
		outbox:      make(map[string]memoryOutboxRecord),
		eventDedup:  make(map[string]memoryDedupRecord),
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

func (s *Store) AppendOutbox(_ context.Context, envelope ports.EventEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	message := ports.OutboxMessage{
		OutboxID:     envelope.EventID,
		EventType:    envelope.EventType,
		PartitionKey: envelope.PartitionKey,
		Payload:      payload,
		CreatedAt:    envelope.OccurredAt.UTC(),
	}
	if existing, ok := s.outbox[message.OutboxID]; ok {
		if !bytes.Equal(existing.Message.Payload, message.Payload) {
			return domainerrors.ErrIdempotencyKeyConflict
		}
		return nil
	}
	s.outbox[message.OutboxID] = memoryOutboxRecord{
		Message: message,
		Status:  outboxStatusPending,
	}
	return nil
}

func (s *Store) ListPendingOutbox(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	items := make([]ports.OutboxMessage, 0)
	for _, row := range s.outbox {
		if row.Status == outboxStatusPending {
			items = append(items, row.Message)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) MarkOutboxPublished(_ context.Context, outboxID string, publishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.outbox[strings.TrimSpace(outboxID)]
	if !ok {
		return domainerrors.ErrInvalidCampaignInput
	}
	timestamp := publishedAt.UTC()
	row.Status = outboxStatusPublished
	row.PublishedAt = &timestamp
	s.outbox[strings.TrimSpace(outboxID)] = row
	return nil
}

func (s *Store) ReserveEvent(
	_ context.Context,
	eventID string,
	payloadHash string,
	expiresAt time.Time,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(eventID)
	existing, ok := s.eventDedup[key]
	if ok {
		if existing.PayloadHash != payloadHash {
			return false, domainerrors.ErrIdempotencyKeyConflict
		}
		return true, nil
	}
	s.eventDedup[key] = memoryDedupRecord{
		PayloadHash: payloadHash,
		ExpiresAt:   expiresAt.UTC(),
	}
	return false, nil
}

func (s *Store) ApplySubmissionCreated(
	_ context.Context,
	campaignID string,
	eventID string,
	occurredAt time.Time,
) (ports.SubmissionCreatedResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	campaign, ok := s.campaigns[strings.TrimSpace(campaignID)]
	if !ok {
		return ports.SubmissionCreatedResult{}, domainerrors.ErrCampaignNotFound
	}
	if campaign.Status != entities.CampaignStatusActive {
		return ports.SubmissionCreatedResult{
			CampaignID:      campaign.CampaignID,
			BudgetRemaining: campaign.BudgetRemaining,
			NewStatus:       campaign.Status,
		}, nil
	}
	reserveAmount := campaign.RatePer1KViews
	campaign.SubmissionCount++
	campaign.BudgetReserved += reserveAmount
	campaign.BudgetRemaining = campaign.BudgetTotal - campaign.BudgetSpent - campaign.BudgetReserved
	campaign.UpdatedAt = occurredAt.UTC()

	autoPaused := false
	if campaign.BudgetRemaining < entities.BudgetAutoPauseThreshold(campaign.RatePer1KViews) {
		autoPaused = true
		from := campaign.Status
		campaign.Status = entities.CampaignStatusPaused
		s.stateLog = append(s.stateLog, entities.StateHistory{
			HistoryID:    uuid.NewString(),
			CampaignID:   campaign.CampaignID,
			FromState:    from,
			ToState:      entities.CampaignStatusPaused,
			ChangedBy:    "system",
			ChangeReason: "budget_exhausted",
			CreatedAt:    occurredAt.UTC(),
		})
		if envelope, err := campaignEnvelopeFromMap(
			eventID+"-paused",
			"campaign.paused",
			campaign.CampaignID,
			occurredAt.UTC(),
			map[string]any{
				"campaign_id":      campaign.CampaignID,
				"reason":           "budget_exhausted",
				"budget_remaining": campaign.BudgetRemaining,
			},
		); err == nil {
			_ = s.appendOutboxEnvelope(envelope)
		}
	}

	s.budgetLog = append(s.budgetLog, entities.BudgetLog{
		LogID:       uuid.NewString(),
		CampaignID:  campaign.CampaignID,
		AmountDelta: reserveAmount,
		Reason:      "submission_created_reserve",
		CreatedAt:   occurredAt.UTC(),
	})
	s.campaigns[campaign.CampaignID] = campaign

	if envelope, err := campaignEnvelopeFromMap(
		eventID+"-budget",
		"campaign.budget_updated",
		campaign.CampaignID,
		occurredAt.UTC(),
		map[string]any{
			"campaign_id":      campaign.CampaignID,
			"budget_total":     campaign.BudgetTotal,
			"budget_spent":     campaign.BudgetSpent,
			"budget_reserved":  campaign.BudgetReserved,
			"budget_remaining": campaign.BudgetRemaining,
		},
	); err == nil {
		_ = s.appendOutboxEnvelope(envelope)
	}

	return ports.SubmissionCreatedResult{
		CampaignID:          campaign.CampaignID,
		BudgetReservedDelta: reserveAmount,
		BudgetRemaining:     campaign.BudgetRemaining,
		AutoPaused:          autoPaused,
		NewStatus:           campaign.Status,
	}, nil
}

func (s *Store) CompleteCampaignsPastDeadline(
	_ context.Context,
	now time.Time,
	limit int,
) ([]ports.DeadlineCompletionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 100
	}
	results := make([]ports.DeadlineCompletionResult, 0)
	for _, campaign := range s.campaigns {
		if len(results) >= limit {
			break
		}
		if campaign.Status != entities.CampaignStatusActive {
			continue
		}
		if campaign.DeadlineAt == nil || !campaign.DeadlineAt.UTC().Before(now.UTC()) {
			continue
		}
		from := campaign.Status
		campaign.Status = entities.CampaignStatusCompleted
		completedAt := now.UTC()
		campaign.CompletedAt = &completedAt
		campaign.UpdatedAt = completedAt
		s.campaigns[campaign.CampaignID] = campaign
		s.stateLog = append(s.stateLog, entities.StateHistory{
			HistoryID:    uuid.NewString(),
			CampaignID:   campaign.CampaignID,
			FromState:    from,
			ToState:      entities.CampaignStatusCompleted,
			ChangedBy:    "system",
			ChangeReason: "deadline_reached",
			CreatedAt:    completedAt,
		})
		if envelope, err := campaignEnvelopeFromMap(
			uuid.NewString(),
			"campaign.completed",
			campaign.CampaignID,
			completedAt,
			map[string]any{
				"campaign_id": campaign.CampaignID,
				"brand_id":    campaign.BrandID,
				"status":      string(campaign.Status),
				"reason":      "deadline_reached",
			},
		); err == nil {
			_ = s.appendOutboxEnvelope(envelope)
		}
		results = append(results, ports.DeadlineCompletionResult{CampaignID: campaign.CampaignID})
	}
	return results, nil
}

func (s *Store) appendOutboxEnvelope(envelope ports.EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	outboxID := strings.TrimSpace(envelope.EventID)
	if outboxID == "" {
		outboxID = uuid.NewString()
	}
	if existing, ok := s.outbox[outboxID]; ok {
		if !bytes.Equal(existing.Message.Payload, payload) {
			return domainerrors.ErrIdempotencyKeyConflict
		}
		return nil
	}
	s.outbox[outboxID] = memoryOutboxRecord{
		Message: ports.OutboxMessage{
			OutboxID:     outboxID,
			EventType:    envelope.EventType,
			PartitionKey: envelope.PartitionKey,
			Payload:      payload,
			CreatedAt:    envelope.OccurredAt.UTC(),
		},
		Status: outboxStatusPending,
	}
	return nil
}

func campaignEnvelopeFromMap(
	eventID string,
	eventType string,
	campaignID string,
	occurredAt time.Time,
	data map[string]any,
) (ports.EventEnvelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return ports.EventEnvelope{}, err
	}
	return ports.EventEnvelope{
		EventID:          eventID,
		EventType:        eventType,
		OccurredAt:       occurredAt.UTC(),
		SourceService:    "campaign-service",
		SchemaVersion:    1,
		PartitionKeyPath: "campaign_id",
		PartitionKey:     campaignID,
		Data:             payload,
	}, nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
