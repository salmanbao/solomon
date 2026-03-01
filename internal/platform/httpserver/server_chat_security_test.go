package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatPostMessageRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"server_id":"srv_001","channel_id":"ch_001","content":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-chat-1")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Idempotency-Key", "idem-chat-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestChatPostMessageRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"server_id":"srv_001","channel_id":"ch_001","content":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-chat-2")
	req.Header.Set("X-User-Id", "user-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestChatListMessagesRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/channels/ch_001/messages?limit=20", nil)
	req.Header.Set("X-Request-Id", "req-chat-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}
