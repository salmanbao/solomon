package memory

import (
	"context"
	"testing"
	"time"

	domainerrors "solomon/contexts/internal-ops/team-management-service/domain/errors"
	"solomon/contexts/internal-ops/team-management-service/ports"
)

func TestCreateTeamAndMembershipCheck(t *testing.T) {
	store := NewStore()
	team, err := store.CreateTeam(
		context.Background(),
		"user_owner_1",
		ports.CreateTeamInput{Name: "Brand Ops", OrgID: "org_1", StorefrontID: "store_1"},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("create team failed: %v", err)
	}
	membership, err := store.CheckMembership(context.Background(), team.TeamID, "user_owner_1")
	if err != nil {
		t.Fatalf("membership check failed: %v", err)
	}
	if membership.Role != ports.RoleOwner {
		t.Fatalf("expected owner role, got %s", membership.Role)
	}
}

func TestAcceptInviteRequiresMatchingM01Email(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	invite, err := store.CreateInvite(context.Background(), "user_owner_1", "team_seed_1", "editor@example.com", ports.RoleEditor, now)
	if err != nil {
		t.Fatalf("create invite failed: %v", err)
	}
	_, err = store.AcceptInvite(context.Background(), "user_viewer_1", invite.Token, now.Add(time.Hour))
	if err == nil {
		t.Fatal("expected forbidden for mismatched email")
	}
	if err != domainerrors.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestOwnerSensitiveActionsRequireMFA(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	_, err := store.UpdateMemberRole(
		context.Background(),
		"user_owner_1",
		"team_seed_1",
		"member_seed_manager",
		ports.RoleEditor,
		"",
		now,
	)
	if err == nil {
		t.Fatal("expected mfa required for owner role change")
	}
	if err != domainerrors.ErrMFARequired {
		t.Fatalf("expected mfa required, got %v", err)
	}
}
