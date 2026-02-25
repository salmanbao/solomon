package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/voting-engine/domain/errors"
	"solomon/contexts/campaign-editorial/voting-engine/ports"

	"github.com/google/uuid"
)

type outboxRecord struct {
	message   ports.OutboxMessage
	published bool
}

type dedupRecord struct {
	payloadHash string
	expiresAt   time.Time
}

type Store struct {
	mu sync.RWMutex

	votes       map[string]entities.Vote
	idempotency map[string]ports.IdempotencyRecord
	outbox      map[string]outboxRecord
	eventDedup  map[string]dedupRecord

	submissions map[string]ports.SubmissionProjection
	campaigns   map[string]ports.CampaignProjection
	reputation  map[string]float64
	rounds      map[string]entities.VotingRound
	quarantine  map[string]entities.VoteQuarantine
}

func NewStore(seed []entities.Vote) *Store {
	votes := make(map[string]entities.Vote, len(seed))
	for _, vote := range seed {
		votes[vote.VoteID] = vote
	}
	return &Store{
		votes:       votes,
		idempotency: make(map[string]ports.IdempotencyRecord),
		outbox:      make(map[string]outboxRecord),
		eventDedup:  make(map[string]dedupRecord),
		submissions: make(map[string]ports.SubmissionProjection),
		campaigns:   make(map[string]ports.CampaignProjection),
		reputation:  make(map[string]float64),
		rounds:      make(map[string]entities.VotingRound),
		quarantine:  make(map[string]entities.VoteQuarantine),
	}
}

func (s *Store) SetSubmission(submission ports.SubmissionProjection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.submissions[strings.TrimSpace(submission.SubmissionID)] = ports.SubmissionProjection{
		SubmissionID: strings.TrimSpace(submission.SubmissionID),
		CampaignID:   strings.TrimSpace(submission.CampaignID),
		CreatorID:    strings.TrimSpace(submission.CreatorID),
		Status:       strings.TrimSpace(submission.Status),
	}
}

func (s *Store) SetCampaign(campaign ports.CampaignProjection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.campaigns[strings.TrimSpace(campaign.CampaignID)] = ports.CampaignProjection{
		CampaignID: strings.TrimSpace(campaign.CampaignID),
		Status:     strings.TrimSpace(campaign.Status),
	}
}

func (s *Store) SetReputationScore(userID string, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reputation[strings.TrimSpace(userID)] = score
}

func (s *Store) SetRound(round entities.VotingRound) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rounds[strings.TrimSpace(round.RoundID)] = round
}

func (s *Store) SetQuarantine(record entities.VoteQuarantine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quarantine[strings.TrimSpace(record.QuarantineID)] = record
}

func (s *Store) SaveVote(_ context.Context, vote entities.Vote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.votes[strings.TrimSpace(vote.VoteID)] = vote
	return nil
}

func (s *Store) GetVote(_ context.Context, voteID string) (entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vote, ok := s.votes[strings.TrimSpace(voteID)]
	if !ok {
		return entities.Vote{}, domainerrors.ErrVoteNotFound
	}
	return vote, nil
}

func (s *Store) GetVoteByIdentity(
	_ context.Context,
	submissionID string,
	userID string,
	roundID string,
) (entities.Vote, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	submissionID = strings.TrimSpace(submissionID)
	userID = strings.TrimSpace(userID)
	roundID = strings.TrimSpace(roundID)

	for _, vote := range s.votes {
		if vote.SubmissionID != submissionID || vote.UserID != userID {
			continue
		}
		if strings.TrimSpace(vote.RoundID) != roundID {
			continue
		}
		return vote, true, nil
	}
	return entities.Vote{}, false, nil
}

func (s *Store) ListVotesBySubmission(_ context.Context, submissionID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.Vote, 0)
	for _, vote := range s.votes {
		if vote.SubmissionID == strings.TrimSpace(submissionID) {
			items = append(items, vote)
		}
	}
	sortVotesByCreation(items)
	return items, nil
}

