package httpserver

import "testing"

func TestAdminNamespaceOwnershipAllowsDeclaredRoutes(t *testing.T) {
	if err := validateAdminRouteOwnership(adminRouteOwnerM20, "POST /api/admin/v1/impersonation/start"); err != nil {
		t.Fatalf("expected M20 route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/identity/roles/grant"); err != nil {
		t.Fatalf("expected M86 route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/finance/refunds"); err != nil {
		t.Fatalf("expected M86 finance route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/finance/rewards/recalculate"); err != nil {
		t.Fatalf("expected M86 reward route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/compliance/exports"); err != nil {
		t.Fatalf("expected M86 compliance export route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "PATCH /api/admin/v1/support/tickets/{ticket_id}"); err != nil {
		t.Fatalf("expected M86 support route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/creator-workflow/editor/campaigns/{campaign_id}/save"); err != nil {
		t.Fatalf("expected M86 creator workflow editor route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/creator-workflow/clipping/projects/{project_id}/export"); err != nil {
		t.Fatalf("expected M86 creator workflow clipping route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/creator-workflow/auto-clipping/models/deploy"); err != nil {
		t.Fatalf("expected M86 creator workflow auto-clipping route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/integrations/keys/rotate"); err != nil {
		t.Fatalf("expected M86 integration key route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "GET /api/admin/v1/integrations/webhooks/{webhook_id}/analytics"); err != nil {
		t.Fatalf("expected M86 webhook analytics route to validate, got %v", err)
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/platform-ops/migrations/runs"); err != nil {
		t.Fatalf("expected M86 migration run route to validate, got %v", err)
	}
}

func TestAdminNamespaceOwnershipRejectsCrossOwnership(t *testing.T) {
	if err := validateAdminRouteOwnership(adminRouteOwnerM20, "POST /api/admin/v1/identity/roles/grant"); err == nil {
		t.Fatalf("expected ownership mismatch for M20 claiming M86 route")
	}
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/users/{user_id}/ban"); err == nil {
		t.Fatalf("expected ownership mismatch for M86 claiming M20 route")
	}
}

func TestAdminNamespaceOwnershipRejectsUnmappedRoute(t *testing.T) {
	if err := validateAdminRouteOwnership(adminRouteOwnerM86, "POST /api/admin/v1/unknown/path"); err == nil {
		t.Fatalf("expected unmapped route validation failure")
	}
}
