package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/moderation-safety/moderation-service/domain/errors"
	"solomon/contexts/moderation-safety/moderation-service/ports"
)

type Service struct {
	Repo             ports.Repository
	Idempotency      ports.IdempotencyStore
	SubmissionClient ports.SubmissionDecisionClient
	Clock            ports.Clock
	IdempotencyTTL   time.Duration
	Logger           *slog.Logger
}

func (s Service) ListQueue(ctx context.Context, filter ports.QueueFilter) ([]ports.QueueItem, error) {
	filter.Status = strings.TrimSpace(strings.ToLower(filter.Status))
	if filter.Status != "" {
		switch filter.Status {
		case "pending", "approved", "rejected", "flagged", "escalated":
		default:
			return nil, domainerrors.ErrInvalidRequest
		}
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		return nil, domainerrors.ErrInvalidRequest
	}
	return s.Repo.ListQueue(ctx, filter)
}

func (s Service) Approve(ctx context.Context, idempotencyKey string, moderatorID string, input ports.ModerationActionInput) (ports.DecisionRecord, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.Severity = "high"
	return s.runDecision(ctx, idempotencyKey, moderatorID, input, "approved", func() error {
		if s.SubmissionClient == nil {
			return domainerrors.ErrDependencyUnavailable
		}
		return s.SubmissionClient.ApproveSubmission(ctx, input.SubmissionID, moderatorID, input.Reason)
	})
}

func (s Service) Reject(ctx context.Context, idempotencyKey string, moderatorID string, input ports.ModerationActionInput) (ports.DecisionRecord, error) {
	input.Reason = strings.TrimSpace(strings.ToLower(input.Reason))
	input.Severity = "high"
	return s.runDecision(ctx, idempotencyKey, moderatorID, input, "rejected", func() error {
		if s.SubmissionClient == nil {
			return domainerrors.ErrDependencyUnavailable
		}
		return s.SubmissionClient.RejectSubmission(ctx, input.SubmissionID, moderatorID, input.Reason, input.Notes)
	})
}

func (s Service) Flag(ctx context.Context, idempotencyKey string, moderatorID string, input ports.ModerationActionInput) (ports.DecisionRecord, error) {
	input.Severity = strings.TrimSpace(strings.ToLower(input.Severity))
	if input.Severity == "" {
		input.Severity = "medium"
	}
	return s.runDecision(ctx, idempotencyKey, moderatorID, input, "flagged", nil)
}

func (s Service) runDecision(
	ctx context.Context,
	idempotencyKey string,
	moderatorID string,
	input ports.ModerationActionInput,
	action string,
	beforePersist func() error,
) (ports.DecisionRecord, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	moderatorID = strings.TrimSpace(moderatorID)
	input.SubmissionID = strings.TrimSpace(input.SubmissionID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.Notes = strings.TrimSpace(input.Notes)

	if idempotencyKey == "" {
		return ports.DecisionRecord{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if moderatorID == "" || input.SubmissionID == "" || input.CampaignID == "" || action == "" {
		return ports.DecisionRecord{}, domainerrors.ErrInvalidRequest
	}
	if action == "rejected" && input.Reason == "" {
		return ports.DecisionRecord{}, domainerrors.ErrInvalidRequest
	}
	if action == "flagged" && input.Reason == "" {
		return ports.DecisionRecord{}, domainerrors.ErrInvalidRequest
	}

	requestHash := hashStrings(moderatorID, input.SubmissionID, input.CampaignID, action, input.Reason, input.Notes, input.Severity)
	var output ports.DecisionRecord
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			if beforePersist != nil {
				if err := beforePersist(); err != nil {
					return nil, err
				}
			}
			record, err := s.Repo.RecordDecision(ctx, ports.DecisionRecord{
				SubmissionID: input.SubmissionID,
				CampaignID:   input.CampaignID,
				ModeratorID:  moderatorID,
				Action:       action,
				Reason:       input.Reason,
				Notes:        input.Notes,
				Severity:     input.Severity,
			}, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(record)
		},
	)
	return output, err
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}
	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}
	resolveLogger(s.Logger).Debug("moderation idempotent mutation committed",
		"event", "moderation_idempotent_mutation_committed",
		"module", "moderation-safety/moderation-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
