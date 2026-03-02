package workers

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

// AutoApproveJob auto-approves pending submissions older than the policy threshold.
type AutoApproveJob struct {
	Repository  ports.Repository
	AutoApprove ports.AutoApproveRepository
	Clock       ports.Clock
	IDGen       ports.IDGenerator
	Outbox      ports.OutboxWriter
	BatchSize   int
	Disabled    bool
	Logger      *slog.Logger
}

func (j AutoApproveJob) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(j.Logger)
	if j.Disabled {
		logger.Info("submission auto-approve job disabled by feature flag",
			"event", "submission_auto_approve_disabled",
			"module", "campaign-editorial/submission-service",
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

	items, err := j.AutoApprove.ListPendingAutoApprove(ctx, now.Add(-48*time.Hour), limit)
	if err != nil {
		logger.Error("submission auto-approve list failed",
			"event", "submission_auto_approve_list_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}

	for _, submission := range items {
		previous := submission.Status
		windowEnd := now.Add(30 * 24 * time.Hour)
		submission.Status = entities.SubmissionStatusApproved
		submission.ApprovedAt = &now
		submission.ApprovedByUserID = ""
		submission.ApprovalReason = "auto_approve_48h"
		submission.VerificationStart = &now
		submission.VerificationWindowEnd = &windowEnd
		submission.UpdatedAt = now

		if err := j.Repository.UpdateSubmission(ctx, submission); err != nil {
			logger.Error("submission auto-approve update failed",
				"event", "submission_auto_approve_update_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "worker",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return err
		}

		auditID, err := j.IDGen.NewID(ctx)
		if err != nil {
			return err
		}
		if err := j.Repository.AddAudit(ctx, entities.SubmissionAudit{
			AuditID:      auditID,
			SubmissionID: submission.SubmissionID,
			Action:       "auto_approved",
			OldStatus:    previous,
			NewStatus:    entities.SubmissionStatusApproved,
			ActorID:      "system",
			ActorRole:    "system",
			ReasonCode:   "auto_approve_48h",
			CreatedAt:    now,
		}); err != nil {
			return err
		}

		if j.Outbox != nil {
			eventID, err := j.IDGen.NewID(ctx)
			if err != nil {
				return err
			}
			envelope, err := newWorkerSubmissionEnvelope(
				eventID,
				"submission.auto_approved",
				submission.SubmissionID,
				now,
				map[string]any{
					"submission_id":    submission.SubmissionID,
					"creator_id":       submission.CreatorID,
					"user_id":          submission.CreatorID,
					"campaign_id":      submission.CampaignID,
					"auto_approved_at": now.Format(time.RFC3339),
				},
			)
			if err != nil {
				return err
			}
			if err := j.Outbox.AppendOutbox(ctx, envelope); err != nil {
				return err
			}
		}
	}

	if len(items) > 0 {
		logger.Info("submission auto-approve cycle completed",
			"event", "submission_auto_approve_cycle_completed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"processed_count", len(items),
		)
	}
	return nil
}
