package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/subscription-service/domain/errors"
	"solomon/contexts/community-experience/subscription-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) CreateSubscription(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	input ports.CreateSubscriptionInput,
) (ports.Subscription, error) {
	var out ports.Subscription
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(input.PlanID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m61_create_subscription", userID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CreateSubscription(ctx, userID, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) ChangePlan(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	subscriptionID string,
	newPlanID string,
) (ports.PlanChangeResult, error) {
	var out ports.PlanChangeResult
	if strings.TrimSpace(userID) == "" ||
		strings.TrimSpace(subscriptionID) == "" ||
		strings.TrimSpace(newPlanID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m61_change_plan", userID, subscriptionID, newPlanID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.ChangePlan(ctx, userID, subscriptionID, newPlanID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) CancelSubscription(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	subscriptionID string,
	cancelAtPeriodEnd bool,
	cancellationFeedback string,
) (ports.CancelSubscriptionResult, error) {
	var out ports.CancelSubscriptionResult
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(subscriptionID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if len(cancellationFeedback) > 1000 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings(
		"m61_cancel_subscription",
		userID,
		subscriptionID,
		strings.ToLower(strings.TrimSpace(cancellationFeedback)),
		map[bool]string{true: "true", false: "false"}[cancelAtPeriodEnd],
	)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CancelSubscription(ctx, userID, subscriptionID, cancelAtPeriodEnd, cancellationFeedback, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
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

	resolveLogger(s.Logger).Debug("subscription idempotent operation committed",
		"event", "subscription_idempotent_operation_committed",
		"module", "community-experience/subscription-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
