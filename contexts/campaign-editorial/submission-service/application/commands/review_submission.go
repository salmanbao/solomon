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

type ApproveSubmissionCommand struct {
	IdempotencyKey string
	SubmissionID   string
	ActorID        string
	Reason         string
}

type RejectSubmissionCommand struct {
	IdempotencyKey string
	SubmissionID   string
	ActorID        string
	Reason         string
	Notes          string
}

type ReviewSubmissionUseCase struct {
	Repository     ports.Repository
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	Outbox         ports.OutboxWriter
	Idempotency    ports.IdempotencyStore
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func (uc ReviewSubmissionUseCase) Approve(ctx context.Context, cmd ApproveSubmissionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	now := uc.resolveNow()
	requestHash := hashApproveCommand(cmd)
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
		logger.Error("submission approve lookup failed",
			"event", "submission_approve_lookup_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		logger.Error("submission approve failed: missing actor",
			"event", "submission_approve_missing_actor",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
		)
		return domainerrors.ErrUnauthorizedActor
	}
	if submission.Status != entities.SubmissionStatusPending && submission.Status != entities.SubmissionStatusFlagged {
		logger.Error("submission approve invalid state transition",
			"event", "submission_approve_invalid_state",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"status", string(submission.Status),
		)
		return domainerrors.ErrInvalidStatusTransition
	}

	windowEnd := now.Add(30 * 24 * time.Hour)
	previousStatus := submission.Status
	submission.Status = entities.SubmissionStatusApproved
	submission.ApprovedAt = &now
	submission.UpdatedAt = now
	submission.ApprovedByUserID = strings.TrimSpace(cmd.ActorID)
	submission.ApprovalReason = strings.TrimSpace(cmd.Reason)
	submission.VerificationStart = &now
	submission.VerificationWindowEnd = &windowEnd
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		logger.Error("submission approve persistence failed",
			"event", "submission_approve_persistence_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	auditID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission approve audit id generation failed",
			"event", "submission_approve_audit_id_failed",
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
		Action:       "approved",
		OldStatus:    previousStatus,
		NewStatus:    entities.SubmissionStatusApproved,
		ActorID:      strings.TrimSpace(cmd.ActorID),
		ActorRole:    "brand_creator",
		ReasonCode:   strings.TrimSpace(cmd.Reason),
		CreatedAt:    now,
	}); err != nil {
		logger.Error("submission approve audit append failed",
			"event", "submission_approve_audit_append_failed",
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
			logger.Error("submission approve event id generation failed",
				"event", "submission_approve_event_id_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		envelope, err := newSubmissionEnvelope(
			eventID,
			"submission.approved",
			submission.SubmissionID,
			now,
			map[string]any{
				"submission_id": submission.SubmissionID,
				"creator_id":    submission.CreatorID,
				"user_id":       submission.CreatorID,
				"campaign_id":   submission.CampaignID,
				"approved_at":   now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("submission approve envelope build failed",
				"event", "submission_approve_envelope_build_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("submission approve outbox append failed",
				"event", "submission_approve_outbox_append_failed",
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
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	now := uc.resolveNow()
	requestHash := hashRejectCommand(cmd)
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
		logger.Error("submission reject lookup failed",
			"event", "submission_reject_lookup_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		logger.Error("submission reject failed: missing actor",
			"event", "submission_reject_missing_actor",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
		)
		return domainerrors.ErrUnauthorizedActor
	}
	if submission.Status != entities.SubmissionStatusPending && submission.Status != entities.SubmissionStatusFlagged {
		logger.Error("submission reject invalid state transition",
			"event", "submission_reject_invalid_state",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"status", string(submission.Status),
		)
		return domainerrors.ErrInvalidStatusTransition
	}

	previousStatus := submission.Status
	isCancelled := isCancellationReason(cmd.Reason)
	newStatus := entities.SubmissionStatusRejected
	action := "rejected"
	eventType := "submission.rejected"
	eventTimestampField := "rejected_at"
	if isCancelled {
		newStatus = entities.SubmissionStatusCancelled
		action = "cancelled"
		eventType = "submission.cancelled"
		eventTimestampField = "cancelled_at"
	}
	submission.Status = newStatus
	submission.UpdatedAt = now
	if isCancelled {
		submission.RejectedAt = nil
		submission.RejectionReason = ""
		submission.RejectionNotes = ""
	} else {
		submission.RejectedAt = &now
		submission.RejectionReason = strings.TrimSpace(cmd.Reason)
		submission.RejectionNotes = strings.TrimSpace(cmd.Notes)
	}
	if err := uc.Repository.UpdateSubmission(ctx, submission); err != nil {
		logger.Error("submission reject persistence failed",
			"event", "submission_reject_persistence_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", cmd.SubmissionID,
			"error", err.Error(),
		)
		return err
	}
	auditID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission reject audit id generation failed",
			"event", "submission_reject_audit_id_failed",
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
		Action:       action,
		OldStatus:    previousStatus,
		NewStatus:    newStatus,
		ActorID:      strings.TrimSpace(cmd.ActorID),
		ActorRole:    "brand_creator",
		ReasonCode:   strings.TrimSpace(cmd.Reason),
		ReasonNotes:  strings.TrimSpace(cmd.Notes),
		CreatedAt:    now,
	}); err != nil {
		logger.Error("submission reject audit append failed",
			"event", "submission_reject_audit_append_failed",
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
			logger.Error("submission reject event id generation failed",
				"event", "submission_reject_event_id_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		envelope, err := newSubmissionEnvelope(
			eventID,
			eventType,
			submission.SubmissionID,
			now,
			map[string]any{
				"submission_id":     submission.SubmissionID,
				"creator_id":        submission.CreatorID,
				"user_id":           submission.CreatorID,
				"campaign_id":       submission.CampaignID,
				"reason":            strings.TrimSpace(cmd.Reason),
				eventTimestampField: now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("submission reject envelope build failed",
				"event", "submission_reject_envelope_build_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", cmd.SubmissionID,
				"error", err.Error(),
			)
			return err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("submission reject outbox append failed",
				"event", "submission_reject_outbox_append_failed",
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

	resultEvent := "submission_rejected"
	if isCancelled {
		resultEvent = "submission_cancelled"
	}
	logger.Info("submission review action completed",
		"event", resultEvent,
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
		"status", string(submission.Status),
	)
	return nil
}

func (uc ReviewSubmissionUseCase) resolveNow() time.Time {
	now := time.Now().UTC()
	if uc.Clock != nil {
		now = uc.Clock.Now().UTC()
	}
	return now
}

func (uc ReviewSubmissionUseCase) resolveIdempotencyTTL() time.Duration {
	if uc.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return uc.IdempotencyTTL
}

func hashApproveCommand(cmd ApproveSubmissionCommand) string {
	raw, _ := json.Marshal(map[string]string{
		"submission_id": strings.TrimSpace(cmd.SubmissionID),
		"actor_id":      strings.TrimSpace(cmd.ActorID),
		"reason":        strings.TrimSpace(cmd.Reason),
		"op":            "approve",
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func hashRejectCommand(cmd RejectSubmissionCommand) string {
	raw, _ := json.Marshal(map[string]string{
		"submission_id": strings.TrimSpace(cmd.SubmissionID),
		"actor_id":      strings.TrimSpace(cmd.ActorID),
		"reason":        strings.TrimSpace(cmd.Reason),
		"notes":         strings.TrimSpace(cmd.Notes),
		"op":            "reject",
	})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func isCancellationReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "campaign_cancelled", "submission_cancelled", "cancelled":
		return true
	default:
		return false
	}
}
