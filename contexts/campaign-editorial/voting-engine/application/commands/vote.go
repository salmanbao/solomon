package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/voting-engine/domain/errors"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

// CreateVoteCommand is the write-model input for vote creation/update.
type CreateVoteCommand struct {
	UserID         string
	IdempotencyKey string
	SubmissionID   string
	CampaignID     string
	RoundID        string
	VoteType       entities.VoteType
	IPAddress      string
	UserAgent      string
}

// CreateVoteResult returns final vote state and replay/update markers that the
// transport layer maps to API semantics.
type CreateVoteResult struct {
	Vote      entities.Vote
	Replayed  bool
	WasUpdate bool
}

// RetractVoteCommand requests a user-owned vote retraction.
type RetractVoteCommand struct {
	VoteID          string
	UserID          string
	IdempotencyKey  string
	RetractionCause string
}

// QuarantineActionCommand applies moderation decisions to quarantined votes.
type QuarantineActionCommand struct {
	QuarantineID   string
	Action         string
	ActorID        string
	IdempotencyKey string
}

// VoteUseCase orchestrates vote commands while enforcing M08 invariants:
// idempotency, eligibility checks, round validation, weighted scoring, and
// outbox event emission.
type VoteUseCase struct {
	Votes          ports.VoteRepository
	Idempotency    ports.IdempotencyStore
	Outbox         ports.OutboxWriter
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

// CreateVote creates or updates a vote by (submission_id, user_id, round_id).
// The method is replay-safe via idempotency key + request hash validation.
func (uc VoteUseCase) CreateVote(ctx context.Context, cmd CreateVoteCommand) (CreateVoteResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	logger.Info("vote create processing started",
		"event", "voting_vote_create_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"user_id", strings.TrimSpace(cmd.UserID),
		"submission_id", strings.TrimSpace(cmd.SubmissionID),
		"campaign_id", strings.TrimSpace(cmd.CampaignID),
		"round_id", strings.TrimSpace(cmd.RoundID),
	)
	if strings.TrimSpace(cmd.UserID) == "" ||
		strings.TrimSpace(cmd.SubmissionID) == "" ||
		(cmd.VoteType != entities.VoteTypeUpvote && cmd.VoteType != entities.VoteTypeDownvote) {
		logger.Warn("vote create validation failed",
			"event", "voting_vote_create_validation_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"user_id", strings.TrimSpace(cmd.UserID),
			"submission_id", strings.TrimSpace(cmd.SubmissionID),
		)
		return CreateVoteResult{}, domainerrors.ErrInvalidVoteInput
	}
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		logger.Warn("vote create idempotency key missing",
			"event", "voting_vote_create_idempotency_missing",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"user_id", strings.TrimSpace(cmd.UserID),
			"submission_id", strings.TrimSpace(cmd.SubmissionID),
		)
		return CreateVoteResult{}, domainerrors.ErrIdempotencyKeyRequired
	}

	now := uc.now()
	requestHash := hashCreateVoteCommand(cmd)
	if record, found, err := uc.Idempotency.Get(ctx, cmd.IdempotencyKey, now); err != nil {
		logger.Error("vote create idempotency lookup failed",
			"event", "voting_vote_create_idempotency_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"user_id", strings.TrimSpace(cmd.UserID),
			"submission_id", strings.TrimSpace(cmd.SubmissionID),
			"error", err.Error(),
		)
		return CreateVoteResult{}, err
	} else if found {
		if record.RequestHash != requestHash {
			logger.Warn("vote create idempotency conflict",
				"event", "voting_vote_create_idempotency_conflict",
				"module", "campaign-editorial/voting-engine",
				"layer", "application",
				"user_id", strings.TrimSpace(cmd.UserID),
				"submission_id", strings.TrimSpace(cmd.SubmissionID),
			)
			return CreateVoteResult{}, domainerrors.ErrIdempotencyConflict
		}
		vote, err := uc.Votes.GetVote(ctx, record.VoteID)
		if err != nil {
			return CreateVoteResult{}, err
		}
		logger.Info("vote create replayed",
			"event", "voting_vote_create_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", vote.VoteID,
			"submission_id", vote.SubmissionID,
			"user_id", strings.TrimSpace(cmd.UserID),
		)
		return CreateVoteResult{Vote: vote, Replayed: true}, nil
	}

	submission, err := uc.Votes.GetSubmission(ctx, cmd.SubmissionID)
	if err != nil {
		return CreateVoteResult{}, err
	}
	campaignID := strings.TrimSpace(submission.CampaignID)
	if strings.TrimSpace(cmd.CampaignID) != "" {
		if !strings.EqualFold(strings.TrimSpace(cmd.CampaignID), campaignID) {
			return CreateVoteResult{}, domainerrors.ErrInvalidVoteInput
		}
		campaignID = strings.TrimSpace(cmd.CampaignID)
	}

	if strings.EqualFold(strings.TrimSpace(submission.CreatorID), strings.TrimSpace(cmd.UserID)) {
		return CreateVoteResult{}, domainerrors.ErrSelfVoteForbidden
	}
	if !isSubmissionVoteEligible(strings.TrimSpace(submission.Status)) {
		return CreateVoteResult{}, domainerrors.ErrSubmissionNotFound
	}

	campaign, err := uc.Votes.GetCampaign(ctx, campaignID)
	if err != nil {
		return CreateVoteResult{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(campaign.Status), "active") {
		return CreateVoteResult{}, domainerrors.ErrCampaignNotActive
	}

	roundID, err := uc.resolveRoundID(ctx, campaignID, cmd.RoundID, now)
	if err != nil {
		return CreateVoteResult{}, err
	}
	scoreSnapshot, weight := uc.resolveWeight(ctx, cmd.UserID)

	if existing, found, err := uc.Votes.GetVoteByIdentity(ctx, cmd.SubmissionID, cmd.UserID, roundID); err != nil {
		return CreateVoteResult{}, err
	} else if found {
		existing.CampaignID = campaignID
		existing.RoundID = roundID
		existing.VoteType = cmd.VoteType
		existing.Weight = weight
		existing.ReputationScoreSnapshot = scoreSnapshot
		existing.IPAddress = strings.TrimSpace(cmd.IPAddress)
		existing.UserAgent = strings.TrimSpace(cmd.UserAgent)
		existing.Retracted = false
		existing.UpdatedAt = now
		if err := uc.Votes.SaveVote(ctx, existing); err != nil {
			return CreateVoteResult{}, err
		}
		if err := uc.appendVoteEvent(ctx, "vote.updated", existing, now, map[string]any{
			"reason": "vote_toggled_or_reactivated",
		}); err != nil {
			return CreateVoteResult{}, err
		}
		if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
			Key:         cmd.IdempotencyKey,
			RequestHash: requestHash,
			VoteID:      existing.VoteID,
			ExpiresAt:   now.Add(uc.resolveIdempotencyTTL()),
		}); err != nil {
			return CreateVoteResult{}, err
		}
		logger.Info("vote updated",
			"event", "voting_vote_updated",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", existing.VoteID,
			"submission_id", existing.SubmissionID,
			"user_id", existing.UserID,
			"vote_type", string(existing.VoteType),
			"weight", existing.Weight,
			"round_id", existing.RoundID,
		)
		return CreateVoteResult{Vote: existing, WasUpdate: true}, nil
	}

	voteID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return CreateVoteResult{}, err
	}
	vote := entities.Vote{
		VoteID:                  voteID,
		SubmissionID:            strings.TrimSpace(cmd.SubmissionID),
		CampaignID:              campaignID,
		RoundID:                 roundID,
		UserID:                  strings.TrimSpace(cmd.UserID),
		VoteType:                cmd.VoteType,
		Weight:                  weight,
		ReputationScoreSnapshot: scoreSnapshot,
		IPAddress:               strings.TrimSpace(cmd.IPAddress),
		UserAgent:               strings.TrimSpace(cmd.UserAgent),
		Retracted:               false,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := uc.Votes.SaveVote(ctx, vote); err != nil {
		return CreateVoteResult{}, err
	}
	if err := uc.appendVoteEvent(ctx, "vote.created", vote, now, nil); err != nil {
		return CreateVoteResult{}, err
	}
	if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         cmd.IdempotencyKey,
		RequestHash: requestHash,
		VoteID:      vote.VoteID,
		ExpiresAt:   now.Add(uc.resolveIdempotencyTTL()),
	}); err != nil {
		return CreateVoteResult{}, err
	}

	logger.Info("vote created",
		"event", "voting_vote_created",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"vote_id", vote.VoteID,
		"submission_id", vote.SubmissionID,
		"user_id", vote.UserID,
		"vote_type", string(vote.VoteType),
		"weight", vote.Weight,
		"round_id", vote.RoundID,
	)
	return CreateVoteResult{Vote: vote}, nil
}

