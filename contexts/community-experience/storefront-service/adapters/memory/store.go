package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	"solomon/contexts/community-experience/storefront-service/ports"
)

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

type eventDedupRecord struct {
	RequestHash string
	ExpiresAt   time.Time
}

type Store struct {
	mu sync.RWMutex

	storefrontsByID           map[string]ports.Storefront
	storefrontIDBySlug        map[string]string
	storefrontIDByCreator     map[string]string
	productIDsByStorefront    map[string]map[string]struct{}
	catalogSyncedByStorefront map[string]bool
	productEventDedupByID     map[string]eventDedupRecord
	subscriptionActiveByUser  map[string]bool
	validProductIDs           map[string]struct{}
	reportDedupAt             map[string]time.Time

	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	return &Store{
		storefrontsByID:           make(map[string]ports.Storefront),
		storefrontIDBySlug:        make(map[string]string),
		storefrontIDByCreator:     make(map[string]string),
		productIDsByStorefront:    make(map[string]map[string]struct{}),
		catalogSyncedByStorefront: make(map[string]bool),
		productEventDedupByID:     make(map[string]eventDedupRecord),
		subscriptionActiveByUser: map[string]bool{
			"creator_1": true,
		},
		validProductIDs: map[string]struct{}{
			"prod_001": {},
			"prod_002": {},
			"prod_003": {},
		},
		reportDedupAt: make(map[string]time.Time),
		idempotency:   make(map[string]ports.IdempotencyRecord),
		sequence:      1,
	}
}

