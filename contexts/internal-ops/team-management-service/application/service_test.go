package application

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/internal-ops/team-management-service/adapters/memory"
	domainerrors "solomon/contexts/internal-ops/team-management-service/domain/errors"
	"solomon/contexts/internal-ops/team-management-service/ports"
)

func TestCreateTeamIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	input := ports.CreateTeamInput{Name: "Brand Ops", OrgID: "org_1", StorefrontID: "store_1"}
	first, err := service.CreateTeam(context.Background(), "idem-m87-1", "user_owner_1", input)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := service.CreateTeam(context.Background(), "idem-m87-1", "user_owner_1", input)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if first.TeamID != second.TeamID {
		t.Fatalf("expected same team id, got %s vs %s", first.TeamID, second.TeamID)
	}
}

func TestCreateTeamIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	_, err := service.CreateTeam(
		context.Background(),
		"idem-m87-2",
		"user_owner_1",
		ports.CreateTeamInput{Name: "Brand Ops", OrgID: "org_1"},
	)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = service.CreateTeam(
		context.Background(),
		"idem-m87-2",
		"user_owner_1",
		ports.CreateTeamInput{Name: "Growth Team", OrgID: "org_1"},
	)
	if err == nil {
		t.Fatal("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}
