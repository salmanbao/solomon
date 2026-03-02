package application

import (
	"context"
	"errors"
	"testing"

	"solomon/contexts/campaign-editorial/influencer-dashboard-service/adapters/memory"
	domainerrors "solomon/contexts/campaign-editorial/influencer-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/ports"
)

func TestCreateGoalIdempotent(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:                 store,
		Idempotency:          store,
		RewardProvider:       store,
		GamificationProvider: store,
		Clock:                store,
	}
	input := ports.GoalCreateInput{
		UserID:      "creator-1",
		GoalType:    "earnings",
		GoalName:    "Earn 500",
		TargetValue: 500,
		StartDate:   "2026-02-05",
		EndDate:     "2026-02-11",
	}
	first, err := svc.CreateGoal(context.Background(), "goal-key-1", input)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := svc.CreateGoal(context.Background(), "goal-key-1", input)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected idempotent replay to return same goal id")
	}
}

func TestCreateGoalIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:                 store,
		Idempotency:          store,
		RewardProvider:       store,
		GamificationProvider: store,
		Clock:                store,
	}
	_, err := svc.CreateGoal(context.Background(), "goal-key-2", ports.GoalCreateInput{
		UserID:      "creator-1",
		GoalType:    "earnings",
		GoalName:    "Earn 500",
		TargetValue: 500,
	})
	if err != nil {
		t.Fatalf("seed create failed: %v", err)
	}
	_, err = svc.CreateGoal(context.Background(), "goal-key-2", ports.GoalCreateInput{
		UserID:      "creator-1",
		GoalType:    "earnings",
		GoalName:    "Earn 900",
		TargetValue: 900,
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestSummaryDependencyReadinessDegraded(t *testing.T) {
	store := memory.NewStore()
	store.FailRewardProvider = true
	svc := Service{
		Repo:                 store,
		Idempotency:          store,
		RewardProvider:       store,
		GamificationProvider: store,
		Clock:                store,
	}
	summary, err := svc.GetSummary(context.Background(), "creator-1")
	if err != nil {
		t.Fatalf("summary should succeed with degraded dependency: %v", err)
	}
	if summary.DependencyStatus["m41_reward_engine"] != "degraded" {
		t.Fatalf("expected m41 degraded")
	}
	if summary.DependencyStatus["m47_gamification_service"] != "ready" {
		t.Fatalf("expected m47 ready")
	}
}
