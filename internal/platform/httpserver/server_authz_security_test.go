package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthzCheckRequiresAuthorizationHeader(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/authz/v1/check", bytes.NewReader([]byte(`{"permission":"campaign.view"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-authz-1")
	req.Header.Set("X-User-Id", "user-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthzCheckRequiresRequestIDHeader(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/authz/v1/check", bytes.NewReader([]byte(`{"permission":"campaign.view"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-User-Id", "user-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAuthzGrantRequiresRequestIDHeader(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/authz/v1/users/user-1/roles/grant",
		bytes.NewReader([]byte(`{"role_id":"editor"}`)),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-User-Id", "admin-1")
	req.Header.Set("Idempotency-Key", "authz-grant-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
