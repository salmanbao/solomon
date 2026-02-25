package unit

import (
	"context"
	"errors"
	"testing"

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/campaign-service/transport/http"
)

func TestCampaignCreateAndIdempotencyReplay(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)

	req := httptransport.CreateCampaignRequest{
		Title:            "Nike Campaign",
		Description:      "Campaign for short clips",
		Instructions:     "Use product in first 5 seconds",
		Niche:            "fitness",
		AllowedPlatforms: []string{"tiktok"},
		BudgetTotal:      100,
		RatePer1KViews:   1.25,
	}

	first, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-1", "idem-campaign-1", req)
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	second, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-1", "idem-campaign-1", req)
	if err != nil {
		t.Fatalf("replay campaign failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed response")
	}
	if first.Campaign.CampaignID != second.Campaign.CampaignID {
		t.Fatalf("expected same campaign id, got %s and %s", first.Campaign.CampaignID, second.Campaign.CampaignID)
	}
}

func TestCampaignInvalidStateTransition(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)

	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-2", "idem-campaign-2", httptransport.CreateCampaignRequest{
		Title:            "Pause Invalid",
		Description:      "description",
		Instructions:     "instructions",
		Niche:            "tech",
		AllowedPlatforms: []string{"youtube"},
		BudgetTotal:      80,
		RatePer1KViews:   0.8,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}

	err = module.Handler.PauseCampaignHandler(context.Background(), "brand-2", created.Campaign.CampaignID, "invalid")
	if !errors.Is(err, domainerrors.ErrInvalidStateTransition) {
		t.Fatalf("expected invalid state transition, got %v", err)
	}
}
