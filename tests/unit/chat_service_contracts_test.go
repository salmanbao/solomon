package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestChatServiceOpenAPIContractIncludesImplementedRoutes(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "chat-service.openapi.json"))
	if err != nil {
		t.Fatalf("read chat-service openapi: %v", err)
	}

	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode chat-service openapi: %v", err)
	}

	expected := map[string][]string{
		"/api/v1/chat/messages":                                          {"get", "post"},
		"/api/v1/chat/messages/{message_id}":                             {"put", "delete"},
		"/api/v1/chat/channels/{channel_id}/messages":                    {"get"},
		"/api/v1/chat/channels/{channel_id}/unread-count":                {"get"},
		"/api/v1/chat/search":                                            {"get"},
		"/api/v1/chat/poll":                                              {"get"},
		"/api/v1/chat/messages/subscribe":                                {"get"},
		"/api/v1/chat/messages/{message_id}/reactions":                   {"post"},
		"/api/v1/chat/messages/{message_id}/reactions/{emoji}":           {"delete"},
		"/api/v1/chat/messages/{message_id}/pin":                         {"post"},
		"/api/v1/chat/messages/{message_id}/report":                      {"post"},
		"/api/v1/chat/messages/{message_id}/attachments":                 {"post"},
		"/api/v1/chat/messages/{message_id}/attachments/{attachment_id}": {"get"},
		"/api/v1/chat/threads/{thread_id}/lock":                          {"put"},
		"/api/v1/chat/servers/{server_id}/moderators":                    {"put"},
		"/api/v1/chat/users/{user_id}/mute":                              {"post"},
		"/api/v1/chat/export":                                            {"post"},
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

func TestChatServiceOpenAPIContractRequiresIdempotencyOnMutations(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "contracts", "api", "v1", "chat-service.openapi.json"))
	if err != nil {
		t.Fatalf("read chat-service openapi: %v", err)
	}

	var doc struct {
		Paths      map[string]map[string]any `json:"paths"`
		Components struct {
			Parameters map[string]map[string]any `json:"parameters"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode chat-service openapi: %v", err)
	}

	mutating := map[string]string{
		"/api/v1/chat/messages":                                "post",
		"/api/v1/chat/messages/{message_id}":                   "put",
		"/api/v1/chat/messages/{message_id}/reactions":         "post",
		"/api/v1/chat/messages/{message_id}/reactions/{emoji}": "delete",
		"/api/v1/chat/messages/{message_id}/pin":               "post",
		"/api/v1/chat/messages/{message_id}/report":            "post",
		"/api/v1/chat/messages/{message_id}/attachments":       "post",
		"/api/v1/chat/threads/{thread_id}/lock":                "put",
		"/api/v1/chat/servers/{server_id}/moderators":          "put",
		"/api/v1/chat/users/{user_id}/mute":                    "post",
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
