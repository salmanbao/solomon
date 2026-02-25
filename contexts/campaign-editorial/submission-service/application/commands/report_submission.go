package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

type ReportSubmissionCommand struct {
	IdempotencyKey string
	SubmissionID   string
	ReporterID     string
	Reason         string
	Description    string
}

type ReportSubmissionUseCase struct {
	Repository     ports.Repository
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	Outbox         ports.OutboxWriter
	Idempotency    ports.IdempotencyStore
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func (uc ReportSubmissionUseCase) Execute(ctx context.Context, cmd ReportSubmissionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(cmd.ReporterID) == "" {
		return domainerrors.ErrUnauthorizedActor
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return domainerrors.ErrInvalidSubmissionInput
	}

	now := uc.resolveNow()
	requestHash := hashReportCommand(cmd)
	if uc.Idempotency != nil {
		if record, found, err := uc.Idempotency.GetRecord(ctx, cmd.IdempotencyKey, now); err != nil {
			return err
		} else if found {
			if record.RequestHash != requestHash {
				return domainerrors.ErrIdempotencyKeyConflict
			}
			return nil
		}
	}

	submission, err := uc.Repository.GetSubmission(ctx, strings.TrimSpace(cmd.SubmissionID))
	if err != nil {
		logger.Error("submission report lookup failed",
			"event", "submission_report_lookup_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	reportID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission report id generation failed",
			"event", "submission_report_id_generation_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
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
		if err == domainerrors.ErrAlreadyReported {
			return err
		}
		logger.Error("submission report persistence failed",
			"event", "submission_report_persistence_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}

	previousStatus := submission.Status
	submission.ReportedCount++
	if submission.Status == entities.SubmissionStatusPending {
		submission.Status = entities.SubmissionStatusFlagged
	}
	submission.UpdatedAt = now
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		logger.Error("submission report status update failed",
			"event", "submission_report_status_update_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	flagID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission report flag id generation failed",
			"event", "submission_report_flag_id_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	if err := uc.Repository.AddFlag(ctx, entities.SubmissionFlag{
		FlagID:       flagID,
		SubmissionID: submission.SubmissionID,
		FlagType:     "user_report",
		Severity:     "medium",
		Details: map[string]any{
			"reason":      strings.TrimSpace(cmd.Reason),
			"description": strings.TrimSpace(cmd.Description),
			"reporter_id": strings.TrimSpace(cmd.ReporterID),
		},
		CreatedAt: now,
	}); err != nil {
		logger.Error("submission report flag persistence failed",
			"event", "submission_report_flag_persistence_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}

	auditID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission report audit id generation failed",
			"event", "submission_report_audit_id_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	if err := uc.Repository.AddAudit(ctx, entities.SubmissionAudit{
		AuditID:      auditID,
		SubmissionID: submission.SubmissionID,
		Action:       "flagged",
		OldStatus:    previousStatus,
		NewStatus:    submission.Status,
		ActorID:      strings.TrimSpace(cmd.ReporterID),
		ActorRole:    "user",
		ReasonCode:   strings.TrimSpace(cmd.Reason),
		ReasonNotes:  strings.TrimSpace(cmd.Description),
		CreatedAt:    now,
	}); err != nil {
		logger.Error("submission report audit append failed",
			"event", "submission_report_audit_append_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	if uc.Outbox != nil {
		eventID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("submission report event id generation failed",
				"event", "submission_report_event_id_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		envelope, err := newSubmissionEnvelope(
			eventID,
			"submission.flagged",
			submission.SubmissionID,
			now,
			map[string]any{
				"submission_id": submission.SubmissionID,
				"creator_id":    submission.CreatorID,
				"user_id":       submission.CreatorID,
				"campaign_id":   submission.CampaignID,
				"reason":        strings.TrimSpace(cmd.Reason),
				"flagged_at":    now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("submission report envelope build failed",
				"event", "submission_report_envelope_build_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("submission report outbox append failed",
				"event", "submission_report_outbox_append_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
	}

	if uc.Idempotency != nil {
		payload, _ := json.Marshal(map[string]string{
			"submission_id": submission.SubmissionID,
			"status":        string(submission.Status),
		})
		if err := uc.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
			Key:             cmd.IdempotencyKey,
			RequestHash:     requestHash,
			ResponsePayload: payload,
			ExpiresAt:       now.Add(uc.resolveIdempotencyTTL()),
		}); err != nil {
			return err
		}
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

func (uc ReportSubmissionUseCase) resolveNow() time.Time {
	now := time.Now().UTC()
	if uc.Clock != nil {
		now = uc.Clock.Now().UTC()
	}
	return now
}

func (uc ReportSubmissionUseCase) resolveIdempotencyTTL() time.Duration {
	if uc.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return uc.IdempotencyTTL
}

func hashReportCommand(cmd ReportSubmissionCommand) string {
	raw, _ := json.Marshal(map[string]string{
		"submission_id": strings.TrimSpace(cmd.SubmissionID),
		"reporter_id":   strings.TrimSpace(cmd.ReporterID),
		"reason":        strings.TrimSpace(cmd.Reason),
		"description":   strings.TrimSpace(cmd.Description),
		"op":            "report",
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
