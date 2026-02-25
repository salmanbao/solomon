package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	distributionmemory "solomon/contexts/campaign-editorial/distribution-service/adapters/memory"
	distributioncommands "solomon/contexts/campaign-editorial/distribution-service/application/commands"
	distributionworkers "solomon/contexts/campaign-editorial/distribution-service/application/workers"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

type stubSubscriber struct {
	topic   string
	group   string
	handler func(context.Context, ports.EventEnvelope) error
}

func (s *stubSubscriber) Subscribe(
	_ context.Context,
	topic string,
	group string,
	handler func(context.Context, ports.EventEnvelope) error,
) error {
	s.topic = topic
	s.group = group
	s.handler = handler
	return nil
}

func TestDistributionClaimedConsumerCreatesItem(t *testing.T) {
	now := time.Now().UTC()
	store := distributionmemory.NewStore([]entities.DistributionItem{
		{
			ID:             "seed-item",
			InfluencerID:   "seed-user",
			ClipID:         "clip-1",
			CampaignID:     "campaign-1",
			Status:         entities.DistributionStatusClaimed,
			ClaimedAt:      now.Add(-time.Hour),
			ClaimExpiresAt: now.Add(23 * time.Hour),
			UpdatedAt:      now.Add(-time.Hour),
		},
	})
	sub := &stubSubscriber{}
	consumer := distributionworkers.ClaimedConsumer{
		Subscriber:    sub,
		Repository:    store,
		Clock:         store,
		IDGen:         store,
		ConsumerGroup: "test-distribution-claims",
	}

	if err := consumer.Start(context.Background()); err != nil {
		t.Fatalf("start claimed consumer failed: %v", err)
	}
	if sub.handler == nil {
		t.Fatalf("expected subscriber handler to be registered")
	}

	payload, _ := json.Marshal(map[string]any{
		"claim_id":   "claim-123",
		"clip_id":    "clip-1",
		"user_id":    "influencer-1",
		"claim_type": "non_exclusive",
	})
	if err := sub.handler(context.Background(), ports.EventEnvelope{
		EventID:   "event-123",
		EventType: "distribution.claimed",
		Data:      payload,
	}); err != nil {
		t.Fatalf("claimed handler failed: %v", err)
	}

	item, err := store.GetItem(context.Background(), "claim-123")
	if err != nil {
		t.Fatalf("expected ingested item: %v", err)
	}
	if item.CampaignID != "campaign-1" {
		t.Fatalf("expected campaign-1, got %s", item.CampaignID)
	}
	if item.InfluencerID != "influencer-1" {
		t.Fatalf("expected influencer-1, got %s", item.InfluencerID)
	}
}

func TestDistributionSchedulerJobPublishesDueItems(t *testing.T) {
	now := time.Now().UTC()
	scheduledFor := now.Add(-2 * time.Minute)
	store := distributionmemory.NewStore([]entities.DistributionItem{
		{
			ID:              "item-1",
			InfluencerID:    "influencer-1",
			ClipID:          "clip-9",
			CampaignID:      "campaign-9",
			Status:          entities.DistributionStatusScheduled,
			ClaimedAt:       now.Add(-24 * time.Hour),
			ClaimExpiresAt:  now.Add(24 * time.Hour),
			ScheduledForUTC: &scheduledFor,
			Timezone:        "UTC",
			Platforms:       []string{"tiktok"},
			UpdatedAt:       now.Add(-3 * time.Minute),
		},
	})

	commands := distributioncommands.UseCase{
		Repository: store,
		Clock:      store,
		IDGen:      store,
		Outbox:     store,
	}
	job := distributionworkers.SchedulerJob{
		Commands:  commands,
		BatchSize: 100,
	}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("scheduler run failed: %v", err)
	}

	item, err := store.GetItem(context.Background(), "item-1")
	if err != nil {
		t.Fatalf("load updated item failed: %v", err)
	}
	if item.Status != entities.DistributionStatusPublished {
		t.Fatalf("expected published status, got %s", item.Status)
	}

	outbox, err := store.ListPendingOutbox(context.Background(), 20)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	if len(outbox) == 0 {
		t.Fatalf("expected outbox message for distribution.published")
	}

	found := false
	for _, message := range outbox {
		var envelope struct {
			EventType string `json:"event_type"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		if envelope.EventType == "distribution.published" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected distribution.published event")
	}
}

func TestDistributionSchedulerJobMarksFailedAndEmitsFailureEvent(t *testing.T) {
	now := time.Now().UTC()
	scheduledFor := now.Add(-2 * time.Minute)
	store := distributionmemory.NewStore([]entities.DistributionItem{
		{
			ID:              "item-failed-1",
			InfluencerID:    "influencer-1",
			ClipID:          "clip-9",
			CampaignID:      "campaign-9",
			Status:          entities.DistributionStatusScheduled,
			ClaimedAt:       now.Add(-24 * time.Hour),
			ClaimExpiresAt:  now.Add(24 * time.Hour),
			ScheduledForUTC: &scheduledFor,
			Timezone:        "UTC",
			Platforms:       []string{"invalid-platform"},
			UpdatedAt:       now.Add(-3 * time.Minute),
		},
	})

	commands := distributioncommands.UseCase{
		Repository: store,
		Clock:      store,
		IDGen:      store,
		Outbox:     store,
	}
	job := distributionworkers.SchedulerJob{
		Commands:  commands,
		BatchSize: 100,
	}
	if err := job.RunOnce(context.Background()); err == nil {
		t.Fatalf("expected scheduler run to fail for invalid platform")
	}

	item, err := store.GetItem(context.Background(), "item-failed-1")
	if err != nil {
		t.Fatalf("load failed item failed: %v", err)
	}
	if item.Status != entities.DistributionStatusFailed {
		t.Fatalf("expected failed status, got %s", item.Status)
	}
	if item.RetryCount != 1 {
		t.Fatalf("expected retry_count=1 after failure, got %d", item.RetryCount)
	}

	outbox, err := store.ListPendingOutbox(context.Background(), 20)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	found := false
	for _, message := range outbox {
		var envelope struct {
			EventType string `json:"event_type"`
			Data      struct {
				ClaimID string `json:"claim_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(message.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope failed: %v", err)
		}
		if envelope.EventType == "distribution.failed" {
			found = true
			if envelope.Data.ClaimID != "item-failed-1" {
				t.Fatalf("expected claim_id=item-failed-1, got %s", envelope.Data.ClaimID)
			}
		}
	}
	if !found {
		t.Fatalf("expected distribution.failed event")
	}
}
