package unit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	"solomon/contexts/campaign-editorial/submission-service/adapters/memory"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

func TestSubmissionApproveIdempotencyReplay(t *testing.T) {
	module := submissionservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-idem", httptransport.CreateSubmissionRequest{
		IdempotencyKey: "idem-create-review",
		CampaignID:     "campaign-idem",
		Platform:       "tiktok",
		PostURL:        "https://tiktok.com/@creator/video/idem-1",
	})
	if err != nil {
		t.Fatalf("create submission failed: %v", err)
	}

	err = module.Handler.ApproveSubmissionHandler(context.Background(), "brand-idem", created.Submission.SubmissionID, httptransport.ApproveSubmissionRequest{
		IdempotencyKey: "idem-approve-1",
		Reason:         "looks_good",
	})
	if err != nil {
		t.Fatalf("first approve failed: %v", err)
	}

	err = module.Handler.ApproveSubmissionHandler(context.Background(), "brand-idem", created.Submission.SubmissionID, httptransport.ApproveSubmissionRequest{
		IdempotencyKey: "idem-approve-1",
		Reason:         "looks_good",
	})
	if err != nil {
		t.Fatalf("expected replayed approve to succeed, got %v", err)
	}

	err = module.Handler.ApproveSubmissionHandler(context.Background(), "brand-idem", created.Submission.SubmissionID, httptransport.ApproveSubmissionRequest{
		IdempotencyKey: "idem-approve-1",
		Reason:         "different_reason",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyKeyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestSubmissionReportDuplicateByReporterRejected(t *testing.T) {
	module := submissionservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-report", httptransport.CreateSubmissionRequest{
		IdempotencyKey: "idem-create-report",
		CampaignID:     "campaign-report",
		Platform:       "tiktok",
		PostURL:        "https://tiktok.com/@creator/video/report-1",
	})
	if err != nil {
		t.Fatalf("create submission failed: %v", err)
	}

	err = module.Handler.ReportSubmissionHandler(context.Background(), "reporter-1", created.Submission.SubmissionID, httptransport.ReportSubmissionRequest{
		IdempotencyKey: "idem-report-1",
		Reason:         "spam",
		Description:    "first report",
	})
	if err != nil {
		t.Fatalf("first report failed: %v", err)
	}

	err = module.Handler.ReportSubmissionHandler(context.Background(), "reporter-1", created.Submission.SubmissionID, httptransport.ReportSubmissionRequest{
		IdempotencyKey: "idem-report-2",
		Reason:         "spam",
		Description:    "duplicate report",
	})
	if !errors.Is(err, domainerrors.ErrAlreadyReported) {
		t.Fatalf("expected already reported error, got %v", err)
	}
}

func TestSubmissionBulkOperationPartialFailureAndReplay(t *testing.T) {
	now := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
	store := memory.NewStore([]entities.Submission{
		{
			SubmissionID: "submission-bulk-1",
			CampaignID:   "campaign-bulk",
			CreatorID:    "creator-bulk-1",
			Platform:     "tiktok",
			PostURL:      "https://tiktok.com/@creator/video/bulk1",
			Status:       entities.SubmissionStatusPending,
			CreatedAt:    now.Add(-time.Hour),
			UpdatedAt:    now.Add(-time.Hour),
		},
		{
			SubmissionID: "submission-bulk-2",
			CampaignID:   "campaign-bulk",
			CreatorID:    "creator-bulk-2",
			Platform:     "tiktok",
			PostURL:      "https://tiktok.com/@creator/video/bulk2",
			Status:       entities.SubmissionStatusApproved,
			CreatedAt:    now.Add(-time.Hour),
			UpdatedAt:    now.Add(-time.Hour),
		},
	})
	module := submissionservice.NewModule(submissionservice.Dependencies{
		Repository:     store,
		Idempotency:    store,
		Outbox:         store,
		Clock:          fixedClock{now: now},
		IDGen:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	})

	resp, err := module.Handler.BulkOperationHandler(context.Background(), "brand-bulk", httptransport.BulkOperationRequest{
		IdempotencyKey: "idem-bulk-1",
		OperationType:  "bulk_approve",
		SubmissionIDs:  []string{"submission-bulk-1", "submission-bulk-2"},
		ReasonCode:     "bulk_approve_48h",
	})
	if err != nil {
		t.Fatalf("bulk operation failed: %v", err)
	}
	if resp.Processed != 2 || resp.SucceededCount != 1 || resp.FailedCount != 1 {
		t.Fatalf("unexpected bulk counts: %+v", resp)
	}

	replayed, err := module.Handler.BulkOperationHandler(context.Background(), "brand-bulk", httptransport.BulkOperationRequest{
		IdempotencyKey: "idem-bulk-1",
		OperationType:  "bulk_approve",
		SubmissionIDs:  []string{"submission-bulk-1", "submission-bulk-2"},
		ReasonCode:     "bulk_approve_48h",
	})
	if err != nil {
		t.Fatalf("replayed bulk operation failed: %v", err)
	}
	if replayed != resp {
		t.Fatalf("expected replayed response to match first response: first=%+v replayed=%+v", resp, replayed)
	}
}

func TestSubmissionRejectWithCancellationReasonEmitsCancelledEvent(t *testing.T) {
	module := submissionservice.NewInMemoryModule(nil, nil)
	created, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-cancel", httptransport.CreateSubmissionRequest{
		IdempotencyKey: "idem-create-cancel",
		CampaignID:     "campaign-cancel",
		Platform:       "tiktok",
		PostURL:        "https://tiktok.com/@creator/video/cancel-1",
	})
	if err != nil {
		t.Fatalf("create submission failed: %v", err)
	}

	if err := module.Handler.RejectSubmissionHandler(context.Background(), "brand-cancel", created.Submission.SubmissionID, httptransport.RejectSubmissionRequest{
		IdempotencyKey: "idem-cancel-1",
		Reason:         "campaign_cancelled",
		Notes:          "campaign was stopped",
	}); err != nil {
		t.Fatalf("cancel via reject path failed: %v", err)
	}

	fetched, err := module.Handler.GetSubmissionHandler(context.Background(), "creator-cancel", created.Submission.SubmissionID)
	if err != nil {
		t.Fatalf("get submission failed: %v", err)
	}
	if fetched.Submission.Status != string(entities.SubmissionStatusCancelled) {
		t.Fatalf("expected cancelled status, got %s", fetched.Submission.Status)
	}

	outbox, err := module.Store.ListPendingOutbox(context.Background(), 20)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	foundCancelled := false
	for _, msg := range outbox {
		var envelope struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox payload failed: %v", err)
		}
		if envelope.EventType == "submission.cancelled" {
			foundCancelled = true
		}
	}
	if !foundCancelled {
		t.Fatalf("expected submission.cancelled event in outbox")
	}
}
