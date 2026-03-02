package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"solomon/contexts/campaign-editorial/editor-dashboard-service/adapters/memory"
	domainerrors "solomon/contexts/campaign-editorial/editor-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/ports"
)

func TestSaveCampaignIsIdempotent(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:           store,
		Idempotency:    store,
		EventDedup:     store,
		Clock:          store,
		IdempotencyTTL: 24 * time.Hour,
		EventDedupTTL:  24 * time.Hour,
	}
	first, err := svc.SaveCampaign(context.Background(), "key-1", "editor-1", "camp-1")
	if err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	second, err := svc.SaveCampaign(context.Background(), "key-1", "editor-1", "camp-1")
	if err != nil {
		t.Fatalf("second save failed: %v", err)
	}
	if !first.Saved || !second.Saved {
		t.Fatalf("expected saved=true")
	}
	if first.CampaignID != second.CampaignID {
		t.Fatalf("expected same campaign id")
	}
}

func TestSaveCampaignConflictOnDifferentPayload(t *testing.T) {
	store := memory.NewStore()
	svc := Service{Repo: store, Idempotency: store, EventDedup: store, Clock: store}
	_, err := svc.SaveCampaign(context.Background(), "key-2", "editor-1", "camp-1")
	if err != nil {
		t.Fatalf("seed save failed: %v", err)
	}
	_, err = svc.SaveCampaign(context.Background(), "key-2", "editor-1", "camp-2")
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestApplySubmissionEventDeduplicates(t *testing.T) {
	store := memory.NewStore()
	svc := Service{Repo: store, Idempotency: store, EventDedup: store, Clock: store}
	evt := ports.SubmissionLifecycleEvent{
		EventID:      "evt-1",
		EventType:    "submission.approved",
		SubmissionID: "sub-1",
		UserID:       "editor-1",
		Status:       "approved",
		OccurredAt:   time.Now().UTC(),
	}
	if err := svc.ApplySubmissionLifecycleEvent(context.Background(), evt); err != nil {
		t.Fatalf("first apply failed: %v", err)
	}
	if err := svc.ApplySubmissionLifecycleEvent(context.Background(), evt); err != nil {
		t.Fatalf("second apply should be noop: %v", err)
	}
}
