package memory

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/community-experience/subscription-service/ports"
)

func TestCreateSubscriptionTrialEligibilityPerPlan(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, time.February, 6, 12, 0, 0, 0, time.UTC)

	first, err := store.CreateSubscription(context.Background(), "user_1", ports.CreateSubscriptionInput{
		PlanID: "plan_pro_monthly",
		Trial:  true,
	}, now)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if first.Status != "trialing" {
		t.Fatalf("expected trialing status, got %q", first.Status)
	}

	_, err = store.CancelSubscription(context.Background(), "user_1", first.SubscriptionID, false, "done", now.Add(time.Hour))
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}

	_, err = store.CreateSubscription(context.Background(), "user_1", ports.CreateSubscriptionInput{
		PlanID: "plan_pro_monthly",
		Trial:  true,
	}, now.Add(2*time.Hour))
	if err == nil {
		t.Fatal("expected trial reuse to fail for same plan")
	}
}

func TestChangePlanPreservesBillingAnchorAndPeriodEnd(t *testing.T) {
	store := NewStore()
	now := time.Date(2026, time.February, 6, 12, 0, 0, 0, time.UTC)

	item, err := store.CreateSubscription(context.Background(), "user_2", ports.CreateSubscriptionInput{
		PlanID: "plan_pro_monthly",
		Trial:  false,
	}, now)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	oldEnd := *item.CurrentPeriodEnd
	oldAnchor := item.BillingAnchorDay

	changed, err := store.ChangePlan(context.Background(), "user_2", item.SubscriptionID, "plan_enterprise_monthly", now.Add(5*24*time.Hour))
	if err != nil {
		t.Fatalf("change plan failed: %v", err)
	}
	if changed.ProrationAmountCents <= 0 {
		t.Fatalf("expected positive upgrade proration, got %d", changed.ProrationAmountCents)
	}

	canceled, err := store.CancelSubscription(context.Background(), "user_2", item.SubscriptionID, true, "cost", now.Add(6*24*time.Hour))
	if err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if canceled.AccessEndsAt == nil || !canceled.AccessEndsAt.Equal(oldEnd) {
		t.Fatalf("expected access_ends_at to preserve period end")
	}
	if oldAnchor != item.BillingAnchorDay {
		t.Fatalf("billing anchor changed unexpectedly")
	}
}

func TestCreateSubscriptionRejectsMissingProductDependency(t *testing.T) {
	store := NewStore()
	store.mu.Lock()
	store.productIDsByPlanID["plan_pro_monthly"] = "missing_prod"
	store.mu.Unlock()

	_, err := store.CreateSubscription(context.Background(), "user_3", ports.CreateSubscriptionInput{
		PlanID: "plan_pro_monthly",
		Trial:  false,
	}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected missing product dependency error")
	}
}
