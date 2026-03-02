package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClippingCreateProjectRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/clipping/v1/projects", bytes.NewReader([]byte(`{
		"title":"clip one",
		"description":"desc",
		"source_url":"https://cdn.whop.dev/source.mp4",
		"source_type":"url"
	}`)))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-clip-1")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestClippingCreateProjectSuccess(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/clipping/v1/projects", bytes.NewReader([]byte(`{
		"title":"clip two",
		"description":"desc",
		"source_url":"https://cdn.whop.dev/source.mp4",
		"source_type":"url"
	}`)))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-clip-2")
	req.Header.Set("X-User-Id", "user-2")
	req.Header.Set("Idempotency-Key", "idem-clip-2")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
}
