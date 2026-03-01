package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProductServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "product-service.openapi.json"))
	if err != nil {
		t.Fatalf("read product-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode product-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/v1/products":                              {"get", "post"},
		"/api/v1/products/{product_id}/access":          {"get"},
		"/api/v1/products/{id}/purchase":                {"post"},
		"/api/v1/products/{id}/fulfill":                 {"post"},
		"/api/v1/admin/products/{product_id}/inventory": {"post"},
		"/api/v1/products/{product_id}/media/reorder":   {"put"},
		"/api/v1/discover":                              {"get"},
		"/api/v1/search":                                {"get"},
		"/api/v1/users/{user_id}/data-export":           {"get"},
		"/api/v1/users/{user_id}/delete-account":        {"post"},
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

func TestProductServiceOpenAPIContractRequiresIdempotencyOnMutations(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "product-service.openapi.json"))
	if err != nil {
		t.Fatalf("read product-service openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode product-service openapi: %v", err)
	}

	mutating := map[string]string{
		"/api/v1/products":                              "post",
		"/api/v1/products/{id}/purchase":                "post",
		"/api/v1/products/{id}/fulfill":                 "post",
		"/api/v1/admin/products/{product_id}/inventory": "post",
		"/api/v1/products/{product_id}/media/reorder":   "put",
		"/api/v1/users/{user_id}/delete-account":        "post",
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
