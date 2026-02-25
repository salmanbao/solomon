package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	distributionservice "solomon/contexts/campaign-editorial/distribution-service"
	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/distribution-service/transport/http"
)

func TestDistributionScheduleWindowValidation(t *testing.T) {
	module := distributionservice.NewInMemoryModule(nil, nil)

	item, err := module.Handler.Commands.Claim(context.Background(), commands.ClaimItemCommand{
		InfluencerID: "influencer-1",
		ClipID:       "clip-1",
		CampaignID:   "campaign-1",
	})
	if err != nil {
		t.Fatalf("claim item failed: %v", err)
	}

	err = module.Handler.ScheduleHandler(context.Background(), "influencer-1", item.ID, httptransport.ScheduleRequest{
		Platform:     "tiktok",
		ScheduledFor: time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339),
		Timezone:     "UTC",
	})
	if !errors.Is(err, domainerrors.ErrInvalidScheduleWindow) {
		t.Fatalf("expected invalid schedule window error, got %v", err)
	}
}

func TestDistributionPublishMultiFlow(t *testing.T) {
	module := distributionservice.NewInMemoryModule(nil, nil)

	item, err := module.Handler.Commands.Claim(context.Background(), commands.ClaimItemCommand{
		InfluencerID: "influencer-2",
		ClipID:       "clip-2",
		CampaignID:   "campaign-2",
	})
	if err != nil {
		t.Fatalf("claim item failed: %v", err)
	}

	err = module.Handler.PublishMultiHandler(context.Background(), "influencer-2", item.ID, httptransport.PublishMultiRequest{
		Platforms: []string{"tiktok", "instagram"},
		Caption:   "new clip",
	})
	if err != nil {
		t.Fatalf("publish multi failed: %v", err)
	}
}

func TestDistributionRescheduleRequiresScheduledState(t *testing.T) {
	module := distributionservice.NewInMemoryModule(nil, nil)

	item, err := module.Handler.Commands.Claim(context.Background(), commands.ClaimItemCommand{
		InfluencerID: "influencer-3",
		ClipID:       "clip-3",
		CampaignID:   "campaign-3",
	})
	if err != nil {
		t.Fatalf("claim item failed: %v", err)
	}

	err = module.Handler.RescheduleHandler(context.Background(), "influencer-3", item.ID, httptransport.ScheduleRequest{
		Platform:     "tiktok",
		ScheduledFor: time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		Timezone:     "UTC",
	})
	if !errors.Is(err, domainerrors.ErrInvalidStateTransition) {
		t.Fatalf("expected invalid state transition, got %v", err)
	}
}

func TestDistributionScheduleParsesTimezoneLocalTime(t *testing.T) {
	module := distributionservice.NewInMemoryModule(nil, nil)

	item, err := module.Handler.Commands.Claim(context.Background(), commands.ClaimItemCommand{
		InfluencerID: "influencer-4",
		ClipID:       "clip-4",
		CampaignID:   "campaign-4",
	})
	if err != nil {
		t.Fatalf("claim item failed: %v", err)
	}

	err = module.Handler.ScheduleHandler(context.Background(), "influencer-4", item.ID, httptransport.ScheduleRequest{
		Platform:     "tiktok",
		ScheduledFor: time.Now().UTC().Add(2 * time.Hour).Format("2006-01-02T15:04:05"),
		Timezone:     "America/New_York",
	})
	if err != nil {
		t.Fatalf("schedule should parse timezone local timestamp: %v", err)
	}
}