// RetractVote performs user-initiated vote retraction and emits vote.retracted.
// Retract operations are idempotent via the supplied idempotency key.
func (uc VoteUseCase) RetractVote(ctx context.Context, cmd RetractVoteCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	logger.Info("vote retract processing started",
		"event", "voting_vote_retract_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"vote_id", strings.TrimSpace(cmd.VoteID),
		"user_id", strings.TrimSpace(cmd.UserID),
	)
	if strings.TrimSpace(cmd.VoteID) == "" || strings.TrimSpace(cmd.UserID) == "" {
		logger.Warn("vote retract validation failed",
			"event", "voting_vote_retract_validation_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", strings.TrimSpace(cmd.VoteID),
			"user_id", strings.TrimSpace(cmd.UserID),
		)
		return domainerrors.ErrInvalidVoteInput
	}
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		logger.Warn("vote retract idempotency key missing",
			"event", "voting_vote_retract_idempotency_missing",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", strings.TrimSpace(cmd.VoteID),
			"user_id", strings.TrimSpace(cmd.UserID),
		)
		return domainerrors.ErrIdempotencyKeyRequired
	}

	now := uc.now()
	requestHash := hashRetractVoteCommand(cmd)
	if record, found, err := uc.Idempotency.Get(ctx, cmd.IdempotencyKey, now); err != nil {
		logger.Error("vote retract idempotency lookup failed",
			"event", "voting_vote_retract_idempotency_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", strings.TrimSpace(cmd.VoteID),
			"user_id", strings.TrimSpace(cmd.UserID),
			"error", err.Error(),
		)
		return err
	} else if found {
		if record.RequestHash != requestHash {
			logger.Warn("vote retract idempotency conflict",
				"event", "voting_vote_retract_idempotency_conflict",
				"module", "campaign-editorial/voting-engine",
				"layer", "application",
				"vote_id", strings.TrimSpace(cmd.VoteID),
				"user_id", strings.TrimSpace(cmd.UserID),
			)
			return domainerrors.ErrIdempotencyConflict
		}
		logger.Info("vote retract replayed",
			"event", "voting_vote_retract_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"vote_id", strings.TrimSpace(cmd.VoteID),
			"user_id", strings.TrimSpace(cmd.UserID),
		)
		return nil
	}

	vote, err := uc.Votes.GetVote(ctx, strings.TrimSpace(cmd.VoteID))
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(vote.UserID), strings.TrimSpace(cmd.UserID)) {
		return domainerrors.ErrConflict
	}
	if vote.Retracted {
		return domainerrors.ErrAlreadyRetracted
	}
	vote.Retracted = true
	vote.UpdatedAt = now
	if err := uc.Votes.SaveVote(ctx, vote); err != nil {
		return err
	}
	if err := uc.appendVoteEvent(ctx, "vote.retracted", vote, now, map[string]any{
		"reason": strings.TrimSpace(cmd.RetractionCause),
	}); err != nil {
		return err
	}
	if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         strings.TrimSpace(cmd.IdempotencyKey),
		RequestHash: requestHash,
		VoteID:      vote.VoteID,
		ExpiresAt:   now.Add(uc.resolveIdempotencyTTL()),
	}); err != nil {
		return err
	}
	logger.Info("vote retracted",
		"event", "voting_vote_retracted",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"vote_id", vote.VoteID,
		"submission_id", vote.SubmissionID,
		"user_id", vote.UserID,
	)
	return nil
}

