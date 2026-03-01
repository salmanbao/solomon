package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/community-health-service/domain/errors"
	"solomon/contexts/community-experience/community-health-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) IngestWebhook(
	ctx context.Context,
	idempotencyKey string,
	input ports.WebhookIngestInput,
) (ports.IngestionResult, error) {
	var out ports.IngestionResult
	input.EventType = normalizeWebhookEventType(input.EventType)
	if strings.TrimSpace(input.MessageID) == "" ||
		strings.TrimSpace(input.ServerID) == "" ||
		input.EventType == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if input.EventType == "chat.message.created" && strings.TrimSpace(input.UserID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("community_health_ingest_webhook", string(payload))
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.IngestWebhook(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) GetCommunityHealthScore(ctx context.Context, serverID string) (ports.CommunityHealthScore, error) {
	if strings.TrimSpace(serverID) == "" {
		return ports.CommunityHealthScore{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetCommunityHealthScore(ctx, serverID)
}

func (s Service) GetUserRiskScore(ctx context.Context, serverID string, userID string) (ports.UserRiskScore, error) {
	if strings.TrimSpace(serverID) == "" || strings.TrimSpace(userID) == "" {
		return ports.UserRiskScore{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetUserRiskScore(ctx, serverID, userID)
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

	resolveLogger(s.Logger).Debug("community health idempotent operation committed",
		"event", "community_health_idempotent_operation_committed",
		"module", "community-experience/community-health-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}

func normalizeWebhookEventType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "chat.message.created", "message.created", "created":
		return "chat.message.created"
	case "chat.message.edited", "message.edited", "edited":
		return "chat.message.edited"
	case "chat.message.deleted", "message.deleted", "deleted":
		return "chat.message.deleted"
	default:
		return ""
	}
}
