package workers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

const (
	submissionApprovedTopic = "submission.approved"
	submissionRejectedTopic = "submission.rejected"
	defaultSubmissionCG     = "voting-engine-submission-cg"
)

// SubmissionLifecycleConsumer reacts to submission status changes that impact
// vote effect (notably rejection-triggered vote retractions).
type SubmissionLifecycleConsumer struct {
	Subscriber    ports.EventSubscriber
	Dedup         ports.EventDedupStore
	Votes         ports.VoteRepository
	Outbox        ports.OutboxWriter
	Clock         ports.Clock
	IDGen         ports.IDGenerator
	ConsumerGroup string
	DedupTTL      time.Duration
	Logger        *slog.Logger
}

// Start subscribes M08 to submission lifecycle events with dedupe semantics.
func (c SubmissionLifecycleConsumer) Start(ctx context.Context) error {
	logger := application.ResolveLogger(c.Logger)
	group := strings.TrimSpace(c.ConsumerGroup)
	if group == "" {
		group = defaultSubmissionCG
	}
	logger.Info("submission consumer starting subscriptions",
		"event", "voting_submission_consumer_starting",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"consumer_group", group,
	)
	if err := c.Subscriber.Subscribe(ctx, submissionApprovedTopic, group, c.handleSubmissionApproved); err != nil {
		logger.Error("submission consumer subscribe failed",
			"event", "voting_submission_consumer_subscribe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"topic", submissionApprovedTopic,
			"consumer_group", group,
			"error", err.Error(),
		)
		return err
	}
	if err := c.Subscriber.Subscribe(ctx, submissionRejectedTopic, group, c.handleSubmissionRejected); err != nil {
		logger.Error("submission consumer subscribe failed",
			"event", "voting_submission_consumer_subscribe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"topic", submissionRejectedTopic,
			"consumer_group", group,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("submission consumer subscriptions active",
		"event", "voting_submission_consumer_started",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"consumer_group", group,
	)
	return nil
}

func (c SubmissionLifecycleConsumer) handleSubmissionApproved(ctx context.Context, event ports.EventEnvelope) error {
	// Approved submissions require no state mutation in M08 at present.
	logger := application.ResolveLogger(c.Logger)
	if alreadyProcessed, err := c.reserveEvent(ctx, event); err != nil {
		return err
	} else if alreadyProcessed {
		logger.Debug("submission.approved replay skipped",
			"event", "voting_submission_approved_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}

	var payload struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("submission.approved payload decode failed",
			"event", "voting_submission_approved_decode_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("submission.approved consumed",
		"event", "voting_submission_approved_consumed",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"event_id", event.EventID,
		"submission_id", strings.TrimSpace(payload.SubmissionID),
	)
	return nil
}

func (c SubmissionLifecycleConsumer) handleSubmissionRejected(ctx context.Context, event ports.EventEnvelope) error {
	// Rejection retracts non-retracted votes and emits vote.retracted per vote.
	logger := application.ResolveLogger(c.Logger)
	if alreadyProcessed, err := c.reserveEvent(ctx, event); err != nil {
		return err
	} else if alreadyProcessed {
		logger.Debug("submission.rejected replay skipped",
			"event", "voting_submission_rejected_replayed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
		)
		return nil
	}

	var payload struct {
		SubmissionID string `json:"submission_id"`
		CampaignID   string `json:"campaign_id"`
	}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		logger.Error("submission.rejected payload decode failed",
			"event", "voting_submission_rejected_decode_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"error", err.Error(),
		)
		return err
	}
	now := c.now()
	retractedVotes, err := c.Votes.RetractVotesBySubmission(ctx, payload.SubmissionID, now)
	if err != nil {
		logger.Error("submission.rejected retract votes failed",
			"event", "voting_submission_rejected_retract_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"submission_id", strings.TrimSpace(payload.SubmissionID),
			"error", err.Error(),
		)
		return err
	}

	for _, vote := range retractedVotes {
		eventID, err := c.IDGen.NewID(ctx)
		if err != nil {
			logger.Error("submission.rejected event id generation failed",
				"event", "voting_submission_rejected_event_id_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"vote_id", vote.VoteID,
				"error", err.Error(),
			)
			return err
		}
		envelope, err := newVotingEnvelope(
			eventID,
			"vote.retracted",
			vote.SubmissionID,
			"submission_id",
			now,
			map[string]any{
				"vote_id":       vote.VoteID,
				"submission_id": vote.SubmissionID,
				"campaign_id":   vote.CampaignID,
				"user_id":       vote.UserID,
				"vote_type":     string(vote.VoteType),
				"weight":        vote.Weight,
				"retracted":     true,
				"reason":        "submission_rejected",
				"occurred_at":   now.Format(time.RFC3339),
			},
		)
		if err != nil {
			logger.Error("submission.rejected envelope build failed",
				"event", "voting_submission_rejected_envelope_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"vote_id", vote.VoteID,
				"error", err.Error(),
			)
			return err
		}
		if err := c.Outbox.AppendOutbox(ctx, envelope); err != nil {
			logger.Error("submission.rejected outbox append failed",
				"event", "voting_submission_rejected_outbox_failed",
				"module", "campaign-editorial/voting-engine",
				"layer", "worker",
				"event_id", event.EventID,
				"vote_id", vote.VoteID,
				"error", err.Error(),
			)
			return err
		}
	}

	logger.Info("submission.rejected consumed",
		"event", "voting_submission_rejected_consumed",
		"module", "campaign-editorial/voting-engine",
		"layer", "worker",
		"event_id", event.EventID,
		"submission_id", strings.TrimSpace(payload.SubmissionID),
		"retracted_votes", len(retractedVotes),
	)
	return nil
}

func (c SubmissionLifecycleConsumer) reserveEvent(ctx context.Context, event ports.EventEnvelope) (bool, error) {
	// ReserveEvent is used as dedupe gate for at-least-once delivery semantics.
	logger := application.ResolveLogger(c.Logger)
	alreadyProcessed, err := c.Dedup.ReserveEvent(ctx, event.EventID, hashPayload(event.Data), c.now().Add(c.dedupTTL()))
	if err != nil {
		logger.Error("submission event dedupe failed",
			"event", "voting_submission_event_dedupe_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "worker",
			"event_id", event.EventID,
			"event_type", event.EventType,
			"error", err.Error(),
		)
		return false, err
	}
	return alreadyProcessed, nil
}

func (c SubmissionLifecycleConsumer) now() time.Time {
	now := time.Now().UTC()
	if c.Clock != nil {
		now = c.Clock.Now().UTC()
	}
	return now
}

func (c SubmissionLifecycleConsumer) dedupTTL() time.Duration {
	if c.DedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return c.DedupTTL
}
