package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	"solomon/contexts/campaign-editorial/submission-service/adapters/memory"
	submissionworkers "solomon/contexts/campaign-editorial/submission-service/application/workers"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

func TestSubmissionServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "submission-service.openapi.json"))
	if err != nil {
		t.Fatalf("read submission-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode submission-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/submissions":                              {"post", "get"},
		"/submissions/{submission_id}":              {"get"},
		"/submissions/{submission_id}/approve":      {"post"},
		"/submissions/{submission_id}/reject":       {"post"},
		"/submissions/{submission_id}/report":       {"post"},
		"/submissions/bulk-operations":              {"post"},
		"/submissions/{submission_id}/analytics":    {"get"},
		"/dashboard/creator":                        {"get"},
		"/dashboard/brand":                          {"get"},
		"/v1/submissions":                           {"post", "get"},
		"/v1/submissions/{submission_id}":           {"get"},
		"/v1/submissions/{submission_id}/approve":   {"post"},
		"/v1/submissions/{submission_id}/reject":    {"post"},
		"/v1/submissions/{submission_id}/report":    {"post"},
		"/v1/submissions/bulk-operations":           {"post"},
		"/v1/submissions/{submission_id}/analytics": {"get"},
		"/v1/dashboard/creator":                     {"get"},
		"/v1/dashboard/brand":                       {"get"},
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

func TestSubmissionServiceEventSchemasCoverCanonicalEventSet(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	eventTypes := []string{
		"submission.created",
		"submission.approved",
		"submission.rejected",
		"submission.flagged",
		"submission.auto_approved",
		"submission.verified",
		"submission.view_locked",
		"submission.cancelled",
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
		if partitionConst, _ := partitionPathProp["const"].(string); partitionConst != "submission_id" {
			t.Fatalf("schema %s has wrong partition_key_path const: %q", eventType, partitionConst)
		}
	}
}

func TestSubmissionServiceEmittedEventEnvelopeContractConsistency(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore([]entities.Submission{
		{
			SubmissionID: "submission-auto-1",
			CampaignID:   "campaign-auto-1",
			CreatorID:    "creator-auto-1",
			Platform:     "tiktok",
			PostURL:      "https://tiktok.com/@creator/video/auto1",
			Status:       entities.SubmissionStatusPending,
			CreatedAt:    now.Add(-49 * time.Hour),
			UpdatedAt:    now.Add(-49 * time.Hour),
			CpvRate:      0.25,
		},
	})

	module := submissionservice.NewModule(submissionservice.Dependencies{
		Repository:     store,
		Idempotency:    store,
		Outbox:         store,
		Clock:          fixedClock{now: now},
		IDGen:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	})

	ctx := context.Background()
	create := func(idempotencyKey string, campaignID string, creatorID string, postURL string) string {
		resp, err := module.Handler.CreateSubmissionHandler(ctx, creatorID, httptransport.CreateSubmissionRequest{
			IdempotencyKey: idempotencyKey,
			CampaignID:     campaignID,
			Platform:       "tiktok",
			PostURL:        postURL,
		})
		if err != nil {
			t.Fatalf("create submission failed: %v", err)
		}
		return resp.Submission.SubmissionID
	}

	submissionApprove := create("idem-evt-approve", "campaign-evt-1", "creator-evt-1", "https://tiktok.com/@creator/video/evt1")
	if err := module.Handler.ApproveSubmissionHandler(ctx, "brand-reviewer-1", submissionApprove, httptransport.ApproveSubmissionRequest{
		IdempotencyKey: "idem-contract-approve-1",
		Reason:         "ok",
	}); err != nil {
		t.Fatalf("approve submission failed: %v", err)
	}

	submissionReject := create("idem-evt-reject", "campaign-evt-2", "creator-evt-2", "https://tiktok.com/@creator/video/evt2")
	if err := module.Handler.RejectSubmissionHandler(ctx, "brand-reviewer-2", submissionReject, httptransport.RejectSubmissionRequest{
		IdempotencyKey: "idem-contract-reject-1",
		Reason:         "invalid",
	}); err != nil {
		t.Fatalf("reject submission failed: %v", err)
	}

	submissionFlag := create("idem-evt-flag", "campaign-evt-3", "creator-evt-3", "https://tiktok.com/@creator/video/evt3")
	if err := module.Handler.ReportSubmissionHandler(ctx, "reporter-1", submissionFlag, httptransport.ReportSubmissionRequest{
		IdempotencyKey: "idem-contract-report-1",
		Reason:         "spam",
	}); err != nil {
		t.Fatalf("report submission failed: %v", err)
	}

	auto := submissionworkers.AutoApproveJob{
		Repository:  store,
		AutoApprove: store,
		Clock:       fixedClock{now: now},
		IDGen:       store,
		Outbox:      store,
		BatchSize:   100,
	}
	if err := auto.RunOnce(ctx); err != nil {
		t.Fatalf("auto-approve run failed: %v", err)
	}

	autoApproved, err := store.GetSubmission(ctx, "submission-auto-1")
	if err != nil {
		t.Fatalf("get auto-approved submission failed: %v", err)
	}
	autoApproved.ViewsCount = 2100
	windowEnd := now.Add(-time.Hour)
	autoApproved.VerificationWindowEnd = &windowEnd
	if err := store.UpdateSubmission(ctx, autoApproved); err != nil {
		t.Fatalf("prepare view-lock submission failed: %v", err)
	}

	viewLock := submissionworkers.ViewLockJob{
		Repository:      store,
		ViewLock:        store,
		Clock:           fixedClock{now: now},
		IDGen:           store,
		Outbox:          store,
		BatchSize:       100,
		PlatformFeeRate: 0.15,
	}
	if err := viewLock.RunOnce(ctx); err != nil {
		t.Fatalf("view-lock run failed: %v", err)
	}

	pendingOutbox, err := store.ListPendingOutbox(ctx, 500)
	if err != nil {
		t.Fatalf("list pending outbox failed: %v", err)
	}

	expectedEventTypes := map[string]bool{
		"submission.created":       false,
		"submission.approved":      false,
		"submission.rejected":      false,
		"submission.flagged":       false,
		"submission.auto_approved": false,
		"submission.verified":      false,
		"submission.view_locked":   false,
	}

	for _, message := range pendingOutbox {
		var envelope map[string]any
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}

		eventType, _ := envelope["event_type"].(string)
		if _, tracked := expectedEventTypes[eventType]; tracked {
			expectedEventTypes[eventType] = true
		}

		if !strings.HasPrefix(eventType, "submission.") {
			continue
		}

		eventID, _ := envelope["event_id"].(string)
		if strings.TrimSpace(eventID) == "" {
			t.Fatalf("submission event missing event_id: %#v", envelope)
		}
		if sourceService, _ := envelope["source_service"].(string); sourceService != "submission-service" {
			t.Fatalf("submission event has invalid source_service %q", sourceService)
		}
		if traceID, _ := envelope["trace_id"].(string); strings.TrimSpace(traceID) == "" {
			t.Fatalf("submission event %s missing trace_id", eventType)
		}
		if partitionPath, _ := envelope["partition_key_path"].(string); partitionPath != "submission_id" {
			t.Fatalf("submission event %s has invalid partition_key_path %q", eventType, partitionPath)
		}

		partitionKey, _ := envelope["partition_key"].(string)
		if strings.TrimSpace(partitionKey) == "" {
			t.Fatalf("submission event %s missing partition_key", eventType)
		}

		data, _ := envelope["data"].(map[string]any)
		dataSubmissionID, _ := data["submission_id"].(string)
		if dataSubmissionID != partitionKey {
			t.Fatalf("submission event %s partition mismatch: data.submission_id=%q partition_key=%q", eventType, dataSubmissionID, partitionKey)
		}
		if creatorID, _ := data["creator_id"].(string); strings.TrimSpace(creatorID) == "" {
			t.Fatalf("submission event %s missing data.creator_id", eventType)
		}
		if userID, _ := data["user_id"].(string); strings.TrimSpace(userID) == "" {
			t.Fatalf("submission event %s missing data.user_id compatibility field", eventType)
		}
	}

	for eventType, seen := range expectedEventTypes {
		if !seen {
			t.Fatalf("expected emitted event type not found in outbox: %s", eventType)
		}
	}
}

func containsAnyString(values []any, target string) bool {
	for _, item := range values {
		if value, ok := item.(string); ok && value == target {
			return true
		}
	}
	return false
}