func (s *Store) ListVotesByCampaign(_ context.Context, campaignID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.Vote, 0)
	campaignID = strings.TrimSpace(campaignID)
	for _, vote := range s.votes {
		if campaignID == "" || vote.CampaignID == campaignID {
			items = append(items, vote)
		}
	}
	sortVotesByCreation(items)
	return items, nil
}

func (s *Store) ListVotesByRound(_ context.Context, roundID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.Vote, 0)
	for _, vote := range s.votes {
		if strings.TrimSpace(vote.RoundID) == strings.TrimSpace(roundID) {
			items = append(items, vote)
		}
	}
	sortVotesByCreation(items)
	return items, nil
}

func (s *Store) ListVotesByCreator(_ context.Context, creatorID string) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.Vote, 0)
	for _, vote := range s.votes {
		submission, ok := s.submissions[vote.SubmissionID]
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(submission.CreatorID), strings.TrimSpace(creatorID)) {
			items = append(items, vote)
		}
	}
	sortVotesByCreation(items)
	return items, nil
}

func (s *Store) ListVotes(_ context.Context) ([]entities.Vote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.Vote, 0, len(s.votes))
	for _, vote := range s.votes {
		items = append(items, vote)
	}
	sortVotesByCreation(items)
	return items, nil
}

func (s *Store) GetSubmission(_ context.Context, submissionID string) (ports.SubmissionProjection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.submissions[strings.TrimSpace(submissionID)]
	if !ok {
		return ports.SubmissionProjection{}, domainerrors.ErrSubmissionNotFound
	}
	return item, nil
}

func (s *Store) GetCampaign(_ context.Context, campaignID string) (ports.CampaignProjection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.campaigns[strings.TrimSpace(campaignID)]
	if !ok {
		return ports.CampaignProjection{}, domainerrors.ErrCampaignNotFound
	}
	return item, nil
}

func (s *Store) GetReputationScore(_ context.Context, userID string) (float64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	score, ok := s.reputation[strings.TrimSpace(userID)]
	if !ok {
		return 0, false, nil
	}
	return score, true, nil
}

func (s *Store) GetRound(_ context.Context, roundID string) (entities.VotingRound, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	round, ok := s.rounds[strings.TrimSpace(roundID)]
	if !ok {
		return entities.VotingRound{}, domainerrors.ErrRoundNotFound
	}
	return round, nil
}

func (s *Store) GetActiveRoundByCampaign(_ context.Context, campaignID string) (entities.VotingRound, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	for _, round := range s.rounds {
		if !strings.EqualFold(strings.TrimSpace(round.CampaignID), strings.TrimSpace(campaignID)) {
			continue
		}
		if round.Status != entities.RoundStatusActive {
			continue
		}
		if round.EndsAt != nil && round.EndsAt.UTC().Before(now) {
			continue
		}
		return round, true, nil
	}
	return entities.VotingRound{}, false, nil
}

func (s *Store) TransitionRoundsForCampaign(
	_ context.Context,
	campaignID string,
	toStatus entities.RoundStatus,
	updatedAt time.Time,
) ([]entities.VotingRound, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]entities.VotingRound, 0)
	for key, round := range s.rounds {
		if !strings.EqualFold(strings.TrimSpace(round.CampaignID), strings.TrimSpace(campaignID)) {
			continue
		}
		if round.Status == toStatus {
			continue
		}
		if toStatus == entities.RoundStatusClosingSoon && round.Status != entities.RoundStatusActive {
			continue
		}
		if toStatus == entities.RoundStatusClosed &&
			round.Status != entities.RoundStatusActive &&
			round.Status != entities.RoundStatusClosingSoon {
			continue
		}
		round.Status = toStatus
		round.UpdatedAt = updatedAt.UTC()
		if toStatus == entities.RoundStatusClosed && round.EndsAt == nil {
			endedAt := updatedAt.UTC()
			round.EndsAt = &endedAt
		}
		s.rounds[key] = round
		items = append(items, round)
	}
	return items, nil
}

