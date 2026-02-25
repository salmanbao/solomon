package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/services"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

const claimedEventType = "distribution.claimed"

type ClaimClipCommand struct {
	ClipID         string
	UserID         string
	RequestID      string
	IdempotencyKey string
}

type ClaimClipResult struct {
	Claim    entities.Claim
	Created  bool
	Replayed bool
}

type ClaimClipUseCase struct {
	Clips          ports.ClipRepository
	Claims         ports.ClaimRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	ClaimTTL       time.Duration
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

// Execute runs the claim workflow in this order:
// 1) idempotency lookup/replay
// 2) domain eligibility validation
// 3) atomic claim + outbox persistence
// 4) idempotency record write.
func (u ClaimClipUseCase) Execute(ctx context.Context, cmd ClaimClipCommand) (ClaimClipResult, error) {
	logger := application.ResolveLogger(u.Logger)
	if strings.TrimSpace(cmd.ClipID) == "" ||
		strings.TrimSpace(cmd.UserID) == "" ||
		strings.TrimSpace(cmd.RequestID) == "" {
		return ClaimClipResult{}, domainerrors.ErrInvalidClaimRequest
	}

	now := u.now()
	idempotencyKey := resolveIdempotencyKey(cmd)
	requestHash := hashRequest(cmd)

	logger.Info("claim clip started",
		"event", "claim_clip_started",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"user_id", cmd.UserID,
		"clip_id", cmd.ClipID,
		"idempotency_key", idempotencyKey,
	)

	idempotencyRecord, found, err := u.Idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		logger.Error("idempotency get failed",
			"event", "claim_clip_idempotency_get_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", cmd.ClipID,
			"user_id", cmd.UserID,
			"error", err.Error(),
		)
		return ClaimClipResult{}, err
	}
	if found {
		// A reused idempotency key must map to an identical request payload.
		if idempotencyRecord.RequestHash != requestHash {
			logger.Warn("idempotency key conflict",
				"event", "claim_clip_idempotency_conflict",
				"module", "campaign-editorial/content-library-marketplace",
				"layer", "application",
				"clip_id", cmd.ClipID,
				"user_id", cmd.UserID,
			)
			return ClaimClipResult{}, domainerrors.ErrIdempotencyKeyConflict
		}
		claim, err := u.Claims.GetClaim(ctx, idempotencyRecord.ClaimID)
		if err != nil {
			logger.Error("idempotency replay failed to load claim",
				"event", "claim_clip_idempotency_replay_load_failed",
				"module", "campaign-editorial/content-library-marketplace",
				"layer", "application",
				"claim_id", idempotencyRecord.ClaimID,
				"error", err.Error(),
			)
			return ClaimClipResult{}, err
		}

		logger.Info("claim clip replayed from idempotency",
			"event", "claim_clip_replayed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"claim_id", claim.ClaimID,
			"clip_id", claim.ClipID,
			"user_id", claim.UserID,
		)
		return ClaimClipResult{Claim: claim, Replayed: true}, nil
	}

	// request_id dedupe is maintained separately so callers can safely retry even
	// when idempotency headers are not consistently reused.
	if byRequest, requestFound, err := u.Claims.GetClaimByRequestID(ctx, cmd.RequestID); err != nil {
		return ClaimClipResult{}, err
	} else if requestFound {
		if err := u.Idempotency.Put(ctx, ports.IdempotencyRecord{
			Key:         idempotencyKey,
			RequestHash: requestHash,
			ClaimID:     byRequest.ClaimID,
			ExpiresAt:   now.Add(u.idempotencyTTL()),
		}); err != nil {
			return ClaimClipResult{}, err
		}
		return ClaimClipResult{Claim: byRequest, Replayed: true}, nil
	}

	clip, err := u.Clips.GetClip(ctx, cmd.ClipID)
	if err != nil {
		logger.Error("claim clip failed loading clip",
			"event", "claim_clip_get_clip_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", cmd.ClipID,
			"error", err.Error(),
		)
		return ClaimClipResult{}, err
	}
	claims, err := u.Claims.ListClaimsByClip(ctx, cmd.ClipID)
	if err != nil {
		logger.Error("claim clip failed loading clip claims",
			"event", "claim_clip_list_claims_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", cmd.ClipID,
			"error", err.Error(),
		)
		return ClaimClipResult{}, err
	}

	existingClaim, err := services.EvaluateClaimEligibility(clip, claims, cmd.UserID, now)
	if err != nil {
		logger.Warn("claim clip conflict",
			"event", "claim_clip_conflict",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", cmd.ClipID,
			"user_id", cmd.UserID,
			"error", err.Error(),
		)
		return ClaimClipResult{}, err
	}
	if existingClaim != nil {
		if err := u.Idempotency.Put(ctx, ports.IdempotencyRecord{
			Key:         idempotencyKey,
			RequestHash: requestHash,
			ClaimID:     existingClaim.ClaimID,
			ExpiresAt:   now.Add(u.idempotencyTTL()),
		}); err != nil {
			return ClaimClipResult{}, err
		}
		return ClaimClipResult{Claim: *existingClaim, Replayed: true}, nil
	}

	claimID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return ClaimClipResult{}, err
	}
	claimType := entities.ClaimType(clip.Exclusivity)
	claim, err := entities.NewClaim(
		claimID,
		cmd.ClipID,
		cmd.UserID,
		claimType,
		cmd.RequestID,
		now,
		now.Add(u.claimTTL()),
	)
	if err != nil {
		return ClaimClipResult{}, err
	}

	eventID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return ClaimClipResult{}, err
	}
	event := ports.ClaimedEvent{
		EventID:      eventID,
		EventType:    claimedEventType,
		ClaimID:      claim.ClaimID,
		ClipID:       claim.ClipID,
		UserID:       claim.UserID,
		ClaimType:    string(claim.ClaimType),
		PartitionKey: claim.ClipID,
		OccurredAt:   now,
	}

	// M09 write boundary: claim row and distribution.claimed outbox message are
	// committed together by the repository adapter.
	if err := u.Claims.CreateClaimWithOutbox(ctx, claim, event); err != nil {
		logger.Error("claim clip failed on write transaction",
			"event", "claim_clip_write_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", cmd.ClipID,
			"user_id", cmd.UserID,
			"error", err.Error(),
		)
		return ClaimClipResult{}, err
	}

	if err := u.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         idempotencyKey,
		RequestHash: requestHash,
		ClaimID:     claim.ClaimID,
		ExpiresAt:   now.Add(u.idempotencyTTL()),
	}); err != nil {
		return ClaimClipResult{}, err
	}

	logger.Info("claim clip created",
		"event", "content_marketplace_claim_created",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"claim_id", claim.ClaimID,
		"clip_id", claim.ClipID,
		"user_id", claim.UserID,
		"claim_type", claim.ClaimType,
	)

	return ClaimClipResult{
		Claim:   claim,
		Created: true,
	}, nil
}

func (u ClaimClipUseCase) claimTTL() time.Duration {
	if u.ClaimTTL <= 0 {
		return 24 * time.Hour
	}
	return u.ClaimTTL
}

func (u ClaimClipUseCase) idempotencyTTL() time.Duration {
	if u.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return u.IdempotencyTTL
}

func (u ClaimClipUseCase) now() time.Time {
	if u.Clock == nil {
		return time.Now().UTC()
	}
	return u.Clock.Now().UTC()
}

func resolveIdempotencyKey(cmd ClaimClipCommand) string {
	if strings.TrimSpace(cmd.IdempotencyKey) != "" {
		return cmd.IdempotencyKey
	}
	// Canonical fallback pattern for claim operations.
	return fmt.Sprintf("cms:%s:%s:claim", cmd.UserID, cmd.ClipID)
}

func hashRequest(cmd ClaimClipCommand) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s", cmd.UserID, cmd.ClipID, cmd.RequestID)))
	return hex.EncodeToString(sum[:])
}
