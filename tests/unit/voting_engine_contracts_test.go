package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
	httptransport "solomon/contexts/campaign-editorial/voting-engine/transport/http"
)

func TestVotingServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "voting-engine.openapi.json"))
	if err != nil {
		t.Fatalf("read voting-engine openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode voting-engine openapi: %v", err)
	}

	expected := map[string][]string{
		"/v1/votes":                               {"post"},
		"/v1/votes/{vote_id}":                     {"delete"},
		"/v1/votes/submissions/{submission_id}":   {"get"},
		"/v1/votes/leaderboard":                   {"get"},
		"/v1/leaderboards/campaign/{campaign_id}": {"get"},
		"/v1/leaderboards/round/{round_id}":       {"get"},
		"/v1/leaderboards/trending":               {"get"},
		"/v1/leaderboards/creator/{user_id}":      {"get"},
		"/v1/rounds/{round_id}/results":           {"get"},
		"/v1/analytics/votes":                     {"get"},
		"/v1/quarantine/{quarantine_id}/action":   {"post"},
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

func TestVotingServiceEventSchemasCoverCanonicalEventSet(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	eventTypes := []string{
		"vote.created",
		"vote.updated",
		"vote.retracted",
		"voting_round.closed",
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
		expectedPartitionPath := "submission_id"
		if eventType == "voting_round.closed" {
			expectedPartitionPath = "round_id"
		}
		if partitionConst, _ := partitionPathProp["const"].(string); partitionConst != expectedPartitionPath {
			t.Fatalf("schema %s has wrong partition_key_path const: %q", eventType, partitionConst)
		}
	}
}

func TestVotingServiceEmittedEventEnvelopeContractConsistency(t *testing.T) {
	now := time.Now().UTC()
	module := votingengine.NewInMemoryModule(nil, nil)
	module.Store.SetCampaign(ports.CampaignProjection{
		CampaignID: "campaign-contract-1",
		Status:     "active",
	})
	module.Store.SetSubmission(ports.SubmissionProjection{
		SubmissionID: "submission-contract-1",
		CampaignID:   "campaign-contract-1",
		CreatorID:    "creator-contract-1",
		Status:       "approved",
	})
	module.Store.SetReputationScore("voter-contract-1", 92)
	roundEnd := now.Add(4 * time.Hour)
	module.Store.SetRound(entities.VotingRound{
		RoundID:    "round-contract-1",
		CampaignID: "campaign-contract-1",
		Status:     entities.RoundStatusActive,
		StartsAt:   now.Add(-2 * time.Hour),
		EndsAt:     &roundEnd,
		CreatedAt:  now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-2 * time.Hour),
	})

	ctx := context.Background()
	createResp, err := module.Handler.CreateVoteHandler(ctx, "voter-contract-1", "idem-vote-create-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-contract-1",
		CampaignID:   "campaign-contract-1",
		RoundID:      "round-contract-1",
		VoteType:     "upvote",
	}, "127.0.0.1", "unit-test")
	if err != nil {
		t.Fatalf("create vote failed: %v", err)
	}

	if _, err := module.Handler.CreateVoteHandler(ctx, "voter-contract-1", "idem-vote-update-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-contract-1",
		CampaignID:   "campaign-contract-1",
		RoundID:      "round-contract-1",
		VoteType:     "downvote",
	}, "127.0.0.1", "unit-test"); err != nil {
		t.Fatalf("update vote failed: %v", err)
	}

	if err := module.Handler.RetractVoteHandler(ctx, createResp.VoteID, "voter-contract-1", "idem-vote-retract-1"); err != nil {
		t.Fatalf("retract vote failed: %v", err)
	}

	pendingOutbox, err := module.Store.ListPendingOutbox(ctx, 100)
	if err != nil {
		t.Fatalf("list pending outbox failed: %v", err)
	}

	expectedTypes := map[string]bool{
		"vote.created":   false,
		"vote.updated":   false,
		"vote.retracted": false,
	}

	for _, message := range pendingOutbox {
		var envelope map[string]any
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		eventType, _ := envelope["event_type"].(string)
		if _, tracked := expectedTypes[eventType]; tracked {
			expectedTypes[eventType] = true
		}
		if !strings.HasPrefix(eventType, "vote.") {
			continue
		}

		if sourceService, _ := envelope["source_service"].(string); sourceService != "voting-engine" {
			t.Fatalf("voting event has invalid source_service %q", sourceService)
		}
		if traceID, _ := envelope["trace_id"].(string); strings.TrimSpace(traceID) == "" {
			t.Fatalf("voting event %s missing trace_id", eventType)
		}
		if partitionPath, _ := envelope["partition_key_path"].(string); partitionPath != "submission_id" {
			t.Fatalf("voting event %s has invalid partition_key_path %q", eventType, partitionPath)
		}
		partitionKey, _ := envelope["partition_key"].(string)
		if strings.TrimSpace(partitionKey) == "" {
			t.Fatalf("voting event %s missing partition_key", eventType)
		}

		data, _ := envelope["data"].(map[string]any)
		dataSubmissionID, _ := data["submission_id"].(string)
		if dataSubmissionID != partitionKey {
			t.Fatalf("voting event %s partition mismatch: data.submission_id=%q partition_key=%q", eventType, dataSubmissionID, partitionKey)
		}
	}

	for eventType, seen := range expectedTypes {
		if !seen {
			t.Fatalf("expected emitted event type not found in outbox: %s", eventType)
		}
	}
}