// ApplyQuarantineAction resolves pending quarantines from moderation workflows.
// "approve" reactivates vote effect; "reject" keeps vote retracted.
func (uc VoteUseCase) ApplyQuarantineAction(ctx context.Context, cmd QuarantineActionCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	action := strings.ToLower(strings.TrimSpace(cmd.Action))
	logger.Info("quarantine action processing started",
		"event", "voting_quarantine_action_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
		"action", action,
		"actor_id", strings.TrimSpace(cmd.ActorID),
	)
	if action != "approve" && action != "reject" {
		logger.Warn("quarantine action validation failed",
			"event", "voting_quarantine_action_validation_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
			"action", action,
			"actor_id", strings.TrimSpace(cmd.ActorID),
		)
		return domainerrors.ErrInvalidQuarantineAction
	}
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		logger.Warn("quarantine action idempotency key missing",
			"event", "voting_quarantine_action_idempotency_missing",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
			"action", action,
			"actor_id", strings.TrimSpace(cmd.ActorID),
		)
		return domainerrors.ErrIdempotencyKeyRequired
	}

	now := uc.now()
	requestHash := hashQuarantineActionCommand(cmd)
	if record, found, err := uc.Idempotency.Get(ctx, cmd.IdempotencyKey, now); err != nil {
		logger.Error("quarantine action idempotency lookup failed",
			"event", "voting_quarantine_action_idempotency_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
			"action", action,
			"actor_id", strings.TrimSpace(cmd.ActorID),
			"error", err.Error(),
		)
		return err
	} else if found {
		if record.RequestHash != requestHash {
			logger.Warn("quarantine action idempotency conflict",
				"event", "voting_quarantine_action_idempotency_conflict",
				"module", "campaign-editorial/voting-engine",
				"layer", "application",
				"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
				"action", action,
				"actor_id", strings.TrimSpace(cmd.ActorID),
			)
			return domainerrors.ErrIdempotencyConflict
		}
		logger.Info("quarantine action replayed",
			"event", "voting_quarantine_action_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"quarantine_id", strings.TrimSpace(cmd.QuarantineID),
			"action", action,
			"actor_id", strings.TrimSpace(cmd.ActorID),
		)
		return nil
	}

	quarantine, err := uc.Votes.GetQuarantine(ctx, cmd.QuarantineID)
	if err != nil {
		return err
	}
	if quarantine.Status != entities.QuarantineStatusPendingReview {
		return domainerrors.ErrQuarantineResolved
	}

	vote, err := uc.Votes.GetVote(ctx, quarantine.VoteID)
	if err != nil {
		return err
	}

	quarantine.UpdatedAt = now
	if action == "approve" {
		quarantine.Status = entities.QuarantineStatusApproved
		vote.Retracted = false
		vote.UpdatedAt = now
	} else {
		quarantine.Status = entities.QuarantineStatusRejected
		vote.Retracted = true
		vote.UpdatedAt = now
	}

	if err := uc.Votes.SaveVote(ctx, vote); err != nil {
		return err
	}
	if err := uc.Votes.SaveQuarantine(ctx, quarantine); err != nil {
		return err
	}

	eventType := "vote.updated"
	reason := "quarantine_approved"
	if action == "reject" {
		eventType = "vote.retracted"
		reason = "quarantine_rejected"
	}
	if err := uc.appendVoteEvent(ctx, eventType, vote, now, map[string]any{
		"reason":        reason,
		"quarantine_id": quarantine.QuarantineID,
		"actioned_by":   strings.TrimSpace(cmd.ActorID),
	}); err != nil {
		return err
	}
	if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         strings.TrimSpace(cmd.IdempotencyKey),
		RequestHash: requestHash,
		VoteID:      vote.VoteID,
		ExpiresAt:   now.Add(uc.resolveIdempotencyTTL()),
	}); err != nil {
		return err
	}
	logger.Info("quarantine action applied",
		"event", "voting_quarantine_action_applied",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"quarantine_id", quarantine.QuarantineID,
		"vote_id", quarantine.VoteID,
		"action", action,
		"actor_id", strings.TrimSpace(cmd.ActorID),
	)
	return nil
}

