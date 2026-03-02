package unit

import (
	"context"
	"testing"
	"time"

	gamificationservice "solomon/contexts/community-experience/gamification-service"
	gamificationmemory "solomon/contexts/community-experience/gamification-service/adapters/memory"
	"solomon/contexts/community-experience/gamification-service/ports"
	httptransport "solomon/contexts/community-experience/gamification-service/transport/http"
)

func TestGamificationAwardPointsIdempotencyReplay(t *testing.T) {
	module := gamificationservice.NewInMemoryModule([]ports.UserProjection{
		{UserID: "user-gam-1", AuthActive: true, ProfileExists: true, ReputationTier: "gold"},
	}, nil)
	ctx := context.Background()

	first, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-award-1", httptransport.AwardPointsRequest{
		UserID:     "user-gam-1",
		ActionType: "submission_approved",
		Points:     10,
		Reason:     "approved",
	})
	if err != nil {
		t.Fatalf("first award points failed: %v", err)
	}
	second, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-award-1", httptransport.AwardPointsRequest{
		UserID:     "user-gam-1",
		ActionType: "submission_approved",
		Points:     10,
		Reason:     "approved",
	})
	if err != nil {
		t.Fatalf("second award points failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed response")
	}
	if first.Data.TotalPoints != second.Data.TotalPoints {
		t.Fatalf("expected replayed total points, got %d and %d", first.Data.TotalPoints, second.Data.TotalPoints)
	}
}

func TestGamificationEnforcesM01M02DependencyBoundary(t *testing.T) {
	module := gamificationservice.NewInMemoryModule([]ports.UserProjection{
		{UserID: "user-gam-2", AuthActive: false, ProfileExists: true, ReputationTier: "silver"},
	}, nil)
	ctx := context.Background()

	if _, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-award-2", httptransport.AwardPointsRequest{
		UserID:     "user-gam-2",
		ActionType: "submission_created",
		Points:     1,
	}); err == nil {
		t.Fatalf("expected dependency boundary error when auth projection inactive")
	}
}

func TestGamificationBadgeAndLeaderboardFlow(t *testing.T) {
	module := gamificationservice.NewInMemoryModule([]ports.UserProjection{
		{UserID: "user-gam-3", AuthActive: true, ProfileExists: true, ReputationTier: "platinum"},
		{UserID: "user-gam-4", AuthActive: true, ProfileExists: true, ReputationTier: "bronze"},
	}, nil)
	ctx := context.Background()

	if _, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-award-3", httptransport.AwardPointsRequest{
		UserID:     "user-gam-3",
		ActionType: "submission_created",
		Points:     10,
	}); err != nil {
		t.Fatalf("award user 3 failed: %v", err)
	}
	if _, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-award-4", httptransport.AwardPointsRequest{
		UserID:     "user-gam-4",
		ActionType: "submission_created",
		Points:     10,
	}); err != nil {
		t.Fatalf("award user 4 failed: %v", err)
	}
	badgeResp, err := module.Handler.GrantBadgeHandler(ctx, "idem-gam-badge-1", httptransport.GrantBadgeRequest{
		UserID:   "user-gam-3",
		BadgeKey: "first_submission",
		Reason:   "milestone",
	})
	if err != nil {
		t.Fatalf("grant badge failed: %v", err)
	}
	if badgeResp.Data.BadgeKey != "first_submission" {
		t.Fatalf("unexpected badge key: %s", badgeResp.Data.BadgeKey)
	}

	summary, err := module.Handler.GetUserSummaryHandler(ctx, "user-gam-3")
	if err != nil {
		t.Fatalf("get summary failed: %v", err)
	}
	if len(summary.Data.Badges) != 1 || summary.Data.Badges[0] != "first_submission" {
		t.Fatalf("unexpected badge summary: %#v", summary.Data.Badges)
	}

	leaderboard, err := module.Handler.GetLeaderboardHandler(ctx, 10, 0)
	if err != nil {
		t.Fatalf("get leaderboard failed: %v", err)
	}
	if len(leaderboard.Data) < 2 {
		t.Fatalf("expected leaderboard entries")
	}
	if leaderboard.Data[0].UserID != "user-gam-3" {
		t.Fatalf("expected user-gam-3 ranked first, got %s", leaderboard.Data[0].UserID)
	}
}

func TestGamificationCanDisableTierMultiplier(t *testing.T) {
	storeSeed := []ports.UserProjection{
		{UserID: "user-gam-flag", AuthActive: true, ProfileExists: true, ReputationTier: "gold"},
	}
	store := gamificationmemory.NewStore(storeSeed)
	module := gamificationservice.NewModule(gamificationservice.Dependencies{
		Repository:            store,
		Idempotency:           store,
		Clock:                 store,
		IDGenerator:           store,
		IdempotencyTTL:        7 * 24 * time.Hour,
		DisableTierMultiplier: true,
	})
	ctx := context.Background()

	resp, err := module.Handler.AwardPointsHandler(ctx, "idem-gam-flag-1", httptransport.AwardPointsRequest{
		UserID:     "user-gam-flag",
		ActionType: "submission_created",
		Points:     10,
		Reason:     "flag test",
	})
	if err != nil {
		t.Fatalf("award points failed: %v", err)
	}
	if resp.Data.TotalPoints != 10 {
		t.Fatalf("expected total points without multiplier to be 10, got %d", resp.Data.TotalPoints)
	}
}
