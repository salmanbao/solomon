package memory

import (
	"context"
	"testing"
	"time"

	domainerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	"solomon/contexts/identity-access/onboarding-service/ports"
)

func TestUserRegisteredCreatesFlowProgress(t *testing.T) {
	store := NewStore()
	flow, err := store.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID:    "evt_reg_1",
		UserID:     "user_onb_1",
		Role:       "editor",
		OccurredAt: time.Now().UTC(),
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("consume event failed: %v", err)
	}
	if flow.UserID != "user_onb_1" {
		t.Fatalf("unexpected user id %s", flow.UserID)
	}
}

func TestUserRegisteredUnknownRoleRejected(t *testing.T) {
	store := NewStore()
	_, err := store.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID: "evt_reg_2",
		UserID:  "user_onb_2",
		Role:    "admin",
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected unknown role error")
	}
	if err != domainerrors.ErrUnknownRole {
		t.Fatalf("expected unknown role, got %v", err)
	}
}

func TestCompleteSkipResumeLifecycle(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	_, err := store.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID: "evt_reg_3",
		UserID:  "user_onb_3",
		Role:    "brand",
	}, now)
	if err != nil {
		t.Fatalf("consume event failed: %v", err)
	}

	if _, err := store.CompleteStep(context.Background(), "user_onb_3", "welcome", map[string]any{}, now); err != nil {
		t.Fatalf("complete step failed: %v", err)
	}
	skipped, err := store.SkipFlow(context.Background(), "user_onb_3", "already_know", now)
	if err != nil {
		t.Fatalf("skip flow failed: %v", err)
	}
	if skipped.Status != "skipped" {
		t.Fatalf("unexpected skip status %s", skipped.Status)
	}

	resumed, err := store.ResumeFlow(context.Background(), "user_onb_3", now)
	if err != nil {
		t.Fatalf("resume flow failed: %v", err)
	}
	if resumed.Status != "in_progress" {
		t.Fatalf("unexpected resume status %s", resumed.Status)
	}
}
