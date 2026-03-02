package workers

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

// DeadlineCompleter sweeps active campaigns that crossed deadline.
type DeadlineCompleter struct {
	Campaigns ports.DeadlineRepository
	Clock     ports.Clock
	BatchSize int
	Disabled  bool
	Logger    *slog.Logger
}

func (j DeadlineCompleter) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(j.Logger)
	if j.Disabled {
		logger.Info("deadline completion sweep disabled by feature flag",
			"event", "campaign_deadline_completion_disabled",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
		)
		return nil
	}
	now := time.Now().UTC()
	if j.Clock != nil {
		now = j.Clock.Now().UTC()
	}

	limit := j.BatchSize
	if limit <= 0 {
		limit = 100
	}

	completed, err := j.Campaigns.CompleteCampaignsPastDeadline(ctx, now, limit)
	if err != nil {
		logger.Error("deadline completion sweep failed",
			"event", "campaign_deadline_completion_failed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}
	if len(completed) > 0 {
		logger.Info("deadline completion sweep completed",
			"event", "campaign_deadline_completion_completed",
			"module", "campaign-editorial/campaign-service",
			"layer", "worker",
			"completed_count", len(completed),
		)
	}
	return nil
}
