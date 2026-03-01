package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	"solomon/contexts/community-experience/storefront-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) CreateStorefront(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	input ports.CreateStorefrontInput,
) (ports.Storefront, error) {
	var out ports.Storefront
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(input.DisplayName) == "" || strings.TrimSpace(input.Category) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m92_create_storefront", actorUserID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.CreateStorefront(ctx, actorUserID, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) UpdateStorefront(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
	input ports.UpdateStorefrontInput,
) (ports.Storefront, error) {
	var out ports.Storefront
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(storefrontID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m92_update_storefront", actorUserID, storefrontID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.UpdateStorefront(ctx, actorUserID, storefrontID, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) GetStorefrontByID(ctx context.Context, storefrontID string, actorUserID string) (ports.Storefront, error) {
	if strings.TrimSpace(storefrontID) == "" {
		return ports.Storefront{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetStorefrontByID(ctx, strings.TrimSpace(storefrontID), strings.TrimSpace(actorUserID))
}

func (s Service) GetStorefrontBySlug(ctx context.Context, slug string) (ports.Storefront, error) {
	if strings.TrimSpace(slug) == "" {
		return ports.Storefront{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetStorefrontBySlug(ctx, strings.TrimSpace(slug))
}

func (s Service) PublishStorefront(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
) (ports.Storefront, error) {
	var out ports.Storefront
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(storefrontID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m92_publish_storefront", actorUserID, storefrontID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.PublishStorefront(ctx, actorUserID, storefrontID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) ReportStorefront(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
	input ports.ReportInput,
) (ports.ReportResult, error) {
	var out ports.ReportResult
	if strings.TrimSpace(storefrontID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m92_report_storefront", actorUserID, storefrontID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.ReportStorefront(ctx, actorUserID, storefrontID, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) ConsumeProductPublishedEvent(
	ctx context.Context,
	event ports.ProductPublishedEvent,
) (ports.ProductProjectionResult, error) {
	if strings.TrimSpace(event.EventID) == "" || strings.TrimSpace(event.StorefrontID) == "" || strings.TrimSpace(event.ProductID) == "" {
		return ports.ProductProjectionResult{}, domainerrors.ErrInvalidRequest
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.now()
	}
	return s.Repo.ConsumeProductPublishedEvent(ctx, event, s.now())
}

func (s Service) UpsertSubscriptionProjection(
	ctx context.Context,
	input ports.SubscriptionProjectionInput,
) error {
	if strings.TrimSpace(input.UserID) == "" {
		return domainerrors.ErrInvalidRequest
	}
	return s.Repo.UpsertSubscriptionProjection(ctx, input, s.now())
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

	resolveLogger(s.Logger).Debug("storefront idempotent operation committed",
		"event", "storefront_idempotent_operation_committed",
		"module", "community-experience/storefront-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
