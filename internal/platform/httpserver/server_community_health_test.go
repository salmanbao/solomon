package httpserver

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func signCommunityHealthBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestCommunityHealthWebhookRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	t.Setenv("COMMUNITY_HEALTH_WEBHOOK_SECRET", "test-secret")

	body := []byte(`{"event_id":"evt-1","event_type":"chat.message.created","message_id":"msg-1","server_id":"server_123","channel_id":"channel-1","user_id":"user-1","content":"hello world","created_at":"2026-02-05T10:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/chat/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signCommunityHealthBody("test-secret", body))

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommunityHealthWebhookRejectsInvalidSignature(t *testing.T) {
	server := newTestServer()
	t.Setenv("COMMUNITY_HEALTH_WEBHOOK_SECRET", "test-secret")

	body := []byte(`{"event_id":"evt-1","event_type":"chat.message.created","message_id":"msg-1","server_id":"server_123","channel_id":"channel-1","user_id":"user-1","content":"hello world","created_at":"2026-02-05T10:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/chat/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("X-Webhook-Signature", "sha256=deadbeef")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommunityHealthWebhookAcceptsValidSignature(t *testing.T) {
	server := newTestServer()
	t.Setenv("COMMUNITY_HEALTH_WEBHOOK_SECRET", "test-secret")

	body := []byte(`{"event_id":"evt-1","event_type":"chat.message.created","message_id":"msg-1","server_id":"server_123","channel_id":"channel-1","user_id":"user-1","content":"hello world","created_at":"2026-02-05T10:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/chat/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("X-Webhook-Signature", signCommunityHealthBody("test-secret", body))

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommunityHealthGetScoreRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/community-health/server_123/health-score", nil)
	req.Header.Set("X-Request-Id", "req-chs-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommunityHealthGetScoreSuccess(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/community-health/server_123/health-score", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-chs-2")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCommunityHealthWebhookRejectsInvalidTimestamp(t *testing.T) {
	server := newTestServer()
	t.Setenv("COMMUNITY_HEALTH_WEBHOOK_SECRET", "test-secret")

	body := []byte(`{"event_id":"evt-1","event_type":"chat.message.created","message_id":"msg-1","server_id":"server_123","channel_id":"channel-1","user_id":"user-1","content":"hello world","created_at":"not-a-time"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/chat/message", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1")
	req.Header.Set("X-Webhook-Signature", signCommunityHealthBody("test-secret", body))

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
