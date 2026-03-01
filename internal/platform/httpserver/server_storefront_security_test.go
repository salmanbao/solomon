package httpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorefrontCreateRequiresAuthorization(t *testing.T) {
	server := newTestServer()
	body := []byte(`{"display_name":"Creator Shop","category":"Tech"}`)
	req := httptest.NewRequest(http.MethodPost, "/storefronts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-stf-1")
	req.Header.Set("X-User-Id", "creator_1")
	req.Header.Set("Idempotency-Key", "idem-stf-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestStorefrontPublishRequiresDependencyProjections(t *testing.T) {
	server := newTestServer()

	createReq := httptest.NewRequest(http.MethodPost, "/storefronts", bytes.NewReader([]byte(`{"display_name":"Creator Shop","category":"Tech"}`)))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer token")
	createReq.Header.Set("X-Request-Id", "req-stf-2a")
	createReq.Header.Set("X-User-Id", "creator_2")
	createReq.Header.Set("Idempotency-Key", "idem-stf-2a")
	createRR := httptest.NewRecorder()
	server.mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createRR.Code, createRR.Body.String())
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/storefronts/storefront_m92_2/publish", nil)
	publishReq.Header.Set("Authorization", "Bearer token")
	publishReq.Header.Set("X-Request-Id", "req-stf-2b")
	publishReq.Header.Set("X-User-Id", "creator_2")
	publishReq.Header.Set("Idempotency-Key", "idem-stf-2b")
	publishRR := httptest.NewRecorder()
	server.mux.ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusFailedDependency {
		t.Fatalf("expected 424 publish without projections, got %d body=%s", publishRR.Code, publishRR.Body.String())
	}
}

func TestStorefrontPublishSuccessAfterM60M61ProjectionSync(t *testing.T) {
	server := newTestServer()

	createReq := httptest.NewRequest(http.MethodPost, "/storefronts", bytes.NewReader([]byte(`{"display_name":"Creator Shop","category":"Tech"}`)))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer token")
	createReq.Header.Set("X-Request-Id", "req-stf-3a")
	createReq.Header.Set("X-User-Id", "creator_3")
	createReq.Header.Set("Idempotency-Key", "idem-stf-3a")
	createRR := httptest.NewRecorder()
	server.mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201 create, got %d body=%s", createRR.Code, createRR.Body.String())
	}

	subReq := httptest.NewRequest(http.MethodPost, "/api/storefront/v1/internal/projections/subscriptions", bytes.NewReader([]byte(`{"user_id":"creator_3","active":true}`)))
	subReq.Header.Set("Content-Type", "application/json")
	subReq.Header.Set("Authorization", "Bearer token")
	subReq.Header.Set("X-Request-Id", "req-stf-3b")
	subRR := httptest.NewRecorder()
	server.mux.ServeHTTP(subRR, subReq)
	if subRR.Code != http.StatusAccepted {
		t.Fatalf("expected 202 subscription projection, got %d body=%s", subRR.Code, subRR.Body.String())
	}

	productReq := httptest.NewRequest(http.MethodPost, "/api/storefront/v1/internal/events/product-published", bytes.NewReader([]byte(`{"event_id":"evt-stf-3","storefront_id":"storefront_m92_2","product_id":"prod_001"}`)))
	productReq.Header.Set("Content-Type", "application/json")
	productReq.Header.Set("Authorization", "Bearer token")
	productReq.Header.Set("X-Request-Id", "req-stf-3c")
	productRR := httptest.NewRecorder()
	server.mux.ServeHTTP(productRR, productReq)
	if productRR.Code != http.StatusAccepted {
		t.Fatalf("expected 202 product projection, got %d body=%s", productRR.Code, productRR.Body.String())
	}

	publishReq := httptest.NewRequest(http.MethodPost, "/storefronts/storefront_m92_2/publish", nil)
	publishReq.Header.Set("Authorization", "Bearer token")
	publishReq.Header.Set("X-Request-Id", "req-stf-3d")
	publishReq.Header.Set("X-User-Id", "creator_3")
	publishReq.Header.Set("Idempotency-Key", "idem-stf-3d")
	publishRR := httptest.NewRecorder()
	server.mux.ServeHTTP(publishRR, publishReq)
	if publishRR.Code != http.StatusOK {
		t.Fatalf("expected 200 publish, got %d body=%s", publishRR.Code, publishRR.Body.String())
	}
}
