package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverBrowseRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/discover/v1/campaigns/browse?page_size=10", nil)
	req.Header.Set("X-Request-Id", "req-discover-1")
	req.Header.Set("X-User-Id", "user-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDiscoverBookmarkRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/discover/v1/campaigns/c-12345678-9abc-def0-1234-56789abcdef0/bookmark",
		bytes.NewReader([]byte(`{"tag":"watch","note":"later"}`)),
	)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-discover-2")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
