package memory

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	domainerrors "solomon/contexts/moderation-safety/abuse-prevention-service/domain/errors"
	"solomon/contexts/moderation-safety/abuse-prevention-service/ports"
)

type lockoutRecord struct {
	ThreatID   string
	UserID     string
	Active     bool
	ReleasedAt *time.Time
}

type Store struct {
	mu          sync.Mutex
	lockouts    map[string]lockoutRecord
	idempotency map[string]ports.IdempotencyRecord
	auditLogs   []ports.AuditLog
	sequence    int64
}

func NewStore() *Store {
	return &Store{
		lockouts: map[string]lockoutRecord{
			"locked-user-1": {
				ThreatID: "threat_locked-user-1",
				UserID:   "locked-user-1",
				Active:   true,
			},
			"user-200": {
				ThreatID: "threat_user-200",
				UserID:   "user-200",
				Active:   true,
			},
		},
		idempotency: map[string]ports.IdempotencyRecord{},
		auditLogs:   make([]ports.AuditLog, 0, 64),
	}
}

func (s *Store) ReleaseLockout(_ context.Context, userID string, releasedAt time.Time) (ports.LockoutRelease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.lockouts[userID]
	if !ok || !row.Active {
		return ports.LockoutRelease{}, domainerrors.ErrThreatNotFound
	}
	releasedAt = releasedAt.UTC()
	row.Active = false
	row.ReleasedAt = &releasedAt
	s.lockouts[userID] = row
	return ports.LockoutRelease{
		ThreatID:   row.ThreatID,
		UserID:     row.UserID,
		Status:     "released",
		ReleasedAt: releasedAt,
	}, nil
}

func (s *Store) AppendAuditLog(_ context.Context, row ports.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	if row.AuditID == "" {
		row.AuditID = "abuse_audit_" + strconv.FormatInt(s.sequence, 10)
	}
	s.auditLogs = append(s.auditLogs, row)
	return nil
}

func (s *Store) ListRecentAuditLogs(_ context.Context, limit int) ([]ports.AuditLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]ports.AuditLog, 0, limit)
	for i := len(s.auditLogs) - 1; i >= 0; i-- {
		out = append(out, s.auditLogs[i])
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
