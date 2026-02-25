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

const unknownCampaignID = "00000000-0000-0000-0000-000000000000"

type BulkOperationCommand struct {
	IdempotencyKey string
	ActorID        string
	OperationType  string
	SubmissionIDs  []string
	ReasonCode     string
	Reason         string
}

type BulkOperationUseCase struct {
	Repository     ports.Repository
	Review         ReviewSubmissionUseCase
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

type BulkOperationResult struct {
	Processed      int `json:"processed"`
	SucceededCount int `json:"succeeded_count"`
	FailedCount    int `json:"failed_count"`
}

func (uc BulkOperationUseCase) Execute(ctx context.Context, cmd BulkOperationCommand) (BulkOperationResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return BulkOperationResult{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return BulkOperationResult{}, domainerrors.ErrUnauthorizedActor
	}
	operationType := strings.TrimSpace(cmd.OperationType)
	if operationType != "bulk_approve" && operationType != "bulk_reject" {
		return BulkOperationResult{}, domainerrors.ErrInvalidSubmissionInput
	}
	if len(cmd.SubmissionIDs) == 0 {
		return BulkOperationResult{}, domainerrors.ErrInvalidSubmissionInput
	}

	now := time.Now().UTC()
	if uc.Clock != nil {
		now = uc.Clock.Now().UTC()
	}
	requestHash := hashBulkOperationCommand(cmd)
	if uc.Idempotency != nil {
		record, found, err := uc.Idempotency.GetRecord(ctx, cmd.IdempotencyKey, now)
		if err != nil {
			return BulkOperationResult{}, err
		}
		if found {
			if record.RequestHash != requestHash {
				return BulkOperationResult{}, domainerrors.ErrIdempotencyKeyConflict
			}
			var replayed BulkOperationResult
			if err := json.Unmarshal(record.ResponsePayload, &replayed); err != nil {
				return BulkOperationResult{}, err
			}
			return replayed, nil
		}
	}

	result := BulkOperationResult{}
	for _, submissionID := range cmd.SubmissionIDs {
		targetID := strings.TrimSpace(submissionID)
		if targetID == "" {
			result.FailedCount++
			result.Processed++
			continue
		}
		itemIdempotencyKey := cmd.IdempotencyKey + ":" + targetID + ":" + operationType
		var opErr error
		switch operationType {
		case "bulk_approve":
			opErr = uc.Review.Approve(ctx, ApproveSubmissionCommand{
				IdempotencyKey: itemIdempotencyKey,
				SubmissionID:   targetID,
				ActorID:        cmd.ActorID,
				Reason:         strings.TrimSpace(cmd.ReasonCode),
			})
		case "bulk_reject":
			opErr = uc.Review.Reject(ctx, RejectSubmissionCommand{
				IdempotencyKey: itemIdempotencyKey,
				SubmissionID:   targetID,
				ActorID:        cmd.ActorID,
				Reason:         resolveBulkReason(cmd),
				Notes:          strings.TrimSpace(cmd.Reason),
			})
		}
		result.Processed++
		if opErr != nil {
			result.FailedCount++
			continue
		}
		result.SucceededCount++
	}

	operationID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return BulkOperationResult{}, err
	}
	campaignID := uc.resolveCampaignID(ctx, cmd.SubmissionIDs)
	if err := uc.Repository.AddBulkOperation(ctx, entities.BulkSubmissionOperation{
		OperationID:       operationID,
		CampaignID:        campaignID,
		OperationType:     operationType,
		SubmissionIDs:     sanitizeIDs(cmd.SubmissionIDs),
		PerformedByUserID: strings.TrimSpace(cmd.ActorID),
		SucceededCount:    result.SucceededCount,
		FailedCount:       result.FailedCount,
		ReasonCode:        strings.TrimSpace(cmd.ReasonCode),
		ReasonNotes:       strings.TrimSpace(cmd.Reason),
		CreatedAt:         now,
	}); err != nil {
		return BulkOperationResult{}, err
	}

	if uc.Idempotency != nil {
		payload, err := json.Marshal(result)
		if err != nil {
			return BulkOperationResult{}, err
		}
		if err := uc.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
			Key:             cmd.IdempotencyKey,
			RequestHash:     requestHash,
			ResponsePayload: payload,
			ExpiresAt:       now.Add(uc.resolveIdempotencyTTL()),
		}); err != nil {
			return BulkOperationResult{}, err
		}
	}

	logger.Info("submission bulk operation completed",
		"event", "submission_bulk_operation_completed",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"operation_type", operationType,
		"processed", result.Processed,
		"succeeded_count", result.SucceededCount,
		"failed_count", result.FailedCount,
	)
	return result, nil
}

func (uc BulkOperationUseCase) resolveCampaignID(ctx context.Context, submissionIDs []string) string {
	for _, submissionID := range submissionIDs {
		targetID := strings.TrimSpace(submissionID)
		if targetID == "" {
			continue
		}
		item, err := uc.Repository.GetSubmission(ctx, targetID)
		if err == nil && strings.TrimSpace(item.CampaignID) != "" {
			return item.CampaignID
		}
	}
	return unknownCampaignID
}

func (uc BulkOperationUseCase) resolveIdempotencyTTL() time.Duration {
	if uc.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return uc.IdempotencyTTL
}

func sanitizeIDs(ids []string) []string {
	items := make([]string, 0, len(ids))
	for _, item := range ids {
		if v := strings.TrimSpace(item); v != "" {
			items = append(items, v)
		}
	}
	return items
}

func resolveBulkReason(cmd BulkOperationCommand) string {
	if reasonCode := strings.TrimSpace(cmd.ReasonCode); reasonCode != "" {
		return reasonCode
	}
	if reason := strings.TrimSpace(cmd.Reason); reason != "" {
		return reason
	}
	return "bulk_reject"
}

func hashBulkOperationCommand(cmd BulkOperationCommand) string {
	payload := map[string]any{
		"actor_id":       strings.TrimSpace(cmd.ActorID),
		"operation_type": strings.TrimSpace(cmd.OperationType),
		"submission_ids": sanitizeIDs(cmd.SubmissionIDs),
		"reason_code":    strings.TrimSpace(cmd.ReasonCode),
		"reason":         strings.TrimSpace(cmd.Reason),
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
