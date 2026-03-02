package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEditorFeedRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/editor/feed?limit=10", nil)
	req.Header.Set("X-Request-Id", "req-editor-1")
	req.Header.Set("X-User-Id", "editor-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEditorSaveRequiresIdempotency(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/editor/campaigns/camp-1/save", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-editor-2")
	req.Header.Set("X-User-Id", "editor-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
