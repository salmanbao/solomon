package httpserver

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	distributionservice "solomon/contexts/campaign-editorial/distribution-service"
	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	authorization "solomon/contexts/identity-access/authorization-service"
)

func newTestServer() *Server {
	return New(
		contentlibrarymarketplace.NewInMemoryModule(nil, slog.Default()),
		authorization.NewInMemoryModule(slog.Default()),
		campaignservice.NewInMemoryModule(nil, slog.Default()),
		submissionservice.NewInMemoryModule(nil, slog.Default()),
		distributionservice.NewInMemoryModule(nil, slog.Default()),
		votingengine.NewInMemoryModule(nil, slog.Default()),
		slog.Default(),
		":0",
	)
}

func TestSubmissionCreateRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"campaign_id":"campaign-1","platform":"tiktok","post_url":"https://tiktok.com/@creator/video/1","idempotency_key":"body-key"}`)
	req := httptest.NewRequest(http.MethodPost, "/submissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("X-User-Id", "creator-1")
	req.Header.Set("Idempotency-Key", "hdr-key")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubmissionCreateRequiresIdempotencyHeader(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"campaign_id":"campaign-1","platform":"tiktok","post_url":"https://tiktok.com/@creator/video/1"}`)
	req := httptest.NewRequest(http.MethodPost, "/submissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("X-User-Id", "creator-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSubmissionGetRequiresUserHeader(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/submissions/submission-1", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}
