package unit

import (
	"context"
	"errors"
	"testing"

	productservice "solomon/contexts/community-experience/product-service"
	domainerrors "solomon/contexts/community-experience/product-service/domain/errors"
	httptransport "solomon/contexts/community-experience/product-service/transport/http"
)

func TestProductServiceCreateProductIdempotency(t *testing.T) {
	module := productservice.NewInMemoryModule(nil)
	ctx := context.Background()

	first, err := module.Handler.CreateProductHandler(ctx, "creator_900", "idem-product-create-1", httptransport.CreateProductRequest{
		Name:         "Creator Product",
		Description:  "description",
		ProductType:  "digital",
		PricingModel: "one_time",
		PriceCents:   1500,
		Visibility:   "public",
	})
	if err != nil {
		t.Fatalf("first create product failed: %v", err)
	}
	second, err := module.Handler.CreateProductHandler(ctx, "creator_900", "idem-product-create-1", httptransport.CreateProductRequest{
		Name:         "Creator Product",
		Description:  "description",
		ProductType:  "digital",
		PricingModel: "one_time",
		PriceCents:   1500,
		Visibility:   "public",
	})
	if err != nil {
		t.Fatalf("replayed create product failed: %v", err)
	}
	if first.Data.ProductID != second.Data.ProductID {
		t.Fatalf("expected idempotent replay to return same product id, got %s and %s", first.Data.ProductID, second.Data.ProductID)
	}

	_, err = module.Handler.CreateProductHandler(ctx, "creator_900", "idem-product-create-1", httptransport.CreateProductRequest{
		Name:         "Different",
		Description:  "description",
		ProductType:  "digital",
		PricingModel: "one_time",
		PriceCents:   1500,
		Visibility:   "public",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestProductServiceAccessAndPurchaseFlow(t *testing.T) {
	module := productservice.NewInMemoryModule(nil)
	ctx := context.Background()

	check, err := module.Handler.CheckAccessHandler(ctx, "user_404", "prod_001")
	if err != nil {
		t.Fatalf("check access failed: %v", err)
	}
	if check.Data.HasAccess {
		t.Fatalf("expected no access before purchase")
	}

	if _, err := module.Handler.PurchaseProductHandler(ctx, "user_404", "idem-product-purchase-1", "prod_001"); err != nil {
		t.Fatalf("purchase failed: %v", err)
	}
	checkAfter, err := module.Handler.CheckAccessHandler(ctx, "user_404", "prod_001")
	if err != nil {
		t.Fatalf("check access after purchase failed: %v", err)
	}
	if !checkAfter.Data.HasAccess {
		t.Fatalf("expected access after purchase")
	}
}

func TestProductServiceInventoryAdjustment(t *testing.T) {
	module := productservice.NewInMemoryModule(nil)
	ctx := context.Background()

	resp, err := module.Handler.AdjustInventoryHandler(ctx, "admin_1", "idem-product-inv-1", "prod_002", httptransport.AdjustInventoryRequest{
		NewCount: 99,
		Reason:   "restock",
	})
	if err != nil {
		t.Fatalf("adjust inventory failed: %v", err)
	}
	if resp.NewCount != 99 {
		t.Fatalf("expected new count 99, got %d", resp.NewCount)
	}
}
