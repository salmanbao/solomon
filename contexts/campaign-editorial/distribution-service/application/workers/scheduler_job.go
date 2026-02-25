package workers

import (
	"context"
	"log/slog"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
)

// SchedulerJob runs periodic due-schedule publishing for M31.
type SchedulerJob struct {
	Commands  commands.UseCase
	BatchSize int
	Logger    *slog.Logger
}

func (j SchedulerJob) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(j.Logger)
	limit := j.BatchSize
	if limit <= 0 {
		limit = 100
	}
	if err := j.Commands.ProcessDueScheduled(ctx, limit); err != nil {
		logger.Error("distribution scheduler cycle failed",
			"event", "distribution_scheduler_cycle_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}
	logger.Debug("distribution scheduler cycle succeeded",
		"event", "distribution_scheduler_cycle_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "worker",
		"limit", limit,
	)
	return nil
}
