package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"

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

type campaignProjection struct {
	id               string
	status           string
	allowedPlatforms []string
	ratePer1KViews   float64
}

type Store struct {
	mu sync.RWMutex

	submissions map[string]entities.Submission
	reports     map[string]entities.SubmissionReport
	flags       map[string]entities.SubmissionFlag
	audits      map[string]entities.SubmissionAudit
	operations  map[string]entities.BulkSubmissionOperation
	snapshots   map[string]entities.ViewSnapshot

	campaigns   map[string]campaignProjection
	idempotency map[string]ports.IdempotencyRecord
	outbox      map[string]outboxRecord
	eventDedup  map[string]dedupRecord
}

func NewStore(seed []entities.Submission) *Store {
	submissions := make(map[string]entities.Submission, len(seed))
	for _, item := range seed {
		submissions[item.SubmissionID] = item
	}
	return &Store{
		submissions: submissions,
		reports:     make(map[string]entities.SubmissionReport),
		flags:       make(map[string]entities.SubmissionFlag),
		audits:      make(map[string]entities.SubmissionAudit),
		operations:  make(map[string]entities.BulkSubmissionOperation),
		snapshots:   make(map[string]entities.ViewSnapshot),
		campaigns:   make(map[string]campaignProjection),
		idempotency: make(map[string]ports.IdempotencyRecord),
		outbox:      make(map[string]outboxRecord),
		eventDedup:  make(map[string]dedupRecord),
	}
}

func (s *Store) SetCampaign(campaignID string, status string, allowedPlatforms []string, ratePer1KViews float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(campaignID)
	s.campaigns[id] = campaignProjection{
		id:               id,
		status:           strings.TrimSpace(status),
		allowedPlatforms: append([]string(nil), allowedPlatforms...),
		ratePer1KViews:   ratePer1KViews,
	}
}

func (s *Store) GetCampaignForSubmission(_ context.Context, campaignID string) (ports.CampaignForSubmission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row, ok := s.campaigns[strings.TrimSpace(campaignID)]
	if !ok {
		return ports.CampaignForSubmission{}, domainerrors.ErrCampaignNotFound
	}
	return ports.CampaignForSubmission{
		CampaignID:       row.id,
		Status:           row.status,
		AllowedPlatforms: append([]string(nil), row.allowedPlatforms...),
		RatePer1KViews:   row.ratePer1KViews,
	}, nil
}

func (s *Store) CreateSubmission(_ context.Context, submission entities.Submission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.submissions {
		if existing.CampaignID != submission.CampaignID ||
			existing.CreatorID != submission.CreatorID ||
			existing.PostURL != submission.PostURL ||
			existing.Status == entities.SubmissionStatusCancelled {
			continue
		}
		if !existing.CreatedAt.IsZero() {
			if submission.CreatedAt.Sub(existing.CreatedAt) > 24*time.Hour {
				continue
			}
		}
		return domainerrors.ErrDuplicateSubmission
	}
	s.submissions[submission.SubmissionID] = submission
	return nil
}

func (s *Store) UpdateSubmission(_ context.Context, submission entities.Submission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.submissions[submission.SubmissionID]; !exists {
		return domainerrors.ErrSubmissionNotFound
	}
	s.submissions[submission.SubmissionID] = submission
	return nil
}

func (s *Store) GetSubmission(_ context.Context, submissionID string) (entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.submissions[strings.TrimSpace(submissionID)]
	if !exists {
		return entities.Submission{}, domainerrors.ErrSubmissionNotFound
	}
	return item, nil
}

func (s *Store) ListSubmissions(_ context.Context, filter ports.SubmissionFilter) ([]entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.Submission, 0, len(s.submissions))
	for _, item := range s.submissions {
		if strings.TrimSpace(filter.CreatorID) != "" && item.CreatorID != strings.TrimSpace(filter.CreatorID) {
			continue
		}
		if strings.TrimSpace(filter.CampaignID) != "" && item.CampaignID != strings.TrimSpace(filter.CampaignID) {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Store) AddReport(_ context.Context, report entities.SubmissionReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.reports {
		if strings.TrimSpace(existing.SubmissionID) == strings.TrimSpace(report.SubmissionID) &&
			strings.TrimSpace(existing.ReportedByID) != "" &&
			strings.TrimSpace(existing.ReportedByID) == strings.TrimSpace(report.ReportedByID) {
			return domainerrors.ErrAlreadyReported
		}
	}

	s.reports[report.ReportID] = report
	return nil
}

func (s *Store) AddFlag(_ context.Context, flag entities.SubmissionFlag) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flags[flag.FlagID] = flag
	return nil
}

func (s *Store) AddAudit(_ context.Context, audit entities.SubmissionAudit) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.audits[audit.AuditID] = audit
	return nil
}

func (s *Store) AddBulkOperation(_ context.Context, operation entities.BulkSubmissionOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.operations[operation.OperationID] = operation
	return nil
}

func (s *Store) AddViewSnapshot(_ context.Context, snapshot entities.ViewSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.snapshots[snapshot.SnapshotID] = snapshot
	return nil
}

func (s *Store) GetRecord(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[strings.TrimSpace(key)]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, strings.TrimSpace(key))
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) PutRecord(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(record.Key)
	if existing, ok := s.idempotency[key]; ok {
		if existing.RequestHash != record.RequestHash || !bytes.Equal(existing.ResponsePayload, record.ResponsePayload) {
			return domainerrors.ErrIdempotencyKeyConflict
		}
		return nil
	}
	s.idempotency[key] = ports.IdempotencyRecord{
		Key:             key,
		RequestHash:     record.RequestHash,
		ResponsePayload: append([]byte(nil), record.ResponsePayload...),
		ExpiresAt:       record.ExpiresAt.UTC(),
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
			return domainerrors.ErrIdempotencyKeyConflict
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
		return domainerrors.ErrInvalidSubmissionInput
	}
	row.published = true
	s.outbox[strings.TrimSpace(outboxID)] = row
	return nil
}

func (s *Store) ReserveEvent(_ context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := strings.TrimSpace(eventID)
	row, ok := s.eventDedup[key]
	if ok {
		if !row.expiresAt.IsZero() && time.Now().UTC().After(row.expiresAt.UTC()) {
			delete(s.eventDedup, key)
		} else {
			if row.payloadHash != strings.TrimSpace(payloadHash) {
				return false, domainerrors.ErrIdempotencyKeyConflict
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

func (s *Store) ListPendingAutoApprove(
	_ context.Context,
	threshold time.Time,
	limit int,
) ([]entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	items := make([]entities.Submission, 0)
	for _, submission := range s.submissions {
		if submission.Status != entities.SubmissionStatusPending {
			continue
		}
		if submission.ReportedCount > 0 {
			continue
		}
		if submission.CreatedAt.After(threshold.UTC()) {
			continue
		}
		items = append(items, submission)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) ListDueViewLock(
	_ context.Context,
	threshold time.Time,
	limit int,
) ([]entities.Submission, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	items := make([]entities.Submission, 0)
	for _, submission := range s.submissions {
		if submission.LockedViews != nil {
			continue
		}
		if submission.Status != entities.SubmissionStatusApproved &&
			submission.Status != entities.SubmissionStatusVerification {
			continue
		}
		if submission.VerificationWindowEnd == nil || submission.VerificationWindowEnd.After(threshold.UTC()) {
			continue
		}
		items = append(items, submission)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].VerificationWindowEnd.Before(*items[j].VerificationWindowEnd)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
