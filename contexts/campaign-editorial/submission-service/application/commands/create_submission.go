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

type CreateSubmissionCommand struct {
	IdempotencyKey string
	CreatorID      string
	CampaignID     string
	Platform       string
	PostURL        string
	CpvRate        float64
}

type CreateSubmissionUseCase struct {
	Repository     ports.Repository
	Campaigns      ports.CampaignReadRepository
	Idempotency    ports.IdempotencyStore
	Outbox         ports.OutboxWriter
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

type CreateSubmissionResult struct {
	Submission entities.Submission
	Replayed   bool
}

type createSubmissionReplayPayload struct {
	SubmissionID string `json:"submission_id"`
}

func (uc CreateSubmissionUseCase) Execute(ctx context.Context, cmd CreateSubmissionCommand) (CreateSubmissionResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		logger.Error("submission create failed: missing idempotency key",
			"event", "submission_create_missing_idempotency",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
		)
		return CreateSubmissionResult{}, domainerrors.ErrIdempotencyKeyRequired
	}

	now := uc.Clock.Now().UTC()
	requestHash := hashCreateSubmissionCommand(cmd)
	if uc.Idempotency != nil {
		if record, found, err := uc.Idempotency.GetRecord(ctx, cmd.IdempotencyKey, now); err != nil {
			logger.Error("submission create idempotency lookup failed",
				"event", "submission_create_idempotency_lookup_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		} else if found {
			if record.RequestHash != requestHash {
				logger.Error("submission create idempotency conflict",
					"event", "submission_create_idempotency_conflict",
					"module", "campaign-editorial/submission-service",
					"layer", "application",
					"idempotency_key", cmd.IdempotencyKey,
				)
				return CreateSubmissionResult{}, domainerrors.ErrIdempotencyKeyConflict
			}
			var payload createSubmissionReplayPayload
			if err := json.Unmarshal(record.ResponsePayload, &payload); err != nil {
				logger.Error("submission create idempotency payload decode failed",
					"event", "submission_create_idempotency_decode_failed",
					"module", "campaign-editorial/submission-service",
					"layer", "application",
					"error", err.Error(),
				)
				return CreateSubmissionResult{}, err
			}
			item, err := uc.Repository.GetSubmission(ctx, payload.SubmissionID)
			if err != nil {
				logger.Error("submission create replay lookup failed",
					"event", "submission_create_replay_lookup_failed",
					"module", "campaign-editorial/submission-service",
					"layer", "application",
					"submission_id", payload.SubmissionID,
					"error", err.Error(),
				)
				return CreateSubmissionResult{}, err
			}
			return CreateSubmissionResult{
				Submission: item,
				Replayed:   true,
			}, nil
		}
	}

	normalizedPlatform := entities.NormalizePlatform(cmd.Platform)
	if !entities.IsSupportedPlatform(normalizedPlatform) {
		logger.Error("submission create failed: unsupported platform",
			"event", "submission_create_unsupported_platform",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"platform", cmd.Platform,
		)
		return CreateSubmissionResult{}, domainerrors.ErrUnsupportedPlatform
	}
	lockedCPVRate := cmd.CpvRate
	if uc.Campaigns != nil {
		campaign, err := uc.Campaigns.GetCampaignForSubmission(ctx, strings.TrimSpace(cmd.CampaignID))
		if err != nil {
			logger.Error("submission create campaign lookup failed",
				"event", "submission_create_campaign_lookup_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"campaign_id", strings.TrimSpace(cmd.CampaignID),
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
		if strings.ToLower(strings.TrimSpace(campaign.Status)) != "active" {
			logger.Error("submission create failed: campaign not active",
				"event", "submission_create_campaign_not_active",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"campaign_id", campaign.CampaignID,
				"campaign_status", campaign.Status,
			)
			return CreateSubmissionResult{}, domainerrors.ErrCampaignNotActive
		}
		if !campaignAllowsPlatform(campaign.AllowedPlatforms, normalizedPlatform) {
			logger.Error("submission create failed: platform not allowed",
				"event", "submission_create_platform_not_allowed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"campaign_id", campaign.CampaignID,
				"platform", normalizedPlatform,
			)
			return CreateSubmissionResult{}, domainerrors.ErrPlatformNotAllowed
		}
		if campaign.RatePer1KViews > 0 {
			lockedCPVRate = campaign.RatePer1KViews
		}
	}

	postID, handle, err := extractPostReference(normalizedPlatform, strings.TrimSpace(cmd.PostURL))
	if err != nil {
		logger.Error("submission create url parse failed",
			"event", "submission_create_url_parse_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"platform", normalizedPlatform,
			"error", err.Error(),
		)
		return CreateSubmissionResult{}, err
	}

	submissionID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission create id generation failed",
			"event", "submission_create_id_generation_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"error", err.Error(),
		)
		return CreateSubmissionResult{}, err
	}
	submission := entities.Submission{
		SubmissionID:          submissionID,
		CampaignID:            strings.TrimSpace(cmd.CampaignID),
		CreatorID:             strings.TrimSpace(cmd.CreatorID),
		Platform:              normalizedPlatform,
		PostURL:               strings.TrimSpace(cmd.PostURL),
		PostID:                postID,
		CreatorPlatformHandle: handle,
		Status:                entities.SubmissionStatusPending,
		CreatedAt:             now,
		UpdatedAt:             now,
		CpvRate:               lockedCPVRate,
	}
	if !submission.ValidateCreate() {
		logger.Error("submission create failed: invalid input",
			"event", "submission_create_invalid_input",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"campaign_id", submission.CampaignID,
			"creator_id", submission.CreatorID,
		)
		return CreateSubmissionResult{}, domainerrors.ErrInvalidSubmissionInput
	}
	if err := uc.Repository.CreateSubmission(ctx, submission); err != nil {
		logger.Error("submission create persistence failed",
			"event", "submission_create_persistence_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"campaign_id", submission.CampaignID,
			"creator_id", submission.CreatorID,
			"error", err.Error(),
		)
		return CreateSubmissionResult{}, err
	}

	auditID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		logger.Error("submission create audit id generation failed",
			"event", "submission_create_audit_id_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", submission.SubmissionID,
			"error", err.Error(),
		)
		return CreateSubmissionResult{}, err
	}
	if err := uc.Repository.AddAudit(ctx, entities.SubmissionAudit{
		AuditID:      auditID,
		SubmissionID: submission.SubmissionID,
		Action:       "created",
		NewStatus:    entities.SubmissionStatusPending,
		ActorID:      submission.CreatorID,
		ActorRole:    "creator",
		CreatedAt:    now,
	}); err != nil {
		logger.Error("submission create audit append failed",
			"event", "submission_create_audit_append_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "application",
			"submission_id", submission.SubmissionID,
			"error", err.Error(),
		)
		return CreateSubmissionResult{}, err
	}

	if uc.Outbox != nil {
		eventID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("submission create event id generation failed",
				"event", "submission_create_event_id_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
		envelope, err := newSubmissionEnvelope(
			eventID,
			"submission.created",
			submission.SubmissionID,
			now,
			map[string]any{
				"submission_id": submission.SubmissionID,
				"creator_id":    submission.CreatorID,
				"user_id":       submission.CreatorID,
				"campaign_id":   submission.CampaignID,
				"status":        string(submission.Status),
				"created_at":    now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("submission create envelope build failed",
				"event", "submission_create_envelope_build_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("submission create outbox append failed",
				"event", "submission_create_outbox_append_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
	}

	if uc.Idempotency != nil {
		payload, err := json.Marshal(createSubmissionReplayPayload{
			SubmissionID: submission.SubmissionID,
		})
		if err != nil {
			logger.Error("submission create replay payload encode failed",
				"event", "submission_create_replay_encode_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"submission_id", submission.SubmissionID,
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
		if err := uc.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
			Key:             cmd.IdempotencyKey,
			RequestHash:     requestHash,
			ResponsePayload: payload,
			ExpiresAt:       now.Add(uc.resolveIdempotencyTTL()),
		}); err != nil {
			logger.Error("submission create idempotency persist failed",
				"event", "submission_create_idempotency_persist_failed",
				"module", "campaign-editorial/submission-service",
				"layer", "application",
				"idempotency_key", cmd.IdempotencyKey,
				"error", err.Error(),
			)
			return CreateSubmissionResult{}, err
		}
	}

	logger.Info("submission created",
		"event", "submission_created",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
		"campaign_id", submission.CampaignID,
		"creator_id", submission.CreatorID,
	)
	return CreateSubmissionResult{Submission: submission}, nil
}

func (uc CreateSubmissionUseCase) resolveIdempotencyTTL() time.Duration {
	if uc.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return uc.IdempotencyTTL
}

func hashCreateSubmissionCommand(cmd CreateSubmissionCommand) string {
	payload := map[string]any{
		"creator_id":  strings.TrimSpace(cmd.CreatorID),
		"campaign_id": strings.TrimSpace(cmd.CampaignID),
		"platform":    entities.NormalizePlatform(cmd.Platform),
		"post_url":    strings.TrimSpace(cmd.PostURL),
		"cpv_rate":    cmd.CpvRate,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func campaignAllowsPlatform(allowed []string, platform string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if entities.NormalizePlatform(item) == entities.NormalizePlatform(platform) {
			return true
		}
	}
	return false
}
