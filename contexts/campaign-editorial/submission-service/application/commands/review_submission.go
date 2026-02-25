package commands

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

type ApproveSubmissionCommand struct {
	SubmissionID string
	ActorID      string
	Reason       string
}

type RejectSubmissionCommand struct {
	SubmissionID string
	ActorID      string
	Reason       string
	Notes        string
}

type ReviewSubmissionUseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	Logger     *slog.Logger
}

func (uc ReviewSubmissionUseCase) Approve(ctx context.Context, cmd ApproveSubmissionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	submission, err := uc.Repository.GetSubmission(ctx, strings.TrimSpace(cmd.SubmissionID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return domainerrors.ErrUnauthorizedActor
	}
	if submission.Status != entities.SubmissionStatusPending && submission.Status != entities.SubmissionStatusFlagged {
		return domainerrors.ErrInvalidStatusTransition
	}

	now := uc.Clock.Now().UTC()
	windowEnd := now.Add(30 * 24 * time.Hour)
	submission.Status = entities.SubmissionStatusApproved
	submission.ApprovedAt = &now
	submission.UpdatedAt = now
	submission.ApprovedByUserID = strings.TrimSpace(cmd.ActorID)
	submission.ApprovalReason = strings.TrimSpace(cmd.Reason)
	submission.VerificationWindowEnd = &windowEnd
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		return err
	}

	logger.Info("submission approved",
		"event", "submission_approved",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
	)
	return nil
}

func (uc ReviewSubmissionUseCase) Reject(ctx context.Context, cmd RejectSubmissionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	submission, err := uc.Repository.GetSubmission(ctx, strings.TrimSpace(cmd.SubmissionID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return domainerrors.ErrUnauthorizedActor
	}
	if submission.Status != entities.SubmissionStatusPending && submission.Status != entities.SubmissionStatusFlagged {
		return domainerrors.ErrInvalidStatusTransition
	}

	now := uc.Clock.Now().UTC()
	submission.Status = entities.SubmissionStatusRejected
	submission.UpdatedAt = now
	submission.RejectedAt = &now
	submission.RejectionReason = strings.TrimSpace(cmd.Reason)
	submission.RejectionNotes = strings.TrimSpace(cmd.Notes)
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		return err
	}

	logger.Info("submission rejected",
		"event", "submission_rejected",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
	)
	return nil
}
