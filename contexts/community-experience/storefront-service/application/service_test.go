package application

import (
	"context"
	"testing"
	"time"

	"solomon/contexts/community-experience/storefront-service/adapters/memory"
	domainerrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	"solomon/contexts/community-experience/storefront-service/ports"
)

func TestCreateStorefrontIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	input := ports.CreateStorefrontInput{DisplayName: "Creator Shop", Category: "Tech"}
	first, err := service.CreateStorefront(context.Background(), "idem-sf-1", "creator_1", input)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	second, err := service.CreateStorefront(context.Background(), "idem-sf-1", "creator_1", input)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	if first.StorefrontID != second.StorefrontID {
		t.Fatalf("expected same storefront id, got %s vs %s", first.StorefrontID, second.StorefrontID)
	}
}

func TestCreateStorefrontIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}

	_, err := service.CreateStorefront(context.Background(), "idem-sf-2", "creator_1", ports.CreateStorefrontInput{
		DisplayName: "Creator Shop",
		Category:    "Tech",
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = service.CreateStorefront(context.Background(), "idem-sf-2", "creator_1", ports.CreateStorefrontInput{
		DisplayName: "Another Shop",
		Category:    "Tech",
	})
	if err == nil {
		t.Fatal("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}
