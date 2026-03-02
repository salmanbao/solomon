package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IdempotencyTTL time.Duration
}

type RecordActionInput struct {
	ActorID       string
	Action        string
	TargetID      string
	Justification string
	SourceIP      string
	CorrelationID string
}

func (s Service) RecordAdminAction(ctx context.Context, idempotencyKey string, input RecordActionInput) (ports.AuditLog, error) {
	if strings.TrimSpace(input.ActorID) == "" {
		return ports.AuditLog{}, domainerrors.ErrUnauthorized
	}
	if strings.TrimSpace(input.Action) == "" || strings.TrimSpace(input.Justification) == "" {
		return ports.AuditLog{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ports.AuditLog{}, domainerrors.ErrIdempotencyRequired
	}

	now := s.Clock.Now().UTC()
	if s.IdempotencyTTL <= 0 {
		s.IdempotencyTTL = 7 * 24 * time.Hour
	}
	requestHash := hashPayload(input)

	existing, err := s.Idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return ports.AuditLog{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return ports.AuditLog{}, domainerrors.ErrIdempotencyConflict
		}
		var cached ports.AuditLog
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return ports.AuditLog{}, err
		}
		return cached, nil
	}
	if err := s.Idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.IdempotencyTTL)); err != nil {
		return ports.AuditLog{}, err
	}

	logRow := ports.AuditLog{
		AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
		ActorID:       strings.TrimSpace(input.ActorID),
		Action:        strings.TrimSpace(input.Action),
		TargetID:      strings.TrimSpace(input.TargetID),
		Justification: strings.TrimSpace(input.Justification),
		OccurredAt:    now,
		SourceIP:      strings.TrimSpace(input.SourceIP),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
	}
	if err := s.Repo.AppendAuditLog(ctx, logRow); err != nil {
		return ports.AuditLog{}, err
	}
	body, err := json.Marshal(logRow)
	if err != nil {
		return ports.AuditLog{}, err
	}
	if err := s.Idempotency.Complete(ctx, idempotencyKey, body, now); err != nil {
		return ports.AuditLog{}, err
	}
	return logRow, nil
}

func (s Service) ListRecentActions(ctx context.Context, limit int) ([]ports.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.Repo.ListRecentAuditLogs(ctx, limit)
}

func hashPayload(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
