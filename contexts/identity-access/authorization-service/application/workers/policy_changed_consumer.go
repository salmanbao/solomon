package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/ports"
)

// PolicyChangedConsumer invalidates permission cache on policy change events.
// It deduplicates events by event_id before applying side effects.
type PolicyChangedConsumer struct {
	Dedup           ports.EventDedupStore
	PermissionCache ports.PermissionCache
	Clock           ports.Clock
	DedupTTL        time.Duration
	Logger          *slog.Logger
}

type policyChangedPayload struct {
	UserID string `json:"user_id"`
}

// Handle applies one consumed policy change event.
func (c PolicyChangedConsumer) Handle(ctx context.Context, event ports.PolicyChangedEvent) error {
	logger := application.ResolveLogger(c.Logger)
	logger.Debug("policy changed event received",
		"event", "authz_policy_changed_received",
		"module", "identity-access/authorization-service",
		"layer", "worker",
		"event_id", event.EventID,
		"event_type", event.EventType,
	)

	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}

	alreadyProcessed, err := c.Dedup.ReserveEvent(
		ctx,
		event.EventID,
		hashPayload(event.Data),
		now.Add(c.dedupTTL()),
	)
	if err != nil {
		logger.Error("policy changed dedupe failed",
			"event", "authz_policy_changed_dedupe_failed",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"error", err.Error(),
		)
		return err
	}
	if alreadyProcessed {
		logger.Debug("policy changed already processed",
			"event", "authz_policy_changed_replayed",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
		)
		return nil
	}

	var payload policyChangedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("policy changed decode failed",
			"event", "authz_policy_changed_decode_failed",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"error", err.Error(),
		)
		return err
	}
	if payload.UserID == "" {
		logger.Warn("policy changed payload missing user",
			"event", "authz_policy_changed_missing_user",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
		)
		return nil
	}
	if err := c.PermissionCache.Invalidate(ctx, payload.UserID); err != nil {
		logger.Error("policy changed cache invalidate failed",
			"event", "authz_policy_changed_cache_invalidate_failed",
			"module", "identity-access/authorization-service",
			"layer", "worker",
			"event_id", event.EventID,
			"user_id", payload.UserID,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("policy changed processed",
		"event", "authz_policy_changed_processed",
		"module", "identity-access/authorization-service",
		"layer", "worker",
		"event_id", event.EventID,
		"event_type", event.EventType,
		"user_id", payload.UserID,
	)
	return nil
}

func (c PolicyChangedConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