func (s *Store) CreateStorefront(
	ctx context.Context,
	actorUserID string,
	input ports.CreateStorefrontInput,
	now time.Time,
) (ports.Storefront, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	actorUserID = strings.TrimSpace(actorUserID)
	if actorUserID == "" ||
		strings.TrimSpace(input.DisplayName) == "" ||
		len(strings.TrimSpace(input.DisplayName)) > 100 ||
		!ports.IsValidCategory(strings.TrimSpace(input.Category)) {
		return ports.Storefront{}, domainerrors.ErrInvalidRequest
	}
	if _, exists := s.storefrontIDByCreator[actorUserID]; exists {
		return ports.Storefront{}, domainerrors.ErrConflict
	}

	slug := buildSlug(actorUserID)
	if _, exists := s.storefrontIDBySlug[slug]; exists {
		return ports.Storefront{}, domainerrors.ErrConflict
	}

	now = now.UTC()
	id := "storefront_" + s.nextID("m92")
	item := ports.Storefront{
		StorefrontID:     id,
		CreatorUserID:    actorUserID,
		Subdomain:        slug + ".whop.com",
		DisplayName:      strings.TrimSpace(input.DisplayName),
		Status:           "draft",
		Category:         strings.TrimSpace(input.Category),
		VisibilityMode:   "public",
		DiscoverEligible: false,
		DiscoverReasons:  []string{"not_published"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	s.storefrontsByID[id] = item
	s.storefrontIDBySlug[slug] = id
	s.storefrontIDByCreator[actorUserID] = id
	s.productIDsByStorefront[id] = make(map[string]struct{})
	s.catalogSyncedByStorefront[id] = false

	return cloneStorefront(item), nil
}

func (s *Store) UpdateStorefront(
	ctx context.Context,
	actorUserID string,
	storefrontID string,
	input ports.UpdateStorefrontInput,
	now time.Time,
) (ports.Storefront, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.storefrontsByID[strings.TrimSpace(storefrontID)]
	if !ok {
		return ports.Storefront{}, domainerrors.ErrStorefrontNotFound
	}
	if item.CreatorUserID != strings.TrimSpace(actorUserID) {
		return ports.Storefront{}, domainerrors.ErrForbidden
	}

	if len(strings.TrimSpace(input.Headline)) > 120 || len(strings.TrimSpace(input.Bio)) > 1000 {
		return ports.Storefront{}, domainerrors.ErrInvalidRequest
	}

	if strings.TrimSpace(input.Headline) != "" {
		item.Headline = strings.TrimSpace(input.Headline)
	}
	if strings.TrimSpace(input.Bio) != "" {
		item.Bio = strings.TrimSpace(input.Bio)
	}
	if mode := ports.NormalizeVisibilityMode(strings.TrimSpace(input.VisibilityMode)); mode != "" {
		if mode == "private" && strings.TrimSpace(input.Password) == "" {
			return ports.Storefront{}, domainerrors.ErrInvalidRequest
		}
		item.VisibilityMode = mode
		if mode == "private" {
			item.PasswordHash = hashPassword(input.Password)
		} else {
			item.PasswordHash = ""
		}
	} else if strings.TrimSpace(input.VisibilityMode) != "" {
		return ports.Storefront{}, domainerrors.ErrInvalidRequest
	}

	item.UpdatedAt = now.UTC()
	s.storefrontsByID[item.StorefrontID] = item
	return cloneStorefront(item), nil
}

func (s *Store) GetStorefrontByID(ctx context.Context, storefrontID string, actorUserID string) (ports.Storefront, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.storefrontsByID[strings.TrimSpace(storefrontID)]
	if !ok {
		return ports.Storefront{}, domainerrors.ErrStorefrontNotFound
	}
	if item.CreatorUserID != strings.TrimSpace(actorUserID) {
		return ports.Storefront{}, domainerrors.ErrForbidden
	}
	return cloneStorefront(item), nil
}

func (s *Store) GetStorefrontBySlug(ctx context.Context, slug string) (ports.Storefront, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.storefrontIDBySlug[normalizeSlug(slug)]
	if !ok {
		return ports.Storefront{}, domainerrors.ErrStorefrontNotFound
	}
	item := s.storefrontsByID[id]
	if item.Status != "published" {
		return ports.Storefront{}, domainerrors.ErrStorefrontNotFound
	}
	if item.VisibilityMode == "private" {
		return ports.Storefront{}, domainerrors.ErrPrivateAccessDenied
	}
	return cloneStorefront(item), nil
}

func (s *Store) PublishStorefront(
	ctx context.Context,
	actorUserID string,
	storefrontID string,
	now time.Time,
) (ports.Storefront, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.storefrontsByID[strings.TrimSpace(storefrontID)]
	if !ok {
		return ports.Storefront{}, domainerrors.ErrStorefrontNotFound
	}
	if item.CreatorUserID != strings.TrimSpace(actorUserID) {
		return ports.Storefront{}, domainerrors.ErrForbidden
	}
	if item.Status == "published" {
		return ports.Storefront{}, domainerrors.ErrAlreadyPublished
	}

	// Enforce DBR assumptions:
	// M61 projection must be present for creator.
	subscriptionActive, hasSubscriptionProjection := s.subscriptionActiveByUser[item.CreatorUserID]
	if !hasSubscriptionProjection {
		return ports.Storefront{}, domainerrors.ErrDependencyUnavailable
	}
	// M60 product projection sync must have occurred for this storefront.
	if !s.catalogSyncedByStorefront[item.StorefrontID] {
		return ports.Storefront{}, domainerrors.ErrDependencyUnavailable
	}

	item.Status = "published"
	reasons := make([]string, 0)
	if !subscriptionActive {
		reasons = append(reasons, "inactive_subscription")
	}
	if len(s.productIDsByStorefront[item.StorefrontID]) == 0 {
		reasons = append(reasons, "no_published_products")
	}
	item.DiscoverEligible = len(reasons) == 0
	item.DiscoverReasons = reasons
	if len(item.DiscoverReasons) == 0 {
		item.DiscoverReasons = []string{"eligible"}
	}
	item.UpdatedAt = now.UTC()

	s.storefrontsByID[item.StorefrontID] = item
	return cloneStorefront(item), nil
}

func (s *Store) ReportStorefront(
	ctx context.Context,
	actorUserID string,
	storefrontID string,
	input ports.ReportInput,
	now time.Time,
) (ports.ReportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(storefrontID)
	if _, ok := s.storefrontsByID[id]; !ok {
		return ports.ReportResult{}, domainerrors.ErrStorefrontNotFound
	}
	if strings.TrimSpace(input.Reason) == "" || strings.TrimSpace(input.Type) == "" {
		return ports.ReportResult{}, domainerrors.ErrInvalidRequest
	}

	key := reportDedupKey(id, strings.TrimSpace(actorUserID))
	if at, exists := s.reportDedupAt[key]; exists && now.UTC().Sub(at.UTC()) < 24*time.Hour {
		return ports.ReportResult{}, domainerrors.ErrConflict
	}
	s.reportDedupAt[key] = now.UTC()
	return ports.ReportResult{
		StorefrontID: id,
		Status:       "queued",
		ReportedAt:   now.UTC(),
	}, nil
}

func (s *Store) ConsumeProductPublishedEvent(
	ctx context.Context,
	event ports.ProductPublishedEvent,
	now time.Time,
) (ports.ProductProjectionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" || strings.TrimSpace(event.StorefrontID) == "" || strings.TrimSpace(event.ProductID) == "" {
		return ports.ProductProjectionResult{}, domainerrors.ErrInvalidRequest
	}
	hash := hashProductEvent(event)
	if dedup, exists := s.productEventDedupByID[eventID]; exists {
		if !dedup.ExpiresAt.IsZero() && now.UTC().After(dedup.ExpiresAt.UTC()) {
			delete(s.productEventDedupByID, eventID)
		} else if dedup.RequestHash != hash {
			return ports.ProductProjectionResult{}, domainerrors.ErrIdempotencyConflict
		} else {
			return ports.ProductProjectionResult{
				StorefrontID: strings.TrimSpace(event.StorefrontID),
				ProductID:    strings.TrimSpace(event.ProductID),
				Accepted:     false,
			}, nil
		}
	}

	storefrontID := strings.TrimSpace(event.StorefrontID)
	productID := strings.TrimSpace(event.ProductID)
	if _, ok := s.storefrontsByID[storefrontID]; !ok {
		return ports.ProductProjectionResult{}, domainerrors.ErrStorefrontNotFound
	}
	if _, ok := s.validProductIDs[productID]; !ok {
		return ports.ProductProjectionResult{}, domainerrors.ErrDependencyUnavailable
	}

	s.catalogSyncedByStorefront[storefrontID] = true
	if _, ok := s.productIDsByStorefront[storefrontID]; !ok {
		s.productIDsByStorefront[storefrontID] = make(map[string]struct{})
	}
	s.productIDsByStorefront[storefrontID][productID] = struct{}{}
	s.productEventDedupByID[eventID] = eventDedupRecord{
		RequestHash: hash,
		ExpiresAt:   now.UTC().Add(7 * 24 * time.Hour),
	}
	return ports.ProductProjectionResult{
		StorefrontID: storefrontID,
		ProductID:    productID,
		Accepted:     true,
	}, nil
}

func (s *Store) UpsertSubscriptionProjection(ctx context.Context, input ports.SubscriptionProjectionInput, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return domainerrors.ErrInvalidRequest
	}
	_ = now
	s.subscriptionActiveByUser[userID] = input.Active
	return nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) NewID(ctx context.Context) (string, error) {
	return s.nextID("m92"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func cloneStorefront(item ports.Storefront) ports.Storefront {
	out := item
	out.DiscoverReasons = append([]string(nil), item.DiscoverReasons...)
	return out
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimSuffix(value, ".whop.com")
	value = slugSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}

func buildSlug(actorUserID string) string {
	slug := normalizeSlug(actorUserID)
	if slug == "" {
		return "creator"
	}
	return slug
}

func hashPassword(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func reportDedupKey(storefrontID string, actorUserID string) string {
	return storefrontID + "|" + actorUserID
}

func hashProductEvent(event ports.ProductPublishedEvent) string {
	raw, _ := json.Marshal(event)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
