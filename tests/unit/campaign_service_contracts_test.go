package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	httptransport "solomon/contexts/campaign-editorial/campaign-service/transport/http"
)

func TestCampaignServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "campaign-service.openapi.json"))
	if err != nil {
		t.Fatalf("read campaign-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode campaign-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/v1/campaigns":                                        {"post", "get"},
		"/v1/campaigns/{campaign_id}":                          {"get", "put"},
		"/v1/campaigns/{campaign_id}/launch":                   {"post"},
		"/v1/campaigns/{campaign_id}/pause":                    {"post"},
		"/v1/campaigns/{campaign_id}/resume":                   {"post"},
		"/v1/campaigns/{campaign_id}/complete":                 {"post"},
		"/v1/campaigns/{campaign_id}/media/upload-url":         {"post"},
		"/v1/campaigns/{campaign_id}/media/{media_id}/confirm": {"post"},
		"/v1/campaigns/{campaign_id}/media":                    {"get"},
		"/v1/campaigns/{campaign_id}/analytics":                {"get"},
		"/v1/campaigns/{campaign_id}/analytics/export":         {"get"},
		"/v1/campaigns/{campaign_id}/budget/increase":          {"post"},
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

func TestCampaignServiceEventSchemasCoverCanonicalEventSet(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	eventTypes := []string{
		"campaign.created",
		"campaign.launched",
		"campaign.paused",
		"campaign.resumed",
		"campaign.completed",
		"campaign.budget_updated",
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

	for _, eventType := range eventTypes {
		raw, err := os.ReadFile(filepath.Join(root, "contracts", "events", "v1", eventType+".schema.json"))
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
		for _, key := range requiredEnvelopeFields {
			if !containsAnyString(required, key) {
				t.Fatalf("schema %s missing required envelope key %s", eventType, key)
			}
		}

		properties, _ := schema["properties"].(map[string]any)
		eventTypeProp, _ := properties["event_type"].(map[string]any)
		if eventConst, _ := eventTypeProp["const"].(string); eventConst != eventType {
			t.Fatalf("schema %s has wrong event_type const: %q", eventType, eventConst)
		}

		partitionPathProp, _ := properties["partition_key_path"].(map[string]any)
		if partitionConst, _ := partitionPathProp["const"].(string); partitionConst != "campaign_id" {
			t.Fatalf("schema %s has wrong partition_key_path const: %q", eventType, partitionConst)
		}
	}
}

func TestCampaignServiceEmittedEventEnvelopeContractConsistency(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	ctx := context.Background()

	created, err := module.Handler.CreateCampaignHandler(ctx, "brand-contract-1", "idem-campaign-contract-1", httptransport.CreateCampaignRequest{
		Title:            "Campaign Contract Flow",
		Description:      "Campaign contract flow description",
		Instructions:     "Campaign contract flow instructions",
		Niche:            "tech",
		AllowedPlatforms: []string{"youtube"},
		BudgetTotal:      100,
		RatePer1KViews:   1,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}

	upload, err := module.Handler.GenerateUploadURLHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "source.mp4",
		FileSize:    4 * 1024 * 1024,
		ContentType: "video/mp4",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/mp4",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch campaign failed: %v", err)
	}
	if err := module.Handler.PauseCampaignHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, "pause"); err != nil {
		t.Fatalf("pause campaign failed: %v", err)
	}
	if err := module.Handler.IncreaseBudgetHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, httptransport.IncreaseBudgetRequest{
		Amount: 20,
		Reason: "refill",
	}); err != nil {
		t.Fatalf("increase budget failed: %v", err)
	}
	if err := module.Handler.ResumeCampaignHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, "resume"); err != nil {
		t.Fatalf("resume campaign failed: %v", err)
	}
	if err := module.Handler.CompleteCampaignHandler(ctx, "brand-contract-1", created.Campaign.CampaignID, "done"); err != nil {
		t.Fatalf("complete campaign failed: %v", err)
	}

	outbox, err := module.Store.ListPendingOutbox(ctx, 200)
	if err != nil {
		t.Fatalf("list pending outbox failed: %v", err)
	}

	expectedEventTypes := map[string]bool{
		"campaign.created":        false,
		"campaign.launched":       false,
		"campaign.paused":         false,
		"campaign.resumed":        false,
		"campaign.completed":      false,
		"campaign.budget_updated": false,
	}

	for _, message := range outbox {
		var envelope map[string]any
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		eventType, _ := envelope["event_type"].(string)
		if _, tracked := expectedEventTypes[eventType]; tracked {
			expectedEventTypes[eventType] = true
		}
		if !strings.HasPrefix(eventType, "campaign.") {
			continue
		}

		if sourceService, _ := envelope["source_service"].(string); sourceService != "campaign-service" {
			t.Fatalf("campaign event has invalid source_service %q", sourceService)
		}
		if traceID, _ := envelope["trace_id"].(string); strings.TrimSpace(traceID) == "" {
			t.Fatalf("campaign event %s missing trace_id", eventType)
		}
		if partitionPath, _ := envelope["partition_key_path"].(string); partitionPath != "campaign_id" {
			t.Fatalf("campaign event %s has invalid partition_key_path %q", eventType, partitionPath)
		}
		partitionKey, _ := envelope["partition_key"].(string)
		if strings.TrimSpace(partitionKey) == "" {
			t.Fatalf("campaign event %s missing partition_key", eventType)
		}

		data, _ := envelope["data"].(map[string]any)
		dataCampaignID, _ := data["campaign_id"].(string)
		if dataCampaignID != partitionKey {
			t.Fatalf("campaign event %s partition mismatch: data.campaign_id=%q partition_key=%q", eventType, dataCampaignID, partitionKey)
		}
	}

	for eventType, seen := range expectedEventTypes {
		if !seen {
			t.Fatalf("expected emitted event type not found in outbox: %s", eventType)
		}
	}
}
