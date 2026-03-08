package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainerrors "solomon/contexts/moderation-safety/abuse-prevention-service/domain/errors"
	"solomon/contexts/moderation-safety/abuse-prevention-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IdempotencyTTL time.Duration
}

type ReleaseLockoutInput struct {
	ActorID       string
	UserID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type ReleaseLockoutResult struct {
	ThreatID        string
	UserID          string
	Status          string
	ReleasedAt      time.Time
	OwnerAuditLogID string
}

func (s Service) ReleaseLockout(
	ctx context.Context,
	idempotencyKey string,
	input ReleaseLockoutInput,
) (ReleaseLockoutResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return ReleaseLockoutResult{}, domainerrors.ErrUnauthorized
	}
	if strings.HasPrefix(strings.ToLower(input.ActorID), "viewer-") {
		return ReleaseLockoutResult{}, domainerrors.ErrForbidden
	}
	if input.UserID == "" || input.Reason == "" {
		return ReleaseLockoutResult{}, domainerrors.ErrInvalidRequest
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ReleaseLockoutResult{}, domainerrors.ErrIdempotencyKeyRequired
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ReleaseLockoutResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			release, err := s.Repo.ReleaseLockout(ctx, input.UserID, now)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("abuse_audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "abuse.lockout.released",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ReleaseLockoutResult{
				ThreatID:        release.ThreatID,
				UserID:          release.UserID,
				Status:          release.Status,
				ReleasedAt:      release.ReleasedAt,
				OwnerAuditLogID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ReleaseLockoutResult{}, err
	}
	return output, nil
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	idempotencyKey string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	existing, err := s.Idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(existing.ResponseBody)
	}

	if err := s.Idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.idempotencyTTL())); err != nil {
		return err
	}
	body, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Complete(ctx, idempotencyKey, body, now); err != nil {
		return err
	}
	return decode(body)
}

func hashPayload(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
