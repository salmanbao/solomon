package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnboardingFlowRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/onboarding/v1/flow", nil)
	req.Header.Set("X-Request-Id", "req-onb-1")
	req.Header.Set("X-User-Id", "user_onb_test_1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOnboardingCompleteStepRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"event_id":"evt-onb-2","user_id":"user_onb_test_2","role":"editor"}`)
	eventReq := httptest.NewRequest(http.MethodPost, "/api/onboarding/v1/internal/events/user-registered", bytes.NewReader(body))
	eventReq.Header.Set("Content-Type", "application/json")
	eventReq.Header.Set("Authorization", "Bearer token")
	eventReq.Header.Set("X-Request-Id", "req-onb-2a")
	eventRR := httptest.NewRecorder()
	server.mux.ServeHTTP(eventRR, eventReq)
	if eventRR.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", eventRR.Code, eventRR.Body.String())
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/onboarding/v1/steps/welcome/complete", bytes.NewReader([]byte(`{"metadata":{"device":"web"}}`)))
	completeReq.Header.Set("Content-Type", "application/json")
	completeReq.Header.Set("Authorization", "Bearer token")
	completeReq.Header.Set("X-Request-Id", "req-onb-2b")
	completeReq.Header.Set("X-User-Id", "user_onb_test_2")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, completeReq)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOnboardingRequiresUserRegisteredEventFirst(t *testing.T) {
	server := newTestServer()

	getReq := httptest.NewRequest(http.MethodGet, "/api/onboarding/v1/flow", nil)
	getReq.Header.Set("Authorization", "Bearer token")
	getReq.Header.Set("X-Request-Id", "req-onb-3a")
	getReq.Header.Set("X-User-Id", "user_onb_test_3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, getReq)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 before event ingest, got %d body=%s", rr.Code, rr.Body.String())
	}

	eventBody := []byte(`{"event_id":"evt-onb-3","user_id":"user_onb_test_3","role":"influencer"}`)
	eventReq := httptest.NewRequest(http.MethodPost, "/api/onboarding/v1/internal/events/user-registered", bytes.NewReader(eventBody))
	eventReq.Header.Set("Content-Type", "application/json")
	eventReq.Header.Set("Authorization", "Bearer token")
	eventReq.Header.Set("X-Request-Id", "req-onb-3b")
	eventRR := httptest.NewRecorder()
	server.mux.ServeHTTP(eventRR, eventReq)
	if eventRR.Code != http.StatusAccepted {
		t.Fatalf("expected 202 event ingest, got %d body=%s", eventRR.Code, eventRR.Body.String())
	}

	getReq2 := httptest.NewRequest(http.MethodGet, "/api/onboarding/v1/flow", nil)
	getReq2.Header.Set("Authorization", "Bearer token")
	getReq2.Header.Set("X-Request-Id", "req-onb-3c")
	getReq2.Header.Set("X-User-Id", "user_onb_test_3")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, getReq2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 after event ingest, got %d body=%s", rr2.Code, rr2.Body.String())
	}
}