func (uc VoteUseCase) now() time.Time {
	now := time.Now().UTC()
	if uc.Clock != nil {
		now = uc.Clock.Now().UTC()
	}
	return now
}

func (uc VoteUseCase) resolveIdempotencyTTL() time.Duration {
	if uc.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return uc.IdempotencyTTL
}

func (uc VoteUseCase) resolveWeight(ctx context.Context, userID string) (float64, float64) {
	logger := application.ResolveLogger(uc.Logger)
	score, found, err := uc.Votes.GetReputationScore(ctx, strings.TrimSpace(userID))
	if err != nil {
		logger.Warn("voting reputation lookup failed; applying fallback weight",
			"event", "voting_reputation_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"user_id", strings.TrimSpace(userID),
			"error", err.Error(),
		)
		return 0, 1.0
	}
	if !found {
		logger.Info("voting reputation missing; applying fallback weight",
			"event", "voting_reputation_missing",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"user_id", strings.TrimSpace(userID),
		)
		return 0, 1.0
	}
	switch {
	case score < 50:
		return score, 1.0
	case score < 75:
		return score, 1.5
	case score < 90:
		return score, 2.0
	default:
		return score, 3.0
	}
}

func (uc VoteUseCase) resolveRoundID(
	ctx context.Context,
	campaignID string,
	requestedRoundID string,
	now time.Time,
) (string, error) {
	// Empty round means "campaign-level" vote; otherwise the round must belong to
	// the campaign and be active/not expired.
	roundID := strings.TrimSpace(requestedRoundID)
	if roundID == "" {
		activeRound, found, err := uc.Votes.GetActiveRoundByCampaign(ctx, campaignID)
		if err != nil {
			return "", err
		}
		if !found {
			return "", nil
		}
		return activeRound.RoundID, nil
	}

	round, err := uc.Votes.GetRound(ctx, roundID)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(strings.TrimSpace(round.CampaignID), strings.TrimSpace(campaignID)) {
		return "", domainerrors.ErrInvalidVoteInput
	}
	if round.Status != entities.RoundStatusActive {
		return "", domainerrors.ErrRoundClosed
	}
	if round.EndsAt != nil && round.EndsAt.UTC().Before(now) {
		return "", domainerrors.ErrRoundClosed
	}
	return round.RoundID, nil
}

