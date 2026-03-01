package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReputationGetUserRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/user/user_123", nil)
	req.Header.Set("X-Request-Id", "req-reputation-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReputationGetUserRequiresRequestID(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/user/user_123", nil)
	req.Header.Set("Authorization", "Bearer token")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReputationGetUserEnforcesDependencyProjection(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/user/user_unknown", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-reputation-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFailedDependency {
		t.Fatalf("expected 424, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReputationLeaderboardRejectsInvalidTier(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/leaderboard?tier=diamond", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-reputation-4")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReputationGetUserReturnsCanonicalResponse(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/user/user_123", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-reputation-5")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["status"] != "success" {
		t.Fatalf("expected success status, got %#v", payload["status"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object data payload, got %#v", payload["data"])
	}
	if data["user_id"] != "user_123" {
		t.Fatalf("expected user_123, got %#v", data["user_id"])
	}
	if data["tier"] != "gold" {
		t.Fatalf("expected gold tier, got %#v", data["tier"])
	}
}

func TestReputationLeaderboardReturnsCanonicalResponse(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reputation/leaderboard?tier=gold&limit=1&offset=0", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-reputation-6")
	req.Header.Set("X-User-Id", "user_123")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["status"] != "success" {
		t.Fatalf("expected success status, got %#v", payload["status"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected object data payload, got %#v", payload["data"])
	}
	if data["total_creators"] == nil {
		t.Fatalf("expected total_creators in response data")
	}
}
