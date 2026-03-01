package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProductCreateRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"name":"p","description":"d","product_type":"digital","pricing_model":"one_time","price_cents":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-product-1")
	req.Header.Set("X-User-Id", "creator-1")
	req.Header.Set("Idempotency-Key", "idem-product-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProductCreateRequiresIdempotencyKey(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"name":"p","description":"d","product_type":"digital","pricing_model":"one_time","price_cents":100}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-product-2")
	req.Header.Set("X-User-Id", "creator-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestProductListRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?page=1&limit=20", nil)
	req.Header.Set("X-Request-Id", "req-product-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}