func (uc VoteUseCase) appendVoteEvent(
	ctx context.Context,
	eventType string,
	vote entities.Vote,
	occurredAt time.Time,
	metadata map[string]any,
) error {
	// Outbox is optional for pure read/test wiring, so nil is treated as no-op.
	if uc.Outbox == nil {
		return nil
	}
	eventID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	data := map[string]any{
		"vote_id":                   vote.VoteID,
		"submission_id":             vote.SubmissionID,
		"campaign_id":               vote.CampaignID,
		"round_id":                  vote.RoundID,
		"user_id":                   vote.UserID,
		"vote_type":                 string(vote.VoteType),
		"weight":                    vote.Weight,
		"retracted":                 vote.Retracted,
		"reputation_score_snapshot": vote.ReputationScoreSnapshot,
		"occurred_at":               occurredAt.Format(time.RFC3339),
	}
	for key, value := range metadata {
		data[key] = value
	}

	envelope, err := newVotingEnvelope(eventID, eventType, vote.SubmissionID, occurredAt, data)
	if err != nil {
		return err
	}
	return uc.Outbox.AppendOutbox(ctx, envelope)
}

// isSubmissionVoteEligible mirrors canonical submission states where voting is
// allowed. Rejected/pending states intentionally fail as not found/eligible.
func isSubmissionVoteEligible(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case
		"approved",
		"verification_period",
		"view_locked",
		"reward_eligible",
		"paid",
		"disputed":
		return true
	default:
		return false
	}
}

func hashCreateVoteCommand(cmd CreateVoteCommand) string {
	payload := map[string]string{
		"user_id":       strings.TrimSpace(cmd.UserID),
		"submission_id": strings.TrimSpace(cmd.SubmissionID),
		"campaign_id":   strings.TrimSpace(cmd.CampaignID),
		"round_id":      strings.TrimSpace(cmd.RoundID),
		"vote_type":     string(cmd.VoteType),
		"ip_address":    strings.TrimSpace(cmd.IPAddress),
		"user_agent":    strings.TrimSpace(cmd.UserAgent),
		"op":            "create_vote",
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func hashRetractVoteCommand(cmd RetractVoteCommand) string {
	payload := map[string]string{
		"vote_id":          strings.TrimSpace(cmd.VoteID),
		"user_id":          strings.TrimSpace(cmd.UserID),
		"retraction_cause": strings.TrimSpace(cmd.RetractionCause),
		"op":               "retract_vote",
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func hashQuarantineActionCommand(cmd QuarantineActionCommand) string {
	payload := map[string]string{
		"quarantine_id": strings.TrimSpace(cmd.QuarantineID),
		"action":        strings.ToLower(strings.TrimSpace(cmd.Action)),
		"actor_id":      strings.TrimSpace(cmd.ActorID),
		"op":            "quarantine_action",
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
