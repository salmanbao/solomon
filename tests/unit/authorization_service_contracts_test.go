package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthorizationServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "authorization-service.openapi.json"))
	if err != nil {
		t.Fatalf("read authorization-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode authorization-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/authz/v1/check":                        {"post"},
		"/api/authz/v1/check-batch":                  {"post"},
		"/api/authz/v1/users/{user_id}/roles":        {"get"},
		"/api/authz/v1/users/{user_id}/roles/grant":  {"post"},
		"/api/authz/v1/users/{user_id}/roles/revoke": {"post"},
		"/api/authz/v1/delegations":                  {"post"},
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

func TestAuthorizationServiceOpenAPIContractRequiresSecurityAndIdempotencyHeaders(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "authorization-service.openapi.json"))
	if err != nil {
		t.Fatalf("read authorization-service openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode authorization-service openapi: %v", err)
	}

	allOps := map[string]string{
		"/api/authz/v1/check":                        "post",
		"/api/authz/v1/check-batch":                  "post",
		"/api/authz/v1/users/{user_id}/roles":        "get",
		"/api/authz/v1/users/{user_id}/roles/grant":  "post",
		"/api/authz/v1/users/{user_id}/roles/revoke": "post",
		"/api/authz/v1/delegations":                  "post",
	}

	for path, method := range allOps {
		ops, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("missing path in openapi contract: %s", path)
		}
		if !isHeaderRequiredWithRefs(ops[method], "Authorization", doc.Components.Parameters) {
			t.Fatalf("expected Authorization header required for %s %s", method, path)
		}
		if !isHeaderRequiredWithRefs(ops[method], "X-Request-Id", doc.Components.Parameters) {
			t.Fatalf("expected X-Request-Id header required for %s %s", method, path)
		}
	}

	mutating := map[string]string{
		"/api/authz/v1/users/{user_id}/roles/grant":  "post",
		"/api/authz/v1/users/{user_id}/roles/revoke": "post",
		"/api/authz/v1/delegations":                  "post",
	}
	for path, method := range mutating {
		ops := doc.Paths[path]
		if !isHeaderRequiredWithRefs(ops[method], "Idempotency-Key", doc.Components.Parameters) {
			t.Fatalf("expected Idempotency-Key header required for %s %s", method, path)
		}
	}
}

func TestAuthorizationServiceOpenAPIContractDeclaresUnauthorizedResponse(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "authorization-service.openapi.json"))
	if err != nil {
		t.Fatalf("read authorization-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode authorization-service openapi: %v", err)
	}

	ops := map[string]string{
		"/api/authz/v1/check":                        "post",
		"/api/authz/v1/check-batch":                  "post",
		"/api/authz/v1/users/{user_id}/roles":        "get",
		"/api/authz/v1/users/{user_id}/roles/grant":  "post",
		"/api/authz/v1/users/{user_id}/roles/revoke": "post",
		"/api/authz/v1/delegations":                  "post",
	}

	for path, method := range ops {
		pathItem, ok := doc.Paths[path]
		if !ok {
			t.Fatalf("missing path in openapi contract: %s", path)
		}
		op, ok := pathItem[method]
		if !ok {
			t.Fatalf("missing operation %s %s in openapi contract", method, path)
		}
		responses, ok := op["responses"].(map[string]any)
		if !ok {
			t.Fatalf("missing responses block for %s %s", method, path)
		}
		if _, ok := responses["401"]; !ok {
			t.Fatalf("expected 401 response for %s %s", method, path)
		}
	}
}
