package unit

import (
	"context"
	"errors"
	"testing"

	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

func TestSubmissionCreateApproveFlow(t *testing.T) {
	module := submissionservice.NewInMemoryModule(nil, nil)

	created, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-1", httptransport.CreateSubmissionRequest{
		CampaignID: "campaign-1",
		Platform:   "tiktok",
		PostURL:    "https://tiktok.com/@creator/video/1",
	})
	if err != nil {
		t.Fatalf("create submission failed: %v", err)
	}

	err = module.Handler.ApproveSubmissionHandler(context.Background(), "brand-1", created.Submission.SubmissionID, httptransport.ApproveSubmissionRequest{
		Reason: "looks good",
	})
	if err != nil {
		t.Fatalf("approve submission failed: %v", err)
	}

	fetched, err := module.Handler.GetSubmissionHandler(context.Background(), created.Submission.SubmissionID)
	if err != nil {
		t.Fatalf("get submission failed: %v", err)
	}
	if fetched.Submission.Status != "approved" {
		t.Fatalf("expected approved status, got %s", fetched.Submission.Status)
	}
}

func TestSubmissionDuplicateBlocked(t *testing.T) {
	module := submissionservice.NewInMemoryModule(nil, nil)
	req := httptransport.CreateSubmissionRequest{
		CampaignID: "campaign-dup",
		Platform:   "youtube",
		PostURL:    "https://youtube.com/watch?v=abc",
	}

	_, err := module.Handler.CreateSubmissionHandler(context.Background(), "creator-dup", req)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err = module.Handler.CreateSubmissionHandler(context.Background(), "creator-dup", req)
	if !errors.Is(err, domainerrors.ErrDuplicateSubmission) {
		t.Fatalf("expected duplicate submission error, got %v", err)
	}
}
