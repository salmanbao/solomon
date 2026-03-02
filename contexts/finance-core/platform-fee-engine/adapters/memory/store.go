package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	domainerrors "solomon/contexts/finance-core/platform-fee-engine/domain/errors"
	"solomon/contexts/finance-core/platform-fee-engine/ports"

	"github.com/google/uuid"
)

type Store struct {
	mu sync.RWMutex

	calculations map[string]ports.FeeCalculation
	idempotency  map[string]ports.IdempotencyRecord
	eventDedup   map[string]dedupRecord
	outbox       map[string]outboxRecord
}

type dedupRecord struct {
	PayloadHash string
	ExpiresAt   time.Time
}

type outboxRecord struct {
	Message     ports.OutboxMessage
	Status      string
	PublishedAt *time.Time
}

const (
	outboxStatusPending   = "pending"
	outboxStatusPublished = "published"
)

func NewStore() *Store {
	return &Store{
		calculations: make(map[string]ports.FeeCalculation),
		idempotency:  make(map[string]ports.IdempotencyRecord),
		eventDedup:   make(map[string]dedupRecord),
		outbox:       make(map[string]outboxRecord),
	}
}

func (s *Store) CreateCalculation(_ context.Context, calculation ports.FeeCalculation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(calculation.CalculationID)
	if id == "" {
		return domainerrors.ErrInvalidInput
	}
	if _, exists := s.calculations[id]; exists {
		return domainerrors.ErrIdempotencyConflict
	}
	s.calculations[id] = calculation
	return nil
}

func (s *Store) GetCalculation(_ context.Context, calculationID string) (ports.FeeCalculation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.calculations[strings.TrimSpace(calculationID)]
	if !ok {
		return ports.FeeCalculation{}, domainerrors.ErrNotFound
	}
	return item, nil
}

func (s *Store) ListCalculationsByUser(_ context.Context, userID string, limit int, offset int) ([]ports.FeeCalculation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	items := make([]ports.FeeCalculation, 0)
	for _, item := range s.calculations {
		if item.UserID == strings.TrimSpace(userID) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CalculatedAt.After(items[j].CalculatedAt)
	})
	if offset >= len(items) {
		return []ports.FeeCalculation{}, nil
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return append([]ports.FeeCalculation(nil), items[offset:end]...), nil
}

func (s *Store) BuildMonthlyReport(_ context.Context, month string) (ports.FeeReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := ports.FeeReport{Month: strings.TrimSpace(month)}
	for _, item := range s.calculations {
		if item.CalculatedAt.UTC().Format("2006-01") != report.Month {
			continue
		}
		report.Count++
		report.TotalGross += item.GrossAmount
		report.TotalFee += item.FeeAmount
		report.TotalNet += item.NetAmount
	}
	return report, nil
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

func (s *Store) ReserveEvent(_ context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(eventID)
	if key == "" {
		return false, domainerrors.ErrInvalidInput
	}
	if existing, ok := s.eventDedup[key]; ok {
		if existing.PayloadHash != payloadHash {
			return false, domainerrors.ErrIdempotencyConflict
		}
		return true, nil
	}
	s.eventDedup[key] = dedupRecord{
		PayloadHash: payloadHash,
		ExpiresAt:   expiresAt.UTC(),
	}
	return false, nil
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
		return domainerrors.ErrInvalidInput
	}

	if existing, ok := s.outbox[outboxID]; ok {
		if !bytes.Equal(existing.Message.Payload, payload) {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}

	s.outbox[outboxID] = outboxRecord{
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
		return domainerrors.ErrNotFound
	}
	ts := publishedAt.UTC()
	row.Status = outboxStatusPublished
	row.PublishedAt = &ts
	s.outbox[strings.TrimSpace(outboxID)] = row
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
