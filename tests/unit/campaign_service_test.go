package unit

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestCampaignLaunchRequiresReadyMedia(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-3", "idem-campaign-3", httptransport.CreateCampaignRequest{
		Title:            "Launch Requires Media",
		Description:      "description for launch requirement",
		Instructions:     "instructions for launch requirement",
		Niche:            "fitness",
		AllowedPlatforms: []string{"tiktok"},
		BudgetTotal:      150,
		RatePer1KViews:   1.2,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-3", created.Campaign.CampaignID, "go live"); !errors.Is(err, domainerrors.ErrMissingReadyMedia) {
		t.Fatalf("expected missing ready media error, got %v", err)
	}
}

func TestCampaignMediaUploadAndLaunch(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-4", "idem-campaign-4", httptransport.CreateCampaignRequest{
		Title:            "Media And Launch",
		Description:      "description for media flow",
		Instructions:     "instructions for media flow",
		Niche:            "tech",
		AllowedPlatforms: []string{"youtube"},
		BudgetTotal:      200,
		RatePer1KViews:   1.0,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	upload, err := module.Handler.GenerateUploadURLHandler(context.Background(), "brand-4", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "source.mp4",
		FileSize:    10 * 1024 * 1024,
		ContentType: "video/mp4",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(context.Background(), "brand-4", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/mp4",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-4", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
}

func TestCampaignUpdateActiveAllowsDeadlineOnly(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-5", "idem-campaign-5", httptransport.CreateCampaignRequest{
		Title:            "Deadline Control",
		Description:      "description for deadline control",
		Instructions:     "instructions for deadline control",
		Niche:            "gaming",
		AllowedPlatforms: []string{"instagram"},
		BudgetTotal:      220,
		RatePer1KViews:   1.1,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	upload, err := module.Handler.GenerateUploadURLHandler(context.Background(), "brand-5", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "clip.mov",
		FileSize:    8 * 1024 * 1024,
		ContentType: "video/quicktime",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(context.Background(), "brand-5", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/quicktime",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-5", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch failed: %v", err)
	}

	tooSoon := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	if err := module.Handler.UpdateCampaignHandler(context.Background(), "brand-5", created.Campaign.CampaignID, httptransport.UpdateCampaignRequest{
		Deadline: &tooSoon,
	}); !errors.Is(err, domainerrors.ErrDeadlineTooSoon) {
		t.Fatalf("expected deadline too soon, got %v", err)
	}

	newTitle := "should fail"
	if err := module.Handler.UpdateCampaignHandler(context.Background(), "brand-5", created.Campaign.CampaignID, httptransport.UpdateCampaignRequest{
		Title: &newTitle,
	}); !errors.Is(err, domainerrors.ErrCampaignEditRestricted) {
		t.Fatalf("expected edit restricted, got %v", err)
	}
}

func TestCampaignIncreaseBudgetRequiresPaused(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-6", "idem-campaign-6", httptransport.CreateCampaignRequest{
		Title:            "Budget Rules",
		Description:      "description for budget rules",
		Instructions:     "instructions for budget rules",
		Niche:            "comedy",
		AllowedPlatforms: []string{"tiktok"},
		BudgetTotal:      180,
		RatePer1KViews:   0.9,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	upload, err := module.Handler.GenerateUploadURLHandler(context.Background(), "brand-6", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "clip.mp4",
		FileSize:    2 * 1024 * 1024,
		ContentType: "video/mp4",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(context.Background(), "brand-6", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/mp4",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-6", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch failed: %v", err)
	}

	err = module.Handler.IncreaseBudgetHandler(context.Background(), "brand-6", created.Campaign.CampaignID, httptransport.IncreaseBudgetRequest{
		Amount: 10,
		Reason: "active should fail",
	})
	if !errors.Is(err, domainerrors.ErrInvalidStateTransition) {
		t.Fatalf("expected invalid state transition, got %v", err)
	}
}

func TestCampaignSubmissionCreatedProjectionAutoPausesWhenBudgetExhausted(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-7", "idem-campaign-7", httptransport.CreateCampaignRequest{
		Title:            "Submission Projection",
		Description:      "description for submission projection",
		Instructions:     "instructions for submission projection",
		Niche:            "tech",
		AllowedPlatforms: []string{"youtube"},
		BudgetTotal:      10,
		RatePer1KViews:   5,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	upload, err := module.Handler.GenerateUploadURLHandler(context.Background(), "brand-7", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "clip.mp4",
		FileSize:    2 * 1024 * 1024,
		ContentType: "video/mp4",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(context.Background(), "brand-7", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/mp4",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-7", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch failed: %v", err)
	}

	now := time.Now().UTC()
	if _, err := module.Store.ApplySubmissionCreated(context.Background(), created.Campaign.CampaignID, "evt-sub-1", now); err != nil {
		t.Fatalf("first submission projection failed: %v", err)
	}
	projected, err := module.Store.ApplySubmissionCreated(context.Background(), created.Campaign.CampaignID, "evt-sub-2", now.Add(time.Second))
	if err != nil {
		t.Fatalf("second submission projection failed: %v", err)
	}
	if !projected.AutoPaused {
		t.Fatalf("expected campaign to auto pause on budget exhaustion")
	}
	if projected.NewStatus != "paused" {
		t.Fatalf("expected paused status, got %s", projected.NewStatus)
	}

	campaign, err := module.Store.GetCampaign(context.Background(), created.Campaign.CampaignID)
	if err != nil {
		t.Fatalf("get campaign failed: %v", err)
	}
	if campaign.Status != "paused" {
		t.Fatalf("expected paused campaign, got %s", campaign.Status)
	}

	outbox, err := module.Store.ListPendingOutbox(context.Background(), 100)
	if err != nil {
		t.Fatalf("list pending outbox failed: %v", err)
	}
	foundPaused := false
	for _, item := range outbox {
		if item.EventType == "campaign.paused" {
			foundPaused = true
			break
		}
	}
	if !foundPaused {
		t.Fatalf("expected campaign.paused event in outbox")
	}
}

func TestCampaignDeadlineCompleterTransitionsToCompleted(t *testing.T) {
	module := campaignservice.NewInMemoryModule(nil, nil)
	deadline := time.Now().UTC().Add(8 * 24 * time.Hour).Format(time.RFC3339)
	created, err := module.Handler.CreateCampaignHandler(context.Background(), "brand-8", "idem-campaign-8", httptransport.CreateCampaignRequest{
		Title:            "Deadline Auto Complete",
		Description:      "description for deadline auto complete",
		Instructions:     "instructions for deadline auto complete",
		Niche:            "fitness",
		AllowedPlatforms: []string{"instagram"},
		Deadline:         deadline,
		BudgetTotal:      120,
		RatePer1KViews:   1.2,
	})
	if err != nil {
		t.Fatalf("create campaign failed: %v", err)
	}
	upload, err := module.Handler.GenerateUploadURLHandler(context.Background(), "brand-8", created.Campaign.CampaignID, httptransport.GenerateUploadURLRequest{
		FileName:    "clip.mov",
		FileSize:    2 * 1024 * 1024,
		ContentType: "video/quicktime",
	})
	if err != nil {
		t.Fatalf("generate upload url failed: %v", err)
	}
	if err := module.Handler.ConfirmMediaHandler(context.Background(), "brand-8", created.Campaign.CampaignID, upload.MediaID, httptransport.ConfirmMediaRequest{
		AssetPath:   upload.AssetPath,
		ContentType: "video/quicktime",
	}); err != nil {
		t.Fatalf("confirm media failed: %v", err)
	}
	if err := module.Handler.LaunchCampaignHandler(context.Background(), "brand-8", created.Campaign.CampaignID, "launch"); err != nil {
		t.Fatalf("launch failed: %v", err)
	}

	campaign, err := module.Store.GetCampaign(context.Background(), created.Campaign.CampaignID)
	if err != nil {
		t.Fatalf("get campaign failed: %v", err)
	}
	past := time.Now().UTC().Add(-time.Hour)
	campaign.DeadlineAt = &past
	if err := module.Store.UpdateCampaign(context.Background(), campaign); err != nil {
		t.Fatalf("set past deadline failed: %v", err)
	}

	completed, err := module.Store.CompleteCampaignsPastDeadline(context.Background(), time.Now().UTC(), 10)
	if err != nil {
		t.Fatalf("deadline completion failed: %v", err)
	}
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed campaign, got %d", len(completed))
	}

	updated, err := module.Store.GetCampaign(context.Background(), created.Campaign.CampaignID)
	if err != nil {
		t.Fatalf("get updated campaign failed: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("expected completed campaign, got %s", updated.Status)
	}

	outbox, err := module.Store.ListPendingOutbox(context.Background(), 100)
	if err != nil {
		t.Fatalf("list pending outbox failed: %v", err)
	}
	foundCompleted := false
	for _, item := range outbox {
		if item.EventType == "campaign.completed" {
			foundCompleted = true
			break
		}
	}
	if !foundCompleted {
		t.Fatalf("expected campaign.completed event in outbox")
	}
}
