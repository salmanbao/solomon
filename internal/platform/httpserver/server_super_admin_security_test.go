package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuperAdminStartImpersonationRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"impersonated_user_id":"user-1","reason":"debug"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/impersonation/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-admin-1")
	req.Header.Set("X-MFA-Code", "123456")
	req.Header.Set("X-Admin-Id", "admin-1")
	req.Header.Set("Idempotency-Key", "idem-admin-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSuperAdminStartImpersonationRequiresMFA(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"impersonated_user_id":"user-1","reason":"debug"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/impersonation/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-admin-2")
	req.Header.Set("X-Admin-Id", "admin-1")
	req.Header.Set("Idempotency-Key", "idem-admin-2")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSuperAdminStartImpersonationRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"impersonated_user_id":"user-1","reason":"debug"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/impersonation/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-admin-3")
	req.Header.Set("X-MFA-Code", "123456")
	req.Header.Set("X-Admin-Id", "admin-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
