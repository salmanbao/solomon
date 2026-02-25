package workers

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

// ClaimExpirer sweeps active claims that crossed expires_at.
type ClaimExpirer struct {
	Claims ports.ClaimRepository
	Clock  ports.Clock
	Logger *slog.Logger
}

func (e ClaimExpirer) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(e.Logger)
	now := time.Now().UTC()
	if e.Clock != nil {
		now = e.Clock.Now().UTC()
	}

	expired, err := e.Claims.ExpireActiveClaims(ctx, now)
	if err != nil {
		logger.Error("claim expiry sweep failed",
			"event", "content_marketplace_claim_expiry_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}
	if expired > 0 {
		logger.Info("claim expiry sweep completed",
			"event", "content_marketplace_claim_expiry_completed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "worker",
			"expired_count", expired,
		)
	}
	return nil
}
