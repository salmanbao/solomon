package unit

import (
	"context"
	"testing"

	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	httptransport "solomon/contexts/campaign-editorial/voting-engine/transport/http"
)

func TestVotingCreateReplayAndRetract(t *testing.T) {
	module := votingengine.NewInMemoryModule(nil, nil)

	first, err := module.Handler.CreateVoteHandler(context.Background(), "user-1", "idem-vote-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-1",
		CampaignID:   "campaign-1",
		VoteType:     "upvote",
	})
	if err != nil {
		t.Fatalf("create vote failed: %v", err)
	}
	second, err := module.Handler.CreateVoteHandler(context.Background(), "user-1", "idem-vote-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-1",
		CampaignID:   "campaign-1",
		VoteType:     "upvote",
	})
	if err != nil {
		t.Fatalf("replay vote failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed vote")
	}
	if first.VoteID != second.VoteID {
		t.Fatalf("expected same vote id, got %s and %s", first.VoteID, second.VoteID)
	}

	if err := module.Handler.RetractVoteHandler(context.Background(), first.VoteID, "user-1"); err != nil {
		t.Fatalf("retract vote failed: %v", err)
	}
	score, err := module.Handler.SubmissionVotesHandler(context.Background(), "submission-1")
	if err != nil {
		t.Fatalf("submission votes failed: %v", err)
	}
	if score.Weighted != 0 {
		t.Fatalf("expected weighted score 0 after retraction, got %f", score.Weighted)
	}
}
