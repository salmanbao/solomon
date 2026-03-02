package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInfluencerSummaryRequiresRequestID(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-User-Id", "creator-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestInfluencerCreateGoalRequiresIdempotency(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/goals", bytes.NewReader([]byte(`{
		"goal_type":"earnings",
		"goal_name":"Earn 500",
		"target_value":500
	}`)))
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-influencer-2")
	req.Header.Set("X-User-Id", "creator-1")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}
