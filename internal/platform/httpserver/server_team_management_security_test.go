package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTeamCreateRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"name":"Brand Ops","org_id":"org_1","storefront_id":"store_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-team-1")
	req.Header.Set("X-User-Id", "user_owner_1")
	req.Header.Set("Idempotency-Key", "idem-team-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTeamCreateRequiresIdempotency(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"name":"Brand Ops","org_id":"org_1","storefront_id":"store_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-team-2")
	req.Header.Set("X-User-Id", "user_owner_1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestTeamCreateEnforcesM01Projection(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"name":"Brand Ops","org_id":"org_1","storefront_id":"store_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-team-3")
	req.Header.Set("X-User-Id", "user_unknown")
	req.Header.Set("Idempotency-Key", "idem-team-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusFailedDependency {
		t.Fatalf("expected 424, got %d body=%s", rr.Code, rr.Body.String())
	}
}
