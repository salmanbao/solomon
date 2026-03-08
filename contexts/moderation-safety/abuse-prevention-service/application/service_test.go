package application

import (
	"context"
	"testing"

	"solomon/contexts/moderation-safety/abuse-prevention-service/adapters/memory"
	domainerrors "solomon/contexts/moderation-safety/abuse-prevention-service/domain/errors"
)

func TestReleaseLockoutIdempotentReplayAndConflict(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:        store,
		Idempotency: store,
		Clock:       store,
	}

	input := ReleaseLockoutInput{
		ActorID:       "admin-1",
		UserID:        "locked-user-1",
		Reason:        "manual false-positive recovery",
		SourceIP:      "127.0.0.1",
		CorrelationID: "corr-abuse-1",
	}
	first, err := svc.ReleaseLockout(context.Background(), "idem-abuse-1", input)
	if err != nil {
		t.Fatalf("first release failed: %v", err)
	}
	replay, err := svc.ReleaseLockout(context.Background(), "idem-abuse-1", input)
	if err != nil {
		t.Fatalf("replay release failed: %v", err)
	}
	if first.ThreatID == "" || first.OwnerAuditLogID == "" {
		t.Fatalf("expected threat id and owner audit id")
	}
	if replay.ThreatID != first.ThreatID {
		t.Fatalf("expected replay threat id %q, got %q", first.ThreatID, replay.ThreatID)
	}

	_, err = svc.ReleaseLockout(context.Background(), "idem-abuse-1", ReleaseLockoutInput{
		ActorID: "admin-1",
		UserID:  "locked-user-1",
		Reason:  "different reason with same idempotency key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestReleaseLockoutRejectsForbiddenActor(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:        store,
		Idempotency: store,
		Clock:       store,
	}
	_, err := svc.ReleaseLockout(context.Background(), "idem-abuse-2", ReleaseLockoutInput{
		ActorID: "viewer-1",
		UserID:  "locked-user-1",
		Reason:  "viewer should not be allowed",
	})
	if err == nil {
		t.Fatalf("expected forbidden error")
	}
	if err != domainerrors.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
