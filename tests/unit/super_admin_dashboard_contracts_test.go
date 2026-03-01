package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSuperAdminDashboardOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "super-admin-dashboard.openapi.json"))
	if err != nil {
		t.Fatalf("read super-admin-dashboard openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode super-admin-dashboard openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/admin/v1/impersonation/start":                  {"post"},
		"/api/admin/v1/impersonation/end":                    {"post"},
		"/api/admin/v1/users/{user_id}/wallet/adjust":        {"post"},
		"/api/admin/v1/users/{user_id}/wallet/history":       {"get"},
		"/api/admin/v1/users/{user_id}/ban":                  {"post"},
		"/api/admin/v1/users/{user_id}/unban":                {"post"},
		"/api/admin/v1/users/search":                         {"get"},
		"/api/admin/v1/users/bulk-action":                    {"post"},
		"/api/admin/v1/campaigns/{campaign_id}/pause":        {"post"},
		"/api/admin/v1/campaigns/{campaign_id}/adjust":       {"patch"},
		"/api/admin/v1/submissions/{submission_id}/override": {"post"},
		"/api/admin/v1/feature-flags":                        {"get"},
		"/api/admin/v1/feature-flags/{flag_key}/toggle":      {"post"},
		"/api/admin/v1/analytics/dashboard":                  {"get"},
		"/api/admin/v1/audit-logs":                           {"get"},
		"/api/admin/v1/audit-logs/export":                    {"get"},
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

func TestSuperAdminDashboardOpenAPIContractRequiresIdempotencyOnMutations(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "super-admin-dashboard.openapi.json"))
	if err != nil {
		t.Fatalf("read super-admin-dashboard openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode super-admin-dashboard openapi: %v", err)
	}

	mutating := map[string]string{
		"/api/admin/v1/impersonation/start":                  "post",
		"/api/admin/v1/impersonation/end":                    "post",
		"/api/admin/v1/users/{user_id}/wallet/adjust":        "post",
		"/api/admin/v1/users/{user_id}/ban":                  "post",
		"/api/admin/v1/users/{user_id}/unban":                "post",
		"/api/admin/v1/users/bulk-action":                    "post",
		"/api/admin/v1/campaigns/{campaign_id}/pause":        "post",
		"/api/admin/v1/campaigns/{campaign_id}/adjust":       "patch",
		"/api/admin/v1/submissions/{submission_id}/override": "post",
		"/api/admin/v1/feature-flags/{flag_key}/toggle":      "post",
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

func isHeaderRequiredWithRefs(operation any, name string, componentParams map[string]map[string]any) bool {
	opMap, ok := operation.(map[string]any)
	if !ok {
		return false
	}
	rawParams, ok := opMap["parameters"].([]any)
	if !ok {
		return false
	}
	for _, raw := range rawParams {
		param, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if ref, _ := param["$ref"].(string); ref != "" {
			const prefix = "#/components/parameters/"
			if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
				key := ref[len(prefix):]
				if resolved, ok := componentParams[key]; ok {
					param = resolved
				}
			}
		}
		paramName, _ := param["name"].(string)
		if paramName != name {
			continue
		}
		required, _ := param["required"].(bool)
		return required
	}
	return false
}
