package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	"solomon/contexts/identity-access/onboarding-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) ConsumeUserRegisteredEvent(
	ctx context.Context,
	event ports.UserRegisteredEvent,
) (ports.FlowState, error) {
	if strings.TrimSpace(event.EventID) == "" ||
		strings.TrimSpace(event.UserID) == "" ||
		strings.TrimSpace(event.Role) == "" {
		return ports.FlowState{}, domainerrors.ErrSchemaInvalid
	}
	if !ports.IsValidRole(strings.ToLower(strings.TrimSpace(event.Role))) {
		return ports.FlowState{}, domainerrors.ErrUnknownRole
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.now()
	}
	event.Role = strings.ToLower(strings.TrimSpace(event.Role))
	return s.Repo.ConsumeUserRegisteredEvent(ctx, event, s.now())
}

func (s Service) GetFlow(ctx context.Context, userID string) (ports.FlowState, error) {
	if strings.TrimSpace(userID) == "" {
		return ports.FlowState{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetFlow(ctx, strings.TrimSpace(userID))
}

func (s Service) CompleteStep(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	stepKey string,
	metadata map[string]any,
) (ports.StepCompletion, error) {
	var out ports.StepCompletion
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(stepKey) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(metadata)
	requestHash := hashStrings("m22_complete_step", userID, stepKey, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.CompleteStep(ctx, userID, stepKey, metadata, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) SkipFlow(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	reason string,
) (ports.SkipResult, error) {
	var out ports.SkipResult
	if strings.TrimSpace(userID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m22_skip_flow", userID, strings.TrimSpace(reason))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.SkipFlow(ctx, userID, reason, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) ResumeFlow(
	ctx context.Context,
	idempotencyKey string,
	userID string,
) (ports.ResumeResult, error) {
	var out ports.ResumeResult
	if strings.TrimSpace(userID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m22_resume_flow", userID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.ResumeFlow(ctx, userID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) ListAdminFlows(ctx context.Context) ([]ports.AdminFlow, error) {
	return s.Repo.ListAdminFlows(ctx)
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) requireIdempotency(key string) error {
	if strings.TrimSpace(key) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	return nil
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("onboarding idempotent operation committed",
		"event", "onboarding_idempotent_operation_committed",
		"module", "identity-access/onboarding-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
