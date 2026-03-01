package memory

import (
	"context"
	"testing"
	"time"

	domainerrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	"solomon/contexts/community-experience/storefront-service/ports"
)

func TestPublishRequiresM60AndM61Projections(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	item, err := store.CreateStorefront(context.Background(), "creator_1", ports.CreateStorefrontInput{
		DisplayName: "Creator Shop",
		Category:    "Tech",
	}, now)
	if err != nil {
		t.Fatalf("create storefront failed: %v", err)
	}

	_, err = store.PublishStorefront(context.Background(), "creator_1", item.StorefrontID, now)
	if err == nil {
		t.Fatal("expected dependency unavailable before product projection sync")
	}
	if err != domainerrors.ErrDependencyUnavailable {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}

	_, err = store.ConsumeProductPublishedEvent(context.Background(), ports.ProductPublishedEvent{
		EventID:      "evt_prod_1",
		StorefrontID: item.StorefrontID,
		ProductID:    "prod_001",
		OccurredAt:   now,
	}, now)
	if err != nil {
		t.Fatalf("consume product event failed: %v", err)
	}
	published, err := store.PublishStorefront(context.Background(), "creator_1", item.StorefrontID, now)
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if published.Status != "published" {
		t.Fatalf("expected published, got %s", published.Status)
	}
}

func TestPublishRequiresSubscriptionProjectionPresence(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	item, err := store.CreateStorefront(context.Background(), "creator_2", ports.CreateStorefrontInput{
		DisplayName: "Creator Two Shop",
		Category:    "Tech",
	}, now)
	if err != nil {
		t.Fatalf("create storefront failed: %v", err)
	}
	_, err = store.ConsumeProductPublishedEvent(context.Background(), ports.ProductPublishedEvent{
		EventID:      "evt_prod_2",
		StorefrontID: item.StorefrontID,
		ProductID:    "prod_001",
		OccurredAt:   now,
	}, now)
	if err != nil {
		t.Fatalf("consume product event failed: %v", err)
	}

	_, err = store.PublishStorefront(context.Background(), "creator_2", item.StorefrontID, now)
	if err == nil {
		t.Fatal("expected dependency unavailable without M61 projection")
	}
	if err != domainerrors.ErrDependencyUnavailable {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}

	if err := store.UpsertSubscriptionProjection(context.Background(), ports.SubscriptionProjectionInput{
		UserID: "creator_2",
		Active: true,
	}, now); err != nil {
		t.Fatalf("upsert subscription failed: %v", err)
	}
	_, err = store.PublishStorefront(context.Background(), "creator_2", item.StorefrontID, now)
	if err != nil {
		t.Fatalf("publish failed after projection upsert: %v", err)
	}
}

func TestReportDedupWithin24Hours(t *testing.T) {
	store := NewStore()
	now := time.Now().UTC()

	item, err := store.CreateStorefront(context.Background(), "creator_1x", ports.CreateStorefrontInput{
		DisplayName: "Creator X Shop",
		Category:    "Tech",
	}, now)
	if err != nil {
		t.Fatalf("create storefront failed: %v", err)
	}
	if _, err := store.ReportStorefront(context.Background(), "viewer_1", item.StorefrontID, ports.ReportInput{
		Type:   "dmca",
		Reason: "copyright",
	}, now); err != nil {
		t.Fatalf("first report failed: %v", err)
	}
	if _, err := store.ReportStorefront(context.Background(), "viewer_1", item.StorefrontID, ports.ReportInput{
		Type:   "dmca",
		Reason: "copyright",
	}, now.Add(time.Hour)); err == nil {
		t.Fatal("expected dedup conflict")
	}
}
