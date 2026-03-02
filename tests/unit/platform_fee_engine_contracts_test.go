package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPlatformFeeEngineOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "platform-fee-engine.openapi.json"))
	if err != nil {
		t.Fatalf("read platform-fee-engine openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode platform-fee-engine openapi: %v", err)
	}

	expected := map[string][]string{
		"/v1/fees/calculate":    {"post"},
		"/v1/fees/history":      {"get"},
		"/v1/admin/fees/report": {"get"},
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

func TestPlatformFeeEngineOpenAPIContractRequiresHeaders(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "platform-fee-engine.openapi.json"))
	if err != nil {
		t.Fatalf("read platform-fee-engine openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode platform-fee-engine openapi: %v", err)
	}

	allOps := map[string]string{
		"/v1/fees/calculate":    "post",
		"/v1/fees/history":      "get",
		"/v1/admin/fees/report": "get",
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

	calculate := doc.Paths["/v1/fees/calculate"]
	if !isHeaderRequiredWithRefs(calculate["post"], "Idempotency-Key", doc.Components.Parameters) {
		t.Fatalf("expected Idempotency-Key header required for post /v1/fees/calculate")
	}
}

func TestPlatformFeeCalculatedEventSchemaMatchesCanonicalEnvelope(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "contracts", "events", "v1", "fee.calculated.schema.json"))
	if err != nil {
		t.Fatalf("read fee.calculated schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode fee.calculated schema: %v", err)
	}

	requiredEnvelopeFields := []string{
		"event_id",
		"event_type",
		"occurred_at",
		"source_service",
		"trace_id",
		"schema_version",
		"partition_key_path",
		"partition_key",
		"data",
	}
	required, _ := schema["required"].([]any)
	for _, key := range requiredEnvelopeFields {
		if !containsAnyString(required, key) {
			t.Fatalf("fee.calculated schema missing required envelope key %s", key)
		}
	}

	properties, _ := schema["properties"].(map[string]any)
	eventTypeProp, _ := properties["event_type"].(map[string]any)
	if eventConst, _ := eventTypeProp["const"].(string); eventConst != "fee.calculated" {
		t.Fatalf("fee.calculated schema has wrong event_type const: %q", eventConst)
	}

	partitionPathProp, _ := properties["partition_key_path"].(map[string]any)
	if partitionConst, _ := partitionPathProp["const"].(string); partitionConst != "submission_id" {
		t.Fatalf("fee.calculated schema has wrong partition_key_path const: %q", partitionConst)
	}
}
