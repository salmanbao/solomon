package unit

import (
	"context"
	"testing"
	"time"

	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
	httptransport "solomon/contexts/campaign-editorial/voting-engine/transport/http"
)

func TestVotingCreateReplayAndRetract(t *testing.T) {
	module := votingengine.NewInMemoryModule(nil, nil)
	module.Store.SetCampaign(ports.CampaignProjection{
		CampaignID: "campaign-1",
		Status:     "active",
	})
	module.Store.SetSubmission(ports.SubmissionProjection{
		SubmissionID: "submission-1",
		CampaignID:   "campaign-1",
		CreatorID:    "creator-9",
		Status:       "approved",
	})
	module.Store.SetReputationScore("user-1", 88)
	roundEnd := time.Now().UTC().Add(2 * time.Hour)
	module.Store.SetRound(entities.VotingRound{
		RoundID:    "round-1",
		CampaignID: "campaign-1",
		Status:     entities.RoundStatusActive,
		StartsAt:   time.Now().UTC().Add(-2 * time.Hour),
		EndsAt:     &roundEnd,
		CreatedAt:  time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt:  time.Now().UTC().Add(-2 * time.Hour),
	})

	first, err := module.Handler.CreateVoteHandler(context.Background(), "user-1", "idem-vote-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-1",
		CampaignID:   "campaign-1",
		RoundID:      "round-1",
		VoteType:     "upvote",
	}, "127.0.0.1", "unit-test")
	if err != nil {
		t.Fatalf("create vote failed: %v", err)
	}
	second, err := module.Handler.CreateVoteHandler(context.Background(), "user-1", "idem-vote-1", httptransport.CreateVoteRequest{
		SubmissionID: "submission-1",
		CampaignID:   "campaign-1",
		RoundID:      "round-1",
		VoteType:     "upvote",
	}, "127.0.0.1", "unit-test")
	if err != nil {
		t.Fatalf("replay vote failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed vote")
	}
	if first.VoteID != second.VoteID {
		t.Fatalf("expected same vote id, got %s and %s", first.VoteID, second.VoteID)
	}

	if err := module.Handler.RetractVoteHandler(context.Background(), first.VoteID, "user-1", "idem-retract-1"); err != nil {
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
