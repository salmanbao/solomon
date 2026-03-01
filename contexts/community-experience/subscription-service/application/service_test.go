package application

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/community-experience/subscription-service/adapters/memory"
	"solomon/contexts/community-experience/subscription-service/ports"
)

func TestCreateSubscriptionIdempotencyReplay(t *testing.T) {
	store := memory.NewStore()
	now := time.Date(2026, time.February, 6, 12, 0, 0, 0, time.UTC)

	service := Service{
		Repo:        store,
		Idempotency: store,
		Clock:       fixedClock{now: now},
	}

	first, err := service.CreateSubscription(context.Background(), "idem-1", "user_1", createInput("plan_pro_monthly", false))
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := service.CreateSubscription(context.Background(), "idem-1", "user_1", createInput("plan_pro_monthly", false))
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if first.SubscriptionID != second.SubscriptionID {
		t.Fatalf("expected replayed subscription id, got %s vs %s", first.SubscriptionID, second.SubscriptionID)
	}
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time { return f.now }

func createInput(planID string, trial bool) ports.CreateSubscriptionInput {
	return ports.CreateSubscriptionInput{
		PlanID: planID,
		Trial:  trial,
	}
}