func (s *Store) GetQuarantine(_ context.Context, quarantineID string) (entities.VoteQuarantine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.quarantine[strings.TrimSpace(quarantineID)]
	if !ok {
		return entities.VoteQuarantine{}, domainerrors.ErrQuarantineNotFound
	}
	return row, nil
}

func (s *Store) SaveQuarantine(_ context.Context, quarantine entities.VoteQuarantine) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quarantine[strings.TrimSpace(quarantine.QuarantineID)] = quarantine
	return nil
}

func (s *Store) ListQuarantines(_ context.Context) ([]entities.VoteQuarantine, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]entities.VoteQuarantine, 0, len(s.quarantine))
	for _, row := range s.quarantine {
		items = append(items, row)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) RetractVotesBySubmission(
	_ context.Context,
	submissionID string,
	updatedAt time.Time,
) ([]entities.Vote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := make([]entities.Vote, 0)
	for key, vote := range s.votes {
		if vote.SubmissionID != strings.TrimSpace(submissionID) {
			continue
		}
		if vote.Retracted {
			continue
		}
		vote.Retracted = true
		vote.UpdatedAt = updatedAt.UTC()
		s.votes[key] = vote
		updated = append(updated, vote)
	}
	return updated, nil
}

func (s *Store) Get(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key = strings.TrimSpace(key)
	record, exists := s.idempotency[key]
	if !exists {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.After(now.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(record.Key)
	existing, exists := s.idempotency[key]
	if exists {
		if existing.RequestHash != record.RequestHash || existing.VoteID != record.VoteID {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: strings.TrimSpace(record.RequestHash),
		VoteID:      strings.TrimSpace(record.VoteID),
		ExpiresAt:   record.ExpiresAt.UTC(),
	}
	return nil
}

func (s *Store) AppendOutbox(_ context.Context, envelope ports.EventEnvelope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	outboxID := strings.TrimSpace(envelope.EventID)
	if outboxID == "" {
		outboxID = uuid.NewString()
	}
	if existing, ok := s.outbox[outboxID]; ok {
		if !bytes.Equal(existing.message.Payload, payload) {
			return domainerrors.ErrConflict
		}
		return nil
	}
	createdAt := envelope.OccurredAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	s.outbox[outboxID] = outboxRecord{
		message: ports.OutboxMessage{
			OutboxID:     outboxID,
			EventType:    strings.TrimSpace(envelope.EventType),
			PartitionKey: strings.TrimSpace(envelope.PartitionKey),
			Payload:      payload,
			CreatedAt:    createdAt,
		},
	}
	return nil
}

func (s *Store) ListPendingOutbox(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	items := make([]ports.OutboxMessage, 0, len(s.outbox))
	for _, row := range s.outbox {
		if row.published {
			continue
		}
		items = append(items, row.message)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) MarkOutboxPublished(_ context.Context, outboxID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.outbox[strings.TrimSpace(outboxID)]
	if !ok {
		return domainerrors.ErrConflict
	}
	row.published = true
	s.outbox[strings.TrimSpace(outboxID)] = row
	return nil
}

func (s *Store) ReserveEvent(
	_ context.Context,
	eventID string,
	payloadHash string,
	expiresAt time.Time,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(eventID)
	existing, ok := s.eventDedup[key]
	if ok {
		if !existing.expiresAt.IsZero() && time.Now().UTC().After(existing.expiresAt.UTC()) {
			delete(s.eventDedup, key)
		} else {
			if existing.payloadHash != strings.TrimSpace(payloadHash) {
				return false, domainerrors.ErrConflict
			}
			return true, nil
		}
	}

	s.eventDedup[key] = dedupRecord{
		payloadHash: strings.TrimSpace(payloadHash),
		expiresAt:   expiresAt.UTC(),
	}
	return false, nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}

func sortVotesByCreation(items []entities.Vote) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}
