package workers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"solomon/contexts/identity-access/authorization-service/ports"
)

type PolicyChangedConsumer struct {
	Dedup           ports.EventDedupStore
	PermissionCache ports.PermissionCache
	Clock           ports.Clock
	DedupTTL        time.Duration
}

type policyChangedPayload struct {
	UserID string `json:"user_id"`
}

func (c PolicyChangedConsumer) Handle(ctx context.Context, event ports.PolicyChangedEvent) error {
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
	if err != nil || alreadyProcessed {
		return err
	}

	var payload policyChangedPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}
	if payload.UserID == "" {
		return nil
	}
	return c.PermissionCache.Invalidate(ctx, payload.UserID)
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
