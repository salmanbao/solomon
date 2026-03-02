package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModerationApproveRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/moderation/approve", bytes.NewReader([]byte(`{
		"submission_id":"sub-1",
		"campaign_id":"camp-1",
		"reason":"manual_review_pass"
	}`)))
	req.Header.Set("X-Request-Id", "req-mod-1")
	req.Header.Set("X-User-Id", "mod-1")
	req.Header.Set("Idempotency-Key", "mod-approve-1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestModerationRejectRequiresIdempotency(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/moderation/reject", bytes.NewReader([]byte(`{
		"submission_id":"sub-1",
		"campaign_id":"camp-1",
		"rejection_reason":"wrong_platform"
	}`)))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-mod-2")
	req.Header.Set("X-User-Id", "mod-1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
