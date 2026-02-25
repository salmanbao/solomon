package commands

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

type ReportSubmissionCommand struct {
	SubmissionID string
	ReporterID   string
	Reason       string
	Description  string
}

type ReportSubmissionUseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	IDGen      ports.IDGenerator
	Logger     *slog.Logger
}

func (uc ReportSubmissionUseCase) Execute(ctx context.Context, cmd ReportSubmissionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	submission, err := uc.Repository.GetSubmission(ctx, strings.TrimSpace(cmd.SubmissionID))
	if err != nil {
		return err
	}
	now := uc.Clock.Now().UTC()
	reportID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	if err := uc.Repository.AddReport(ctx, entities.SubmissionReport{
		ReportID:     reportID,
		SubmissionID: submission.SubmissionID,
		ReportedByID: strings.TrimSpace(cmd.ReporterID),
		Reason:       strings.TrimSpace(cmd.Reason),
		Description:  strings.TrimSpace(cmd.Description),
		ReportedAt:   now,
	}); err != nil {
		return err
	}

	submission.ReportedCount++
	if submission.Status == entities.SubmissionStatusPending {
		submission.Status = entities.SubmissionStatusFlagged
	}
	submission.UpdatedAt = now
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		return err
	}

	logger.Warn("submission reported",
		"event", "submission_reported",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
		"report_count", submission.ReportedCount,
	)
	return nil
}
