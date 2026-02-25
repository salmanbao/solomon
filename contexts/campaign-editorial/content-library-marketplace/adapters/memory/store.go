package memory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

// Store is an in-memory adapter implementing M09 ports for local runtime and tests.
// It is not intended as production persistence.
type Store struct {
	mu            sync.RWMutex
	clips         map[string]entities.Clip
	claims        map[string]entities.Claim
	downloads     map[string]ports.ClipDownload
	claimsByReqID map[string]string
	idempotency   map[string]ports.IdempotencyRecord
	outbox        map[string]ports.OutboxMessage
	outboxOrder   []string
	outboxSent    map[string]time.Time
	eventDedup    map[string]string
	sequence      uint64
	logger        *slog.Logger
}

// NewStore seeds clip catalog state and initializes claim/idempotency stores.
func NewStore(seedClips []entities.Clip, logger *slog.Logger) *Store {
	clipMap := make(map[string]entities.Clip, len(seedClips))
	for _, clip := range seedClips {
		clipMap[clip.ClipID] = clip
	}
	return &Store{
		clips:         clipMap,
		claims:        make(map[string]entities.Claim),
		downloads:     make(map[string]ports.ClipDownload),
		claimsByReqID: make(map[string]string),
		idempotency:   make(map[string]ports.IdempotencyRecord),
		outbox:        make(map[string]ports.OutboxMessage),
		outboxOrder:   make([]string, 0),
		outboxSent:    make(map[string]time.Time),
		eventDedup:    make(map[string]string),
		logger:        application.ResolveLogger(logger),
	}
}

func (s *Store) ListClips(_ context.Context, filter ports.ClipListFilter) ([]entities.Clip, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []entities.Clip
	niches := make(map[string]struct{}, len(filter.Niches))
	for _, niche := range filter.Niches {
		niches[strings.ToLower(strings.TrimSpace(niche))] = struct{}{}
	}

	for _, clip := range s.clips {
		if filter.Status != "" && clip.Status != filter.Status {
			continue
		}
		if !matchesDurationBucket(clip.DurationSeconds, filter.DurationBucket) {
			continue
		}
		if len(niches) > 0 {
			if _, ok := niches[strings.ToLower(clip.Niche)]; !ok {
				continue
			}
		}
		filtered = append(filtered, clip)
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch filter.Popularity {
		case "views_7d":
			if filtered[i].Views7d == filtered[j].Views7d {
				return filtered[i].ClipID < filtered[j].ClipID
			}
			return filtered[i].Views7d > filtered[j].Views7d
		case "votes_7d":
			if filtered[i].Votes7d == filtered[j].Votes7d {
				return filtered[i].ClipID < filtered[j].ClipID
			}
			return filtered[i].Votes7d > filtered[j].Votes7d
		case "engagement_rate":
			if filtered[i].EngagementRate == filtered[j].EngagementRate {
				return filtered[i].ClipID < filtered[j].ClipID
			}
			return filtered[i].EngagementRate > filtered[j].EngagementRate
		default:
			if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
				return filtered[i].ClipID < filtered[j].ClipID
			}
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		}
	})

	start := decodeCursor(filter.Cursor)
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + filter.Limit
	if filter.Limit <= 0 {
		end = start + 20
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	page := append([]entities.Clip(nil), filtered[start:end]...)
	nextCursor := ""
	if end < len(filtered) {
		nextCursor = encodeCursor(end)
	}

	s.logger.Debug("clips listed from memory store",
		"event", "memory_list_clips",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "adapter",
		"start", start,
		"end", end,
		"total", len(filtered),
	)

	return page, nextCursor, nil
}

func (s *Store) GetClip(_ context.Context, clipID string) (entities.Clip, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clip, ok := s.clips[clipID]
	if !ok {
		return entities.Clip{}, domainerrors.ErrClipNotFound
	}
	return clip, nil
}

func (s *Store) ListClaimsByUser(_ context.Context, userID string) ([]entities.Claim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]entities.Claim, 0)
	for _, claim := range s.claims {
		if claim.UserID == userID {
			result = append(result, claim)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClaimedAt.After(result[j].ClaimedAt)
	})
	return result, nil
}

func (s *Store) ListClaimsByClip(_ context.Context, clipID string) ([]entities.Claim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]entities.Claim, 0)
	for _, claim := range s.claims {
		if claim.ClipID == clipID {
			result = append(result, claim)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClaimedAt.After(result[j].ClaimedAt)
	})
	return result, nil
}

func (s *Store) GetClaim(_ context.Context, claimID string) (entities.Claim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	claim, ok := s.claims[claimID]
	if !ok {
		return entities.Claim{}, domainerrors.ErrClaimNotFound
	}
	return claim, nil
}

func (s *Store) GetClaimByRequestID(_ context.Context, requestID string) (entities.Claim, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	claimID, ok := s.claimsByReqID[requestID]
	if !ok {
		return entities.Claim{}, false, nil
	}
	claim, exists := s.claims[claimID]
	if !exists {
		return entities.Claim{}, false, domainerrors.ErrRepositoryInvariantBroke
	}
	return claim, true, nil
}

