package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	superadmindashboard "solomon/contexts/internal-ops/super-admin-dashboard"
	domainerrors "solomon/contexts/internal-ops/super-admin-dashboard/domain/errors"
	httptransport "solomon/contexts/internal-ops/super-admin-dashboard/transport/http"
)

func TestSuperAdminDashboardWalletAdjustIdempotency(t *testing.T) {
	module := superadmindashboard.NewInMemoryModule(nil)
	ctx := context.Background()

	first, err := module.Handler.AdjustWalletHandler(
		ctx,
		"admin-1",
		"idem-wallet-1",
		"user-1",
		httptransport.WalletAdjustRequest{
			Amount:         25,
			AdjustmentType: "credit",
			Reason:         "manual correction",
		},
	)
	if err != nil {
		t.Fatalf("first wallet adjust failed: %v", err)
	}

	second, err := module.Handler.AdjustWalletHandler(
		ctx,
		"admin-1",
		"idem-wallet-1",
		"user-1",
		httptransport.WalletAdjustRequest{
			Amount:         25,
			AdjustmentType: "credit",
			Reason:         "manual correction",
		},
	)
	if err != nil {
		t.Fatalf("replayed wallet adjust failed: %v", err)
	}
	if first.AdjustmentID != second.AdjustmentID {
		t.Fatalf("expected same adjustment id for idempotent replay, got %s and %s", first.AdjustmentID, second.AdjustmentID)
	}

	_, err = module.Handler.AdjustWalletHandler(
		ctx,
		"admin-1",
		"idem-wallet-1",
		"user-1",
		httptransport.WalletAdjustRequest{
			Amount:         50,
			AdjustmentType: "credit",
			Reason:         "different payload",
		},
	)
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestSuperAdminDashboardBanLifecycle(t *testing.T) {
	module := superadmindashboard.NewInMemoryModule(nil)
	ctx := context.Background()

	_, err := module.Handler.BanUserHandler(
		ctx,
		"admin-1",
		"idem-ban-1",
		"user-2",
		httptransport.BanUserRequest{
			BanType:      "temporary",
			DurationDays: 7,
			Reason:       "policy breach",
		},
	)
	if err != nil {
		t.Fatalf("ban user failed: %v", err)
	}

	_, err = module.Handler.UnbanUserHandler(
		ctx,
		"admin-1",
		"idem-unban-1",
		"user-2",
		httptransport.UnbanUserRequest{
			Reason: "appeal accepted",
		},
	)
	if err != nil {
		t.Fatalf("unban user failed: %v", err)
	}
}

func TestSuperAdminDashboardAnalyticsDateValidation(t *testing.T) {
	module := superadmindashboard.NewInMemoryModule(nil)
	_, err := module.Handler.AnalyticsDashboardHandler(
		context.Background(),
		time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
	)
	if !errors.Is(err, domainerrors.ErrUnprocessable) {
		t.Fatalf("expected unprocessable date range, got %v", err)
	}
}
