package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	authorization "solomon/contexts/identity-access/authorization-service"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	httptransport "solomon/contexts/identity-access/authorization-service/transport/http"
)

func TestAuthorizationGrantAndCheckPermission(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	grant, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-1",
		"admin-1",
		"idem-grant-1",
		httptransport.GrantRoleRequest{
			RoleID: "brand",
			Reason: "brand onboarding",
		},
	)
	if err != nil {
		t.Fatalf("grant role failed: %v", err)
	}
	if grant.AssignmentID == "" {
		t.Fatalf("expected assignment id")
	}

	decision, err := module.Handler.CheckPermissionHandler(
		context.Background(),
		"user-1",
		httptransport.CheckPermissionRequest{
			Permission: "campaign.create",
		},
	)
	if err != nil {
		t.Fatalf("permission check failed: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("expected permission allowed")
	}
}

func TestAuthorizationGrantRoleIdempotencyReplay(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	first, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-2",
		"admin-1",
		"idem-grant-replay",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if err != nil {
		t.Fatalf("first grant failed: %v", err)
	}

	second, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-2",
		"admin-1",
		"idem-grant-replay",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if err != nil {
		t.Fatalf("second grant replay failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed response")
	}
	if first.AssignmentID != second.AssignmentID {
		t.Fatalf("expected same assignment id, got %s and %s", first.AssignmentID, second.AssignmentID)
	}
}

func TestAuthorizationGrantRoleIdempotencyConflict(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-3",
		"admin-1",
		"idem-grant-conflict",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if err != nil {
		t.Fatalf("first grant failed: %v", err)
	}

	_, err = module.Handler.GrantRoleHandler(
		context.Background(),
		"user-3",
		"admin-1",
		"idem-grant-conflict",
		httptransport.GrantRoleRequest{RoleID: "brand"},
	)
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestAuthorizationRevokeRoleRemovesPermission(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-4",
		"admin-1",
		"idem-grant-revoke",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if err != nil {
		t.Fatalf("grant before revoke failed: %v", err)
	}

	_, err = module.Handler.RevokeRoleHandler(
		context.Background(),
		"user-4",
		"admin-1",
		"idem-revoke-1",
		httptransport.RevokeRoleRequest{RoleID: "editor", Reason: "cleanup"},
	)
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	decision, err := module.Handler.CheckPermissionHandler(
		context.Background(),
		"user-4",
		httptransport.CheckPermissionRequest{Permission: "submission.edit"},
	)
	if err != nil {
		t.Fatalf("permission check after revoke failed: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected permission denied after revoke")
	}
}

func TestAuthorizationDelegationRequiresFutureExpiry(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.CreateDelegationHandler(
		context.Background(),
		"idem-delegation-1",
		httptransport.CreateDelegationRequest{
			FromAdminID: "super-admin-1",
			ToAdminID:   "admin-2",
			RoleID:      "admin",
			ExpiresAt:   time.Now().Add(-time.Minute),
			Reason:      "temporary",
		},
	)
	if !errors.Is(err, domainerrors.ErrInvalidDelegation) {
		t.Fatalf("expected invalid delegation, got %v", err)
	}
}

func TestAuthorizationGrantRoleRequiresAdminPermission(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-5",
		"user-without-admin-role",
		"idem-grant-no-admin-perm",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if !errors.Is(err, domainerrors.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestAuthorizationRevokeRoleRequiresAdminPermission(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.GrantRoleHandler(
		context.Background(),
		"user-6",
		"admin-1",
		"idem-grant-for-revoke-perm",
		httptransport.GrantRoleRequest{RoleID: "editor"},
	)
	if err != nil {
		t.Fatalf("grant before revoke permission check failed: %v", err)
	}

	_, err = module.Handler.RevokeRoleHandler(
		context.Background(),
		"user-6",
		"user-without-admin-role",
		"idem-revoke-no-admin-perm",
		httptransport.RevokeRoleRequest{RoleID: "editor"},
	)
	if !errors.Is(err, domainerrors.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestAuthorizationDelegationRequiresDelegatePermission(t *testing.T) {
	module := authorization.NewInMemoryModule(nil)

	_, err := module.Handler.CreateDelegationHandler(
		context.Background(),
		"idem-delegation-no-permission",
		httptransport.CreateDelegationRequest{
			FromAdminID: "admin-1",
			ToAdminID:   "admin-2",
			RoleID:      "admin",
			ExpiresAt:   time.Now().Add(time.Hour),
			Reason:      "temporary",
		},
	)
	if !errors.Is(err, domainerrors.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}
