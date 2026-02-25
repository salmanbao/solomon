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

type CreateVoteCommand struct {
	UserID         string
	IdempotencyKey string
	SubmissionID   string
	CampaignID     string
	VoteType       entities.VoteType
}

type CreateVoteResult struct {
	Vote     entities.Vote
	Replayed bool
}

type VoteUseCase struct {
	Votes          ports.VoteRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func (uc VoteUseCase) CreateVote(ctx context.Context, cmd CreateVoteCommand) (CreateVoteResult, error) {
	logger := application.ResolveLogger(uc.Logger)
	if strings.TrimSpace(cmd.UserID) == "" ||
		strings.TrimSpace(cmd.SubmissionID) == "" ||
		strings.TrimSpace(cmd.CampaignID) == "" ||
		(cmd.VoteType != entities.VoteTypeUpvote && cmd.VoteType != entities.VoteTypeDownvote) {
		return CreateVoteResult{}, domainerrors.ErrInvalidVoteInput
	}
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return CreateVoteResult{}, domainerrors.ErrInvalidVoteInput
	}

	now := uc.Clock.Now().UTC()
	requestHash := hashVote(cmd)
	if record, found, err := uc.Idempotency.Get(ctx, cmd.IdempotencyKey, now); err != nil {
		return CreateVoteResult{}, err
	} else if found {
		if record.RequestHash != requestHash {
			return CreateVoteResult{}, domainerrors.ErrConflict
		}
		vote, err := uc.Votes.GetVote(ctx, record.VoteID)
		if err != nil {
			return CreateVoteResult{}, err
		}
		return CreateVoteResult{Vote: vote, Replayed: true}, nil
	}

	weight := 1.0
	if existing, found, err := uc.Votes.GetVoteByIdentity(ctx, cmd.SubmissionID, cmd.UserID); err != nil {
		return CreateVoteResult{}, err
	} else if found {
		existing.VoteType = cmd.VoteType
		existing.Retracted = false
		existing.UpdatedAt = now
		if err := uc.Votes.SaveVote(ctx, existing); err != nil {
			return CreateVoteResult{}, err
		}
		if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
			Key:         cmd.IdempotencyKey,
			RequestHash: requestHash,
			VoteID:      existing.VoteID,
			ExpiresAt:   now.Add(uc.IdempotencyTTL),
		}); err != nil {
			return CreateVoteResult{}, err
		}
		return CreateVoteResult{Vote: existing}, nil
	}

	voteID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return CreateVoteResult{}, err
	}
	vote := entities.Vote{
		VoteID:       voteID,
		SubmissionID: cmd.SubmissionID,
		CampaignID:   cmd.CampaignID,
		UserID:       cmd.UserID,
		VoteType:     cmd.VoteType,
		Weight:       weight,
		Retracted:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.Votes.SaveVote(ctx, vote); err != nil {
		return CreateVoteResult{}, err
	}
	if err := uc.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         cmd.IdempotencyKey,
		RequestHash: requestHash,
		VoteID:      vote.VoteID,
		ExpiresAt:   now.Add(uc.IdempotencyTTL),
	}); err != nil {
		return CreateVoteResult{}, err
	}

	logger.Info("vote created",
		"event", "vote_created",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"vote_id", vote.VoteID,
		"submission_id", vote.SubmissionID,
	)
	return CreateVoteResult{Vote: vote}, nil
}

type RetractVoteCommand struct {
	VoteID string
	UserID string
}

func (uc VoteUseCase) RetractVote(ctx context.Context, cmd RetractVoteCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	vote, err := uc.Votes.GetVote(ctx, strings.TrimSpace(cmd.VoteID))
	if err != nil {
		return err
	}
	if vote.UserID != strings.TrimSpace(cmd.UserID) {
		return domainerrors.ErrConflict
	}
	if vote.Retracted {
		return domainerrors.ErrAlreadyRetracted
	}
	vote.Retracted = true
	vote.UpdatedAt = uc.Clock.Now().UTC()
	if err := uc.Votes.SaveVote(ctx, vote); err != nil {
		return err
	}
	logger.Info("vote retracted",
		"event", "vote_retracted",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"vote_id", vote.VoteID,
		"submission_id", vote.SubmissionID,
	)
	return nil
}

func hashVote(cmd CreateVoteCommand) string {
	payload := map[string]string{
		"user_id":       strings.TrimSpace(cmd.UserID),
		"submission_id": strings.TrimSpace(cmd.SubmissionID),
		"campaign_id":   strings.TrimSpace(cmd.CampaignID),
		"vote_type":     string(cmd.VoteType),
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
