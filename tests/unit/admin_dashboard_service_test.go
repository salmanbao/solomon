package unit

import (
	"context"
	"errors"
	"testing"

	admindashboardservice "solomon/contexts/internal-ops/admin-dashboard-service"
	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	httptransport "solomon/contexts/internal-ops/admin-dashboard-service/transport/http"
)

func TestAdminDashboardActionIdempotency(t *testing.T) {
	module := admindashboardservice.NewInMemoryModule()
	ctx := context.Background()

	first, err := module.Handler.RecordAdminActionHandler(ctx, "admin-1", "idem-admin-1", httptransport.RecordAdminActionRequest{
		Action:        "user_suspend",
		TargetID:      "user-123",
		Justification: "risk signal triggered",
		SourceIP:      "192.168.1.10",
		CorrelationID: "req-1",
	})
	if err != nil {
		t.Fatalf("first action failed: %v", err)
	}
	second, err := module.Handler.RecordAdminActionHandler(ctx, "admin-1", "idem-admin-1", httptransport.RecordAdminActionRequest{
		Action:        "user_suspend",
		TargetID:      "user-123",
		Justification: "risk signal triggered",
		SourceIP:      "192.168.1.10",
		CorrelationID: "req-1",
	})
	if err != nil {
		t.Fatalf("second action failed: %v", err)
	}
	if first.AuditID != second.AuditID {
		t.Fatalf("expected idempotent replay to return same audit id")
	}
}

func TestAdminDashboardRejectsIdempotencyCollision(t *testing.T) {
	module := admindashboardservice.NewInMemoryModule()
	ctx := context.Background()

	_, err := module.Handler.RecordAdminActionHandler(ctx, "admin-1", "idem-admin-2", httptransport.RecordAdminActionRequest{
		Action:        "user_suspend",
		TargetID:      "user-321",
		Justification: "policy violation",
	})
	if err != nil {
		t.Fatalf("initial action failed: %v", err)
	}
	_, err = module.Handler.RecordAdminActionHandler(ctx, "admin-1", "idem-admin-2", httptransport.RecordAdminActionRequest{
		Action:        "campaign_pause",
		TargetID:      "camp-444",
		Justification: "fraud review",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}
