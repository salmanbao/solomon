package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAdminDashboardOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "admin-dashboard-service.openapi.json"))
	if err != nil {
		t.Fatalf("read admin-dashboard-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode admin-dashboard-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/admin/v1/actions/log":                                            {"post"},
		"/api/admin/v1/identity/roles/grant":                                   {"post"},
		"/api/admin/v1/moderation/decisions":                                   {"post"},
		"/api/admin/v1/abuse-prevention/lockouts/{user_id}/release":            {"post"},
		"/api/admin/v1/finance/refunds":                                        {"post"},
		"/api/admin/v1/finance/billing/invoices/{invoice_id}/refund":           {"post"},
		"/api/admin/v1/finance/rewards/recalculate":                            {"post"},
		"/api/admin/v1/finance/affiliates/{affiliate_id}/suspend":              {"post"},
		"/api/admin/v1/finance/affiliates/{affiliate_id}/attributions":         {"post"},
		"/api/admin/v1/finance/payouts/{payout_id}/retry":                      {"post"},
		"/api/admin/v1/compliance/disputes/{dispute_id}/resolve":               {"post"},
		"/api/admin/v1/compliance/disputes/{dispute_id}/reopen":                {"post"},
		"/api/admin/v1/compliance/consent/{user_id}":                           {"get"},
		"/api/admin/v1/compliance/consent/{user_id}/update":                    {"post"},
		"/api/admin/v1/compliance/consent/{user_id}/withdraw":                  {"post"},
		"/api/admin/v1/compliance/exports":                                     {"post"},
		"/api/admin/v1/compliance/exports/{request_id}":                        {"get"},
		"/api/admin/v1/compliance/deletion-requests":                           {"post"},
		"/api/admin/v1/compliance/retention/legal-holds":                       {"post"},
		"/api/admin/v1/compliance/legal-holds/check":                           {"get"},
		"/api/admin/v1/compliance/legal-holds/{hold_id}/release":               {"post"},
		"/api/admin/v1/compliance/legal/compliance-scans":                      {"post"},
		"/api/admin/v1/support/tickets/{ticket_id}":                            {"get", "patch"},
		"/api/admin/v1/support/tickets/search":                                 {"get"},
		"/api/admin/v1/support/tickets/{ticket_id}/assign":                     {"post"},
		"/api/admin/v1/creator-workflow/editor/campaigns/{campaign_id}/save":   {"post"},
		"/api/admin/v1/creator-workflow/clipping/projects/{project_id}/export": {"post"},
		"/api/admin/v1/creator-workflow/auto-clipping/models/deploy":           {"post"},
		"/api/admin/v1/integrations/keys/rotate":                               {"post"},
		"/api/admin/v1/integrations/workflows/test":                            {"post"},
		"/api/admin/v1/integrations/webhooks/{webhook_id}/replay":              {"post"},
		"/api/admin/v1/integrations/webhooks/{webhook_id}/disable":             {"post"},
		"/api/admin/v1/integrations/webhooks/{webhook_id}/deliveries":          {"get"},
		"/api/admin/v1/integrations/webhooks/{webhook_id}/analytics":           {"get"},
		"/api/admin/v1/platform-ops/migrations/plans":                          {"post", "get"},
		"/api/admin/v1/platform-ops/migrations/runs":                           {"post"},
	}

	for path, methods := range expected {
		ops, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("missing path in openapi contract: %s", path)
		}
		for _, method := range methods {
			if _, ok := ops[method]; !ok {
				t.Fatalf("missing method %s for path %s in openapi contract", method, path)
			}
		}
	}
}

func TestAdminDashboardOpenAPIContractRequiresIdempotencyOnMutations(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "admin-dashboard-service.openapi.json"))
	if err != nil {
		t.Fatalf("read admin-dashboard-service openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode admin-dashboard-service openapi: %v", err)
	}

	mutating := map[string]string{
		"/api/admin/v1/actions/log":                                            "post",
		"/api/admin/v1/identity/roles/grant":                                   "post",
		"/api/admin/v1/moderation/decisions":                                   "post",
		"/api/admin/v1/abuse-prevention/lockouts/{user_id}/release":            "post",
		"/api/admin/v1/finance/refunds":                                        "post",
		"/api/admin/v1/finance/billing/invoices/{invoice_id}/refund":           "post",
		"/api/admin/v1/finance/rewards/recalculate":                            "post",
		"/api/admin/v1/finance/affiliates/{affiliate_id}/suspend":              "post",
		"/api/admin/v1/finance/affiliates/{affiliate_id}/attributions":         "post",
		"/api/admin/v1/finance/payouts/{payout_id}/retry":                      "post",
		"/api/admin/v1/compliance/disputes/{dispute_id}/resolve":               "post",
		"/api/admin/v1/compliance/disputes/{dispute_id}/reopen":                "post",
		"/api/admin/v1/compliance/consent/{user_id}/update":                    "post",
		"/api/admin/v1/compliance/consent/{user_id}/withdraw":                  "post",
		"/api/admin/v1/compliance/exports":                                     "post",
		"/api/admin/v1/compliance/deletion-requests":                           "post",
		"/api/admin/v1/compliance/retention/legal-holds":                       "post",
		"/api/admin/v1/compliance/legal-holds/{hold_id}/release":               "post",
		"/api/admin/v1/compliance/legal/compliance-scans":                      "post",
		"/api/admin/v1/support/tickets/{ticket_id}/assign":                     "post",
		"/api/admin/v1/support/tickets/{ticket_id}":                            "patch",
		"/api/admin/v1/creator-workflow/editor/campaigns/{campaign_id}/save":   "post",
		"/api/admin/v1/creator-workflow/clipping/projects/{project_id}/export": "post",
		"/api/admin/v1/creator-workflow/auto-clipping/models/deploy":           "post",
		"/api/admin/v1/integrations/keys/rotate":                               "post",
		"/api/admin/v1/integrations/workflows/test":                            "post",
		"/api/admin/v1/integrations/webhooks/{webhook_id}/replay":              "post",
		"/api/admin/v1/integrations/webhooks/{webhook_id}/disable":             "post",
		"/api/admin/v1/platform-ops/migrations/plans":                          "post",
		"/api/admin/v1/platform-ops/migrations/runs":                           "post",
	}

	for path, method := range mutating {
		ops, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("missing path in openapi contract: %s", path)
		}
		if !isHeaderRequiredWithRefs(ops[method], "Idempotency-Key", doc.Components.Parameters) {
			t.Fatalf("expected Idempotency-Key header required for %s %s", method, path)
		}
	}
}
