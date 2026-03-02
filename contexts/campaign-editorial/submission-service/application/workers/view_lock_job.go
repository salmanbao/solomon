package workers

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

// ViewLockJob locks views for submissions that completed verification period.
type ViewLockJob struct {
	Repository      ports.Repository
	ViewLock        ports.ViewLockRepository
	Clock           ports.Clock
	IDGen           ports.IDGenerator
	Outbox          ports.OutboxWriter
	BatchSize       int
	PlatformFeeRate float64
	Disabled        bool
	Logger          *slog.Logger
}

func (j ViewLockJob) RunOnce(ctx context.Context) error {
	logger := application.ResolveLogger(j.Logger)
	if j.Disabled {
		logger.Info("submission view-lock job disabled by feature flag",
			"event", "submission_view_lock_disabled",
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
	feeRate := j.PlatformFeeRate
	if feeRate <= 0 {
		feeRate = 0.15
	}

	items, err := j.ViewLock.ListDueViewLock(ctx, now, limit)
	if err != nil {
		logger.Error("submission view-lock list failed",
			"event", "submission_view_lock_list_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"error", err.Error(),
		)
		return err
	}

	for _, submission := range items {
		previous := submission.Status
		lockedViews := submission.ViewsCount
		gross := (float64(lockedViews) / 1000.0) * submission.CpvRate
		platformFee := gross * feeRate
		net := gross - platformFee

		submission.LockedViews = &lockedViews
		submission.LockedAt = &now
		submission.GrossAmount = gross
		submission.PlatformFee = platformFee
		submission.NetAmount = net
		submission.Status = entities.SubmissionStatusViewLocked
		submission.UpdatedAt = now

		if err := j.Repository.UpdateSubmission(ctx, submission); err != nil {
			logger.Error("submission view-lock update failed",
				"event", "submission_view_lock_update_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "worker",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return err
		}

		snapshotID, err := j.IDGen.NewID(ctx)
		if err != nil {
			return err
		}
		if err := j.Repository.AddViewSnapshot(ctx, entities.ViewSnapshot{
			SnapshotID:   snapshotID,
			SubmissionID: submission.SubmissionID,
			ViewsCount:   submission.ViewsCount,
			SyncedAt:     now,
		}); err != nil {
			return err
		}

		auditID, err := j.IDGen.NewID(ctx)
		if err != nil {
			return err
		}
		if err := j.Repository.AddAudit(ctx, entities.SubmissionAudit{
			AuditID:      auditID,
			SubmissionID: submission.SubmissionID,
			Action:       "view_locked",
			OldStatus:    previous,
			NewStatus:    entities.SubmissionStatusViewLocked,
			ActorID:      "system",
			ActorRole:    "system",
			ReasonCode:   "verification_completed",
			CreatedAt:    now,
		}); err != nil {
			return err
		}

		if j.Outbox != nil {
			verifiedEventID, err := j.IDGen.NewID(ctx)
			if err != nil {
				return err
			}
			verifiedEnvelope, err := newWorkerSubmissionEnvelope(
				verifiedEventID,
				"submission.verified",
				submission.SubmissionID,
				now,
				map[string]any{
					"submission_id": submission.SubmissionID,
					"creator_id":    submission.CreatorID,
					"user_id":       submission.CreatorID,
					"campaign_id":   submission.CampaignID,
					"verified_at":   now.Format(time.RFC3339),
				},
			)
			if err != nil {
				return err
			}
			if err := j.Outbox.AppendOutbox(ctx, verifiedEnvelope); err != nil {
				return err
			}

			viewLockEventID, err := j.IDGen.NewID(ctx)
			if err != nil {
				return err
			}
			viewLockEnvelope, err := newWorkerSubmissionEnvelope(
				viewLockEventID,
				"submission.view_locked",
				submission.SubmissionID,
				now,
				map[string]any{
					"submission_id": submission.SubmissionID,
					"creator_id":    submission.CreatorID,
					"user_id":       submission.CreatorID,
					"campaign_id":   submission.CampaignID,
					"locked_views":  lockedViews,
					"locked_at":     now.Format(time.RFC3339),
				},
			)
			if err != nil {
				return err
			}
			if err := j.Outbox.AppendOutbox(ctx, viewLockEnvelope); err != nil {
				return err
			}
		}
	}

	if len(items) > 0 {
		logger.Info("submission view-lock cycle completed",
			"event", "submission_view_lock_cycle_completed",
			"module", "campaign-editorial/submission-service",
			"layer", "worker",
			"processed_count", len(items),
		)
	}
	return nil
}
