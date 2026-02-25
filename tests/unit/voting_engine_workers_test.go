package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	votingmemory "solomon/contexts/campaign-editorial/voting-engine/adapters/memory"
	votingworkers "solomon/contexts/campaign-editorial/voting-engine/application/workers"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

type votingStubSubscriber struct {
	handlers map[string]func(context.Context, ports.EventEnvelope) error
}

func (s *votingStubSubscriber) Subscribe(
	_ context.Context,
	topic string,
	_ string,
	handler func(context.Context, ports.EventEnvelope) error,
) error {
	if s.handlers == nil {
		s.handlers = map[string]func(context.Context, ports.EventEnvelope) error{}
	}
	s.handlers[topic] = handler
	return nil
}

func TestVotingSubmissionRejectedConsumerRetractsVotes(t *testing.T) {
	now := time.Date(2026, 2, 25, 13, 0, 0, 0, time.UTC)
	store := votingmemory.NewStore([]entities.Vote{
		{
			VoteID:       "vote-1",
			SubmissionID: "submission-1",
			CampaignID:   "campaign-1",
			UserID:       "user-1",
			VoteType:     entities.VoteTypeUpvote,
			Weight:       1.5,
			Retracted:    false,
			CreatedAt:    now.Add(-time.Hour),
			UpdatedAt:    now.Add(-time.Hour),
		},
	})
	sub := &votingStubSubscriber{}
	consumer := votingworkers.SubmissionLifecycleConsumer{
		Subscriber: sub,
		Dedup:      store,
		Votes:      store,
		Outbox:     store,
		Clock:      fixedClock{now: now},
		IDGen:      store,
	}

	if err := consumer.Start(context.Background()); err != nil {
		t.Fatalf("start submission lifecycle consumer failed: %v", err)
	}
	handler := sub.handlers["submission.rejected"]
	if handler == nil {
		t.Fatalf("expected submission.rejected handler registration")
	}

	payload, _ := json.Marshal(map[string]any{
		"submission_id": "submission-1",
		"campaign_id":   "campaign-1",
	})
	if err := handler(context.Background(), ports.EventEnvelope{
		EventID:   "event-submission-rejected-1",
		EventType: "submission.rejected",
		Data:      payload,
	}); err != nil {
		t.Fatalf("submission.rejected handler failed: %v", err)
	}

	vote, err := store.GetVote(context.Background(), "vote-1")
	if err != nil {
		t.Fatalf("load vote after consumer failed: %v", err)
	}
	if !vote.Retracted {
		t.Fatalf("expected vote to be retracted by submission.rejected consumer")
	}

	outbox, err := store.ListPendingOutbox(context.Background(), 20)
	if err != nil {
		t.Fatalf("list voting outbox failed: %v", err)
	}
	foundRetracted := false
	for _, message := range outbox {
		var envelope struct {
			EventType string `json:"event_type"`
			Data      struct {
				Retracted bool `json:"retracted"`
			} `json:"data"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		if envelope.EventType == "vote.retracted" {
			foundRetracted = true
			if !envelope.Data.Retracted {
				t.Fatalf("expected vote.retracted payload to include retracted=true")
			}
		}
	}
	if !foundRetracted {
		t.Fatalf("expected vote.retracted event in outbox")
	}
}

func TestVotingCampaignCompletedConsumerClosesRoundAndEmitsEvent(t *testing.T) {
	now := time.Date(2026, 2, 25, 14, 0, 0, 0, time.UTC)
	store := votingmemory.NewStore(nil)
	roundEnd := now.Add(2 * time.Hour)
	store.SetRound(entities.VotingRound{
		RoundID:    "round-11",
		CampaignID: "campaign-11",
		Status:     entities.RoundStatusActive,
		StartsAt:   now.Add(-2 * time.Hour),
		EndsAt:     &roundEnd,
		CreatedAt:  now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-2 * time.Hour),
	})

	sub := &votingStubSubscriber{}
	consumer := votingworkers.CampaignStateConsumer{
		Subscriber: sub,
		Dedup:      store,
		Votes:      store,
		Outbox:     store,
		Clock:      fixedClock{now: now},
		IDGen:      store,
	}
	if err := consumer.Start(context.Background()); err != nil {
		t.Fatalf("start campaign state consumer failed: %v", err)
	}

	handler := sub.handlers["campaign.completed"]
	if handler == nil {
		t.Fatalf("expected campaign.completed handler registration")
	}
	payload, _ := json.Marshal(map[string]any{
		"campaign_id": "campaign-11",
	})
	if err := handler(context.Background(), ports.EventEnvelope{
		EventID:   "event-campaign-completed-1",
		EventType: "campaign.completed",
		Data:      payload,
	}); err != nil {
		t.Fatalf("campaign.completed handler failed: %v", err)
	}

	round, err := store.GetRound(context.Background(), "round-11")
	if err != nil {
		t.Fatalf("load round after consumer failed: %v", err)
	}
	if round.Status != entities.RoundStatusClosed {
		t.Fatalf("expected round status closed, got %s", round.Status)
	}

	outbox, err := store.ListPendingOutbox(context.Background(), 20)
	if err != nil {
		t.Fatalf("list voting outbox failed: %v", err)
	}
	foundClosed := false
	for _, message := range outbox {
		var envelope struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		if envelope.EventType == "voting_round.closed" {
			foundClosed = true
		}
	}
	if !foundClosed {
		t.Fatalf("expected voting_round.closed event in outbox")
	}
}
