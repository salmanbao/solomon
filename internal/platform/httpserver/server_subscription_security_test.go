package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSubscriptionCreateRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"plan_id":"plan_pro_monthly","trial":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-sub-1")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Idempotency-Key", "idem-sub-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubscriptionCreateRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"plan_id":"plan_pro_monthly","trial":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-sub-2")
	req.Header.Set("X-User-Id", "user-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubscriptionCreateSuccess(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"plan_id":"plan_pro_monthly","trial":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-sub-3")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("Idempotency-Key", "idem-sub-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
}
