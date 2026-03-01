package application

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/identity-access/onboarding-service/adapters/memory"
	domainerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	"solomon/contexts/identity-access/onboarding-service/ports"
)

func TestConsumeRegisteredThenGetFlow(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	_, err := service.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID: "evt_onb_1",
		UserID:  "user_new_1",
		Role:    "editor",
	})
	if err != nil {
		t.Fatalf("consume event failed: %v", err)
	}
	flow, err := service.GetFlow(context.Background(), "user_new_1")
	if err != nil {
		t.Fatalf("get flow failed: %v", err)
	}
	if flow.UserID != "user_new_1" {
		t.Fatalf("unexpected user_id %s", flow.UserID)
	}
}

func TestCompleteStepIdempotencyReplay(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	_, err := service.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID: "evt_onb_2",
		UserID:  "user_new_2",
		Role:    "brand",
	})
	if err != nil {
		t.Fatalf("consume event failed: %v", err)
	}

	first, err := service.CompleteStep(context.Background(), "idem-onb-1", "user_new_2", "welcome", map[string]any{"device": "web"})
	if err != nil {
		t.Fatalf("first completion failed: %v", err)
	}
	second, err := service.CompleteStep(context.Background(), "idem-onb-1", "user_new_2", "welcome", map[string]any{"device": "web"})
	if err != nil {
		t.Fatalf("second completion failed: %v", err)
	}
	if first.CompletedSteps != second.CompletedSteps {
		t.Fatalf("expected replayed response, got %d vs %d", first.CompletedSteps, second.CompletedSteps)
	}
}

func TestCompleteStepIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}
	_, err := service.ConsumeUserRegisteredEvent(context.Background(), ports.UserRegisteredEvent{
		EventID: "evt_onb_3",
		UserID:  "user_new_3",
		Role:    "influencer",
	})
	if err != nil {
		t.Fatalf("consume event failed: %v", err)
	}
	_, err = service.CompleteStep(context.Background(), "idem-onb-2", "user_new_3", "welcome", map[string]any{"device": "web"})
	if err != nil {
		t.Fatalf("first completion failed: %v", err)
	}
	_, err = service.CompleteStep(context.Background(), "idem-onb-2", "user_new_3", "welcome", map[string]any{"device": "mobile"})
	if err == nil {
		t.Fatal("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}
