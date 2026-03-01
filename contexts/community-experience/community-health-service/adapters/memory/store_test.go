package memory

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/community-experience/community-health-service/ports"
)

func TestIngestWebhookDeduplicatesByEventID(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, time.February, 5, 12, 0, 0, 0, time.UTC)

	input := ports.WebhookIngestInput{
		EventID:   "evt-1",
		EventType: "chat.message.created",
		MessageID: "msg-1",
		ServerID:  "server-a",
		ChannelID: "channel-a",
		UserID:    "user-a",
		Content:   "great work team",
	}

	if _, err := store.IngestWebhook(context.Background(), input, now); err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	if _, err := store.IngestWebhook(context.Background(), input, now.Add(time.Second)); err != nil {
		t.Fatalf("duplicate ingest failed: %v", err)
	}

	score, err := store.GetCommunityHealthScore(context.Background(), "server-a")
	if err != nil {
		t.Fatalf("score lookup failed: %v", err)
	}
	if score.TotalMessages != 1 {
		t.Fatalf("expected total_messages=1 after duplicate event, got %d", score.TotalMessages)
	}
}

func TestIngestWebhookEditDoesNotIncrementAndDeleteUsesStoredOwner(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, time.February, 5, 13, 0, 0, 0, time.UTC)

	createdAt := now
	editedAt := now.Add(2 * time.Minute)
	deletedAt := now.Add(4 * time.Minute)

	created := ports.WebhookIngestInput{
		EventID:   "evt-2",
		EventType: "chat.message.created",
		MessageID: "msg-2",
		ServerID:  "server-b",
		ChannelID: "channel-b",
		UserID:    "user-b",
		Content:   "hello",
		CreatedAt: &createdAt,
	}
	if _, err := store.IngestWebhook(context.Background(), created, now); err != nil {
		t.Fatalf("create ingest failed: %v", err)
	}

	edited := ports.WebhookIngestInput{
		EventID:    "evt-3",
		EventType:  "chat.message.edited",
		MessageID:  "msg-2",
		ServerID:   "server-b",
		NewContent: "hello updated",
		EditedAt:   &editedAt,
	}
	if _, err := store.IngestWebhook(context.Background(), edited, now.Add(time.Minute)); err != nil {
		t.Fatalf("edit ingest failed: %v", err)
	}

	scoreAfterEdit, err := store.GetCommunityHealthScore(context.Background(), "server-b")
	if err != nil {
		t.Fatalf("score lookup after edit failed: %v", err)
	}
	if scoreAfterEdit.TotalMessages != 1 {
		t.Fatalf("expected total_messages=1 after edit, got %d", scoreAfterEdit.TotalMessages)
	}

	deleted := ports.WebhookIngestInput{
		EventID:   "evt-4",
		EventType: "chat.message.deleted",
		MessageID: "msg-2",
		ServerID:  "server-b",
		DeletedAt: &deletedAt,
	}
	if _, err := store.IngestWebhook(context.Background(), deleted, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("delete ingest failed: %v", err)
	}

	scoreAfterDelete, err := store.GetCommunityHealthScore(context.Background(), "server-b")
	if err != nil {
		t.Fatalf("score lookup after delete failed: %v", err)
	}
	if scoreAfterDelete.TotalMessages != 0 {
		t.Fatalf("expected total_messages=0 after delete, got %d", scoreAfterDelete.TotalMessages)
	}
}
