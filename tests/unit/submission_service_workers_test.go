package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	"solomon/contexts/campaign-editorial/submission-service/adapters/memory"
	submissionworkers "solomon/contexts/campaign-editorial/submission-service/application/workers"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now.UTC()
}

func TestSubmissionCreateCampaignValidation(t *testing.T) {
	store := memory.NewStore(nil)
	store.SetCampaign("campaign-guard", "paused", []string{"tiktok"}, 0.3)

	module := submissionservice.NewModule(submissionservice.Dependencies{
		Repository:     store,
		Campaigns:      store,
		Idempotency:    store,
		Outbox:         store,
		Clock:          store,
		IDGen:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	})

	_, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-guard", httptransport.CreateSubmissionRequest{
		IdempotencyKey: "idem-guard-1",
		CampaignID:     "campaign-guard",
		Platform:       "tiktok",
		PostURL:        "https://tiktok.com/@creator/video/123",
	})
	if err != domainerrors.ErrCampaignNotActive {
		t.Fatalf("expected campaign not active, got %v", err)
	}

	store.SetCampaign("campaign-guard", "active", []string{"youtube"}, 0.3)
	_, err = module.Handler.CreateSubmissionHandler(context.Background(), "creator-guard", httptransport.CreateSubmissionRequest{
		IdempotencyKey: "idem-guard-2",
		CampaignID:     "campaign-guard",
		Platform:       "tiktok",
		PostURL:        "https://tiktok.com/@creator/video/123",
	})
	if err != domainerrors.ErrPlatformNotAllowed {
		t.Fatalf("expected platform not allowed, got %v", err)
	}
}

func TestSubmissionWorkerAutoApproveAndViewLock(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore([]entities.Submission{
		{
			SubmissionID: "submission-1",
			CampaignID:   "campaign-1",
			CreatorID:    "creator-1",
			Platform:     "tiktok",
			PostURL:      "https://tiktok.com/@creator/video/1",
			Status:       entities.SubmissionStatusPending,
			CreatedAt:    now.Add(-49 * time.Hour),
			UpdatedAt:    now.Add(-49 * time.Hour),
			CpvRate:      0.2,
		},
	})

	auto := submissionworkers.AutoApproveJob{
		Repository:  store,
		AutoApprove: store,
		Clock:       fixedClock{now: now},
		IDGen:       store,
		Outbox:      store,
		BatchSize:   100,
	}
	if err := auto.RunOnce(context.Background()); err != nil {
		t.Fatalf("auto approve run failed: %v", err)
	}

	autoApproved, err := store.GetSubmission(context.Background(), "submission-1")
	if err != nil {
		t.Fatalf("get after auto approve failed: %v", err)
	}
	if autoApproved.Status != entities.SubmissionStatusApproved {
		t.Fatalf("expected approved status, got %s", autoApproved.Status)
	}
	if autoApproved.ApprovalReason != "auto_approve_48h" {
		t.Fatalf("expected auto approval reason, got %s", autoApproved.ApprovalReason)
	}

	autoApproved.ViewsCount = 5000
	windowEnd := now.Add(-time.Hour)
	autoApproved.VerificationWindowEnd = &windowEnd
	if err := store.UpdateSubmission(context.Background(), autoApproved); err != nil {
		t.Fatalf("seed view lock failed: %v", err)
	}

	viewLock := submissionworkers.ViewLockJob{
		Repository:      store,
		ViewLock:        store,
		Clock:           fixedClock{now: now},
		IDGen:           store,
		Outbox:          store,
		BatchSize:       100,
		PlatformFeeRate: 0.15,
	}
	if err := viewLock.RunOnce(context.Background()); err != nil {
		t.Fatalf("view lock run failed: %v", err)
	}

	locked, err := store.GetSubmission(context.Background(), "submission-1")
	if err != nil {
		t.Fatalf("get after view lock failed: %v", err)
	}
	if locked.Status != entities.SubmissionStatusViewLocked {
		t.Fatalf("expected view_locked status, got %s", locked.Status)
	}
	if locked.LockedViews == nil || *locked.LockedViews != 5000 {
		t.Fatalf("expected locked views to be 5000")
	}
	if locked.GrossAmount <= 0 || locked.NetAmount <= 0 {
		t.Fatalf("expected gross/net amounts to be calculated")
	}

	pendingOutbox, err := store.ListPendingOutbox(context.Background(), 50)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	types := map[string]bool{}
	for _, message := range pendingOutbox {
		var envelope struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		types[envelope.EventType] = true
	}
	if !types["submission.auto_approved"] {
		t.Fatalf("expected submission.auto_approved event")
	}
	if !types["submission.verified"] {
		t.Fatalf("expected submission.verified event")
	}
	if !types["submission.view_locked"] {
		t.Fatalf("expected submission.view_locked event")
	}
}

func TestSubmissionWorkerJobsCanBeDisabledByFeatureFlags(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore([]entities.Submission{
		{
			SubmissionID: "submission-disabled",
			CampaignID:   "campaign-disabled",
			CreatorID:    "creator-disabled",
			Platform:     "tiktok",
			PostURL:      "https://tiktok.com/@creator/video/disabled",
			Status:       entities.SubmissionStatusPending,
			CreatedAt:    now.Add(-72 * time.Hour),
			UpdatedAt:    now.Add(-72 * time.Hour),
			CpvRate:      0.2,
		},
	})

	auto := submissionworkers.AutoApproveJob{
		Repository:  store,
		AutoApprove: store,
		Clock:       fixedClock{now: now},
		IDGen:       store,
		Outbox:      store,
		BatchSize:   100,
		Disabled:    true,
	}
	if err := auto.RunOnce(context.Background()); err != nil {
		t.Fatalf("auto approve run failed: %v", err)
	}

	current, err := store.GetSubmission(context.Background(), "submission-disabled")
	if err != nil {
		t.Fatalf("get submission failed: %v", err)
	}
	if current.Status != entities.SubmissionStatusPending {
		t.Fatalf("expected pending status when auto-approve disabled, got %s", current.Status)
	}
}
