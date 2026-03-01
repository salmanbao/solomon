package application

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/community-experience/community-health-service/ports"
)

type testRepo struct {
	lastInput ports.WebhookIngestInput
}

func (r *testRepo) IngestWebhook(ctx context.Context, input ports.WebhookIngestInput, now time.Time) (ports.IngestionResult, error) {
	r.lastInput = input
	return ports.IngestionResult{
		MessageID:   input.MessageID,
		EventType:   input.EventType,
		ProcessedAt: now.UTC(),
	}, nil
}

func (r *testRepo) GetCommunityHealthScore(ctx context.Context, serverID string) (ports.CommunityHealthScore, error) {
	return ports.CommunityHealthScore{}, nil
}

func (r *testRepo) GetUserRiskScore(ctx context.Context, serverID string, userID string) (ports.UserRiskScore, error) {
	return ports.UserRiskScore{}, nil
}

type testIdempotency struct {
	store map[string]ports.IdempotencyRecord
}

func (t *testIdempotency) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	record, ok := t.store[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (t *testIdempotency) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	t.store[record.Key] = record
	return nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func TestIngestWebhookAllowsEditedEventWithoutUserID(t *testing.T) {
	repo := &testRepo{}
	clock := fixedClock{now: time.Date(2026, time.February, 5, 12, 0, 0, 0, time.UTC)}
	service := Service{
		Repo:        repo,
		Idempotency: &testIdempotency{store: make(map[string]ports.IdempotencyRecord)},
		Clock:       clock,
	}

	editedAt := clock.now
	_, err := service.IngestWebhook(context.Background(), "idem-1", ports.WebhookIngestInput{
		EventType:  "message.edited",
		MessageID:  "msg-1",
		ServerID:   "server-1",
		EditedAt:   &editedAt,
		NewContent: "updated text",
	})
	if err != nil {
		t.Fatalf("expected edited event without user_id to pass validation, got %v", err)
	}
	if repo.lastInput.EventType != "chat.message.edited" {
		t.Fatalf("expected normalized event type, got %q", repo.lastInput.EventType)
	}
}
