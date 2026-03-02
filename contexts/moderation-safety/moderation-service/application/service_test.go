package application

import (
	"context"
	"errors"
	"testing"

	"solomon/contexts/moderation-safety/moderation-service/adapters/memory"
	domainerrors "solomon/contexts/moderation-safety/moderation-service/domain/errors"
	"solomon/contexts/moderation-safety/moderation-service/ports"
)

func TestApproveIdempotent(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		SubmissionClient: store,
		Clock:            store,
	}
	input := ports.ModerationActionInput{
		SubmissionID: "sub-1",
		CampaignID:   "camp-1",
		Reason:       "manual_review_pass",
	}
	first, err := svc.Approve(context.Background(), "mod-key-1", "mod-1", input)
	if err != nil {
		t.Fatalf("first approve failed: %v", err)
	}
	second, err := svc.Approve(context.Background(), "mod-key-1", "mod-1", input)
	if err != nil {
		t.Fatalf("second approve failed: %v", err)
	}
	if first.DecisionID != second.DecisionID {
		t.Fatalf("expected idempotent replay with same decision id")
	}
}

func TestRejectConflictOnDifferentPayload(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		SubmissionClient: store,
		Clock:            store,
	}
	_, err := svc.Reject(context.Background(), "mod-key-2", "mod-1", ports.ModerationActionInput{
		SubmissionID: "sub-1",
		CampaignID:   "camp-1",
		Reason:       "duplicate_content",
	})
	if err != nil {
		t.Fatalf("seed reject failed: %v", err)
	}
	_, err = svc.Reject(context.Background(), "mod-key-2", "mod-1", ports.ModerationActionInput{
		SubmissionID: "sub-1",
		CampaignID:   "camp-1",
		Reason:       "wrong_platform",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestFlagRequiresReason(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		SubmissionClient: store,
		Clock:            store,
	}
	_, err := svc.Flag(context.Background(), "mod-key-3", "mod-1", ports.ModerationActionInput{
		SubmissionID: "sub-1",
		CampaignID:   "camp-1",
	})
	if !errors.Is(err, domainerrors.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}