func (s *Store) CreateClaimWithOutbox(_ context.Context, claim entities.Claim, event ports.ClaimedEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// A single mutex critical section approximates transactional semantics for
	// tests: claim insert and outbox append succeed/fail together.
	if _, ok := s.claims[claim.ClaimID]; ok {
		return domainerrors.ErrRepositoryInvariantBroke
	}
	if existingClaimID, ok := s.claimsByReqID[claim.RequestID]; ok && existingClaimID != claim.ClaimID {
		return domainerrors.ErrDuplicateRequestID
	}

	s.claims[claim.ClaimID] = claim
	s.claimsByReqID[claim.RequestID] = claim.ClaimID

	envelope := ports.EventEnvelope{
		EventID:          event.EventID,
		EventType:        event.EventType,
		OccurredAt:       event.OccurredAt,
		SourceService:    "content-marketplace-service",
		SchemaVersion:    1,
		PartitionKeyPath: "clip_id",
		PartitionKey:     event.PartitionKey,
	}
	data, err := json.Marshal(map[string]string{
		"claim_id":   event.ClaimID,
		"clip_id":    event.ClipID,
		"user_id":    event.UserID,
		"claim_type": event.ClaimType,
	})
	if err != nil {
		return err
	}
	envelope.Data = data
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	s.outbox[event.EventID] = ports.OutboxMessage{
		OutboxID:     event.EventID,
		EventType:    event.EventType,
		PartitionKey: event.PartitionKey,
		Payload:      payload,
		CreatedAt:    event.OccurredAt,
	}
	s.outboxOrder = append(s.outboxOrder, event.EventID)

	s.logger.Info("claim and outbox persisted in memory store",
		"event", "memory_create_claim_with_outbox",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "adapter",
		"claim_id", claim.ClaimID,
		"clip_id", claim.ClipID,
		"user_id", claim.UserID,
		"outbox_event_id", event.EventID,
	)

	return nil
}

func (s *Store) UpdateClaimStatus(_ context.Context, claimID string, status entities.ClaimStatus, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	claim, ok := s.claims[claimID]
	if !ok {
		return domainerrors.ErrClaimNotFound
	}
	claim.Status = status
	claim.UpdatedAt = updatedAt.UTC()
	s.claims[claimID] = claim
	return nil
}

func (s *Store) ExpireActiveClaims(_ context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expired := 0
	for id, claim := range s.claims {
		if claim.Status != entities.ClaimStatusActive {
			continue
		}
		if !claim.ExpiresAt.UTC().Before(now.UTC()) {
			continue
		}
		claim.Status = entities.ClaimStatusExpired
		claim.UpdatedAt = now.UTC()
		s.claims[id] = claim
		expired++
	}
	return expired, nil
}

func (s *Store) Get(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	// Expired keys are lazily evicted on read.
	if !record.ExpiresAt.IsZero() && now.After(record.ExpiresAt) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyKeyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) ListPendingOutbox(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	messages := make([]ports.OutboxMessage, 0, limit)
	for _, id := range s.outboxOrder {
		if _, sent := s.outboxSent[id]; sent {
			continue
		}
		if msg, ok := s.outbox[id]; ok {
			messages = append(messages, msg)
		}
		if len(messages) >= limit {
			break
		}
	}
	return messages, nil
}

func (s *Store) MarkOutboxSent(_ context.Context, outboxID string, sentAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.outbox[outboxID]; !ok {
		return domainerrors.ErrRepositoryInvariantBroke
	}
	s.outboxSent[outboxID] = sentAt.UTC()
	return nil
}

func (s *Store) ReserveEvent(_ context.Context, eventID string, payloadHash string, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.eventDedup[eventID]; ok {
		if existing != payloadHash {
			return false, domainerrors.ErrIdempotencyKeyConflict
		}
		return true, nil
	}
	s.eventDedup[eventID] = payloadHash
	return false, nil
}

func (s *Store) CountUserClipDownloadsSince(
	_ context.Context,
	userID string,
	clipID string,
	since time.Time,
) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, item := range s.downloads {
		if item.UserID != userID || item.ClipID != clipID {
			continue
		}
		if item.DownloadedAt.UTC().Before(since.UTC()) {
			continue
		}
		count++
	}
	return count, nil
}

func (s *Store) CreateDownload(_ context.Context, download ports.ClipDownload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.downloads[download.DownloadID]; exists {
		return domainerrors.ErrRepositoryInvariantBroke
	}
	s.downloads[download.DownloadID] = ports.ClipDownload{
		DownloadID:   download.DownloadID,
		ClipID:       download.ClipID,
		UserID:       download.UserID,
		IPAddress:    download.IPAddress,
		UserAgent:    download.UserAgent,
		DownloadedAt: download.DownloadedAt.UTC(),
	}
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	value := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("m09-%d", value), nil
}

func (s *Store) OutboxEvents() []ports.OutboxMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]ports.OutboxMessage, 0, len(s.outboxOrder))
	for _, id := range s.outboxOrder {
		if evt, ok := s.outbox[id]; ok {
			events = append(events, evt)
		}
	}
	return events
}

func decodeCursor(cursor string) int {
	if strings.TrimSpace(cursor) == "" {
		return 0
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	index, err := strconv.Atoi(string(raw))
	if err != nil || index < 0 {
		return 0
	}
	return index
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func matchesDurationBucket(durationSeconds int, bucket string) bool {
	switch bucket {
	case "":
		return true
	case "0-15":
		return durationSeconds >= 0 && durationSeconds <= 15
	case "16-30":
		return durationSeconds >= 16 && durationSeconds <= 30
	case "31-60":
		return durationSeconds >= 31 && durationSeconds <= 60
	case "60+":
		return durationSeconds >= 61
	default:
		return false
	}
}
