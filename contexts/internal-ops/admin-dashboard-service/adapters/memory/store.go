package memory

import (
	"context"
	"slices"
	"sync"
	"time"

	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Store struct {
	mu          sync.Mutex
	logs        []ports.AuditLog
	idempotency map[string]ports.IdempotencyRecord
}

func NewStore() *Store {
	return &Store{
		logs:        make([]ports.AuditLog, 0, 128),
		idempotency: map[string]ports.IdempotencyRecord{},
	}
}

func (s *Store) AppendAuditLog(_ context.Context, row ports.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, row)
	return nil
}

func (s *Store) ListRecentAuditLogs(_ context.Context, limit int) ([]ports.AuditLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]ports.AuditLog, 0, limit)
	for i := len(s.logs) - 1; i >= 0; i-- {
		out = append(out, s.logs[i])
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.idempotency[key]
	if !ok {
		return nil, nil
	}
	if now.After(row.ExpiresAt) {
		delete(s.idempotency, key)
		return nil, nil
	}
	clone := row
	clone.ResponseBody = slices.Clone(row.ResponseBody)
	return &clone, nil
}

func (s *Store) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if row, ok := s.idempotency[key]; ok && time.Now().UTC().Before(row.ExpiresAt) {
		if row.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   expiresAt,
	}
	return nil
}

func (s *Store) Complete(_ context.Context, key string, responseBody []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.idempotency[key]
	if !ok {
		return nil
	}
	row.ResponseBody = slices.Clone(responseBody)
	if at.After(row.ExpiresAt) {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	s.idempotency[key] = row
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}
