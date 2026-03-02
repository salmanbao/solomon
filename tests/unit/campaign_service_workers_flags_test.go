package unit

import (
	"context"
	"testing"
	"time"

	campaignworkers "solomon/contexts/campaign-editorial/campaign-service/application/workers"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type campaignStubSubscriber struct {
	topics []string
}

func (s *campaignStubSubscriber) Subscribe(
	_ context.Context,
	topic string,
	_ string,
	_ func(context.Context, ports.EventEnvelope) error,
) error {
	s.topics = append(s.topics, topic)
	return nil
}

type campaignStubDeadlineRepo struct {
	calls int
}

func (r *campaignStubDeadlineRepo) CompleteCampaignsPastDeadline(
	_ context.Context,
	_ time.Time,
	_ int,
) ([]ports.DeadlineCompletionResult, error) {
	r.calls++
	return nil, nil
}

func TestCampaignSubmissionCreatedConsumerCanBeDisabledByFeatureFlag(t *testing.T) {
	subscriber := &campaignStubSubscriber{}
	consumer := campaignworkers.SubmissionCreatedConsumer{
		Subscriber: subscriber,
		Disabled:   true,
	}

	if err := consumer.Start(context.Background()); err != nil {
		t.Fatalf("start disabled consumer: %v", err)
	}
	if len(subscriber.topics) != 0 {
		t.Fatalf("expected no topic subscriptions when consumer disabled")
	}
}

func TestCampaignDeadlineCompleterCanBeDisabledByFeatureFlag(t *testing.T) {
	repo := &campaignStubDeadlineRepo{}
	job := campaignworkers.DeadlineCompleter{
		Campaigns: repo,
		Disabled:  true,
	}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("run disabled deadline completer: %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("expected no deadline repository calls when disabled")
	}
}
