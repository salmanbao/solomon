package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/content-library-marketplace/transport/http"
)

func TestContentLibraryMarketplaceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "content-library-marketplace.openapi.json"))
	if err != nil {
		t.Fatalf("read content-library-marketplace openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode content-library-marketplace openapi: %v", err)
	}

	expected := map[string][]string{
		"/library/clips":                           {"get"},
		"/library/clips/{clip_id}":                 {"get"},
		"/library/clips/{clip_id}/preview":         {"get"},
		"/library/clips/{clip_id}/claim":           {"post"},
		"/library/clips/{clip_id}/download":        {"post"},
		"/library/claims":                          {"get"},
		"/v1/marketplace/clips":                    {"get"},
		"/v1/marketplace/clips/{clip_id}":          {"get"},
		"/v1/marketplace/clips/{clip_id}/preview":  {"get"},
		"/v1/marketplace/clips/{clip_id}/claim":    {"post"},
		"/v1/marketplace/clips/{clip_id}/download": {"post"},
		"/v1/marketplace/claims":                   {"get"},
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

	libraryClaim := doc.Paths["/library/clips/{clip_id}/claim"]["post"]
	v1Claim := doc.Paths["/v1/marketplace/clips/{clip_id}/claim"]["post"]

	if isHeaderRequired(libraryClaim, "Idempotency-Key") {
		t.Fatalf("legacy /library claim route should keep non-breaking optional Idempotency-Key")
	}
	if !isHeaderRequired(v1Claim, "Idempotency-Key") {
		t.Fatalf("canonical /v1 claim route must require Idempotency-Key")
	}
}

func TestContentLibraryMarketplaceEventSchemasCoverCanonicalEventSet(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cases := map[string][]string{
		"distribution.claimed":   {"claim_id", "clip_id", "user_id", "claim_type"},
		"distribution.published": {"claim_id"},
		"distribution.failed":    {"claim_id"},
	}

	for eventType, requiredFields := range cases {
		path := filepath.Join(root, "contracts", "events", "v1", eventType+".schema.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read event schema %s: %v", eventType, err)
		}

		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("decode event schema %s: %v", eventType, err)
		}
		if title, _ := schema["title"].(string); title != eventType {
			t.Fatalf("schema %s has wrong title: %q", eventType, title)
		}

		required, _ := schema["required"].([]any)
		for _, key := range requiredFields {
			if !containsAnyString(required, key) {
				t.Fatalf("schema %s missing required payload key %s", eventType, key)
			}
		}
	}
}

func TestContentLibraryMarketplaceClaimedEventEnvelopeContractConsistency(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-contract-1",
			Title:           "Contract Clip",
			Niche:           "fitness",
			DurationSeconds: 20,
			PreviewURL:      "https://cdn.example/preview",
			DownloadAssetID: "asset-contract-1",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			ClaimLimit:      50,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	resp, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-contract-1",
		"clip-contract-1",
		httptransport.ClaimClipRequest{RequestID: "request-contract-1"},
		"idem-contract-1",
	)
	if err != nil {
		t.Fatalf("claim clip failed: %v", err)
	}

	outbox := module.Store.OutboxEvents()
	if len(outbox) == 0 {
		t.Fatalf("expected claimed event in outbox")
	}

	foundClaimed := false
	for _, message := range outbox {
		var envelope map[string]any
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox payload failed: %v", err)
		}

		eventType, _ := envelope["event_type"].(string)
		if eventType != "distribution.claimed" {
			continue
		}
		foundClaimed = true

		if sourceService, _ := envelope["source_service"].(string); sourceService != "content-marketplace-service" {
			t.Fatalf("invalid source_service for claimed event: %q", sourceService)
		}
		if partitionPath, _ := envelope["partition_key_path"].(string); partitionPath != "clip_id" {
			t.Fatalf("invalid partition_key_path for claimed event: %q", partitionPath)
		}
		partitionKey, _ := envelope["partition_key"].(string)
		if strings.TrimSpace(partitionKey) == "" {
			t.Fatalf("claimed event missing partition_key")
		}

		data, _ := envelope["data"].(map[string]any)
		claimID, _ := data["claim_id"].(string)
		clipID, _ := data["clip_id"].(string)
		userID, _ := data["user_id"].(string)
		claimType, _ := data["claim_type"].(string)

		if strings.TrimSpace(claimID) == "" || claimID != resp.ClaimID {
			t.Fatalf("claimed event has invalid claim_id: %q", claimID)
		}
		if strings.TrimSpace(clipID) == "" || clipID != resp.ClipID || clipID != partitionKey {
			t.Fatalf("claimed event has invalid clip_id/partition_key: clip_id=%q partition_key=%q", clipID, partitionKey)
		}
		if strings.TrimSpace(userID) == "" || userID != "user-contract-1" {
			t.Fatalf("claimed event has invalid user_id: %q", userID)
		}
		if strings.TrimSpace(claimType) == "" {
			t.Fatalf("claimed event missing claim_type")
		}
	}

	if !foundClaimed {
		t.Fatalf("expected distribution.claimed event in outbox")
	}
}

func isHeaderRequired(operation any, name string) bool {
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
		paramName, _ := param["name"].(string)
		if !strings.EqualFold(paramName, name) {
			continue
		}
		required, _ := param["required"].(bool)
		return required
	}
	return false
}
