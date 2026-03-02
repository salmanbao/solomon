package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGamificationServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "gamification-service.openapi.json"))
	if err != nil {
		t.Fatalf("read gamification-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode gamification-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/v1/gamification/points/award":            {"post"},
		"/api/v1/gamification/badges/grant":            {"post"},
		"/api/v1/gamification/users/{user_id}/summary": {"get"},
		"/api/v1/gamification/leaderboard":             {"get"},
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

func TestGamificationServiceOpenAPIContractRequiresHeaders(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "gamification-service.openapi.json"))
	if err != nil {
		t.Fatalf("read gamification-service openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode gamification-service openapi: %v", err)
	}

	allOps := map[string]string{
		"/api/v1/gamification/points/award":            "post",
		"/api/v1/gamification/badges/grant":            "post",
		"/api/v1/gamification/users/{user_id}/summary": "get",
		"/api/v1/gamification/leaderboard":             "get",
	}
	for path, method := range allOps {
		ops := doc.Paths[path]
		if !isHeaderRequiredWithRefs(ops[method], "Authorization", doc.Components.Parameters) {
			t.Fatalf("expected Authorization header required for %s %s", method, path)
		}
		if !isHeaderRequiredWithRefs(ops[method], "X-Request-Id", doc.Components.Parameters) {
			t.Fatalf("expected X-Request-Id header required for %s %s", method, path)
		}
	}

	mutating := map[string]string{
		"/api/v1/gamification/points/award": "post",
		"/api/v1/gamification/badges/grant": "post",
	}
	for path, method := range mutating {
		ops := doc.Paths[path]
		if !isHeaderRequiredWithRefs(ops[method], "Idempotency-Key", doc.Components.Parameters) {
			t.Fatalf("expected Idempotency-Key header required for %s %s", method, path)
		}
	}
}
