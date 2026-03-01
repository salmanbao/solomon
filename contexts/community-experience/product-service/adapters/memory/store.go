package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/community-experience/product-service/domain/errors"
	"solomon/contexts/community-experience/product-service/ports"
)

type Store struct {
	mu          sync.RWMutex
	products    map[string]ports.Product
	access      map[string]map[string]ports.AccessRecord
	purchases   map[string]ports.Purchase
	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	products := map[string]ports.Product{
		"prod_001": {
			ProductID:      "prod_001",
			CreatorID:      "creator_001",
			Name:           "Ultimate Marketing Course",
			Description:    "Learn marketing from scratch with 20+ video lessons",
			ProductType:    "digital",
			PricingModel:   "one_time",
			PriceCents:     9900,
			Currency:       "USD",
			CoverImageURL:  "https://cdn.viralforge.com/products/prod_001/cover.jpg",
			Visibility:     "public",
			Status:         "published",
			SalesCount:     47,
			Rating:         4.8,
			InventoryCount: 0,
			UnlimitedStock: true,
			MediaOrder:     []string{"media_1", "media_2"},
			CreatedAt:      now.Add(-30 * 24 * time.Hour),
			UpdatedAt:      now.Add(-24 * time.Hour),
		},
		"prod_002": {
			ProductID:      "prod_002",
			CreatorID:      "creator_001",
			Name:           "Growth Workshop Ticket",
			Description:    "Live event ticket",
			ProductType:    "event",
			PricingModel:   "one_time",
			PriceCents:     2500,
			Currency:       "USD",
			Visibility:     "public",
			Status:         "published",
			SalesCount:     6,
			Rating:         4.6,
			InventoryCount: 10,
			UnlimitedStock: false,
			MediaOrder:     []string{"media_3"},
			CreatedAt:      now.Add(-14 * 24 * time.Hour),
			UpdatedAt:      now.Add(-6 * time.Hour),
		},
	}

	expires := now.Add(365 * 24 * time.Hour)
	access := map[string]map[string]ports.AccessRecord{
		"user_123": {
			"prod_001": {
				UserID:     "user_123",
				ProductID:  "prod_001",
				AccessType: "lifetime",
				Status:     "active",
				GrantedAt:  now.Add(-10 * 24 * time.Hour),
				ExpiresAt:  nil,
			},
			"prod_002": {
				UserID:     "user_123",
				ProductID:  "prod_002",
				AccessType: "subscription",
				Status:     "active",
				GrantedAt:  now.Add(-5 * 24 * time.Hour),
				ExpiresAt:  &expires,
			},
		},
	}

	return &Store{
		products:    products,
		access:      access,
		purchases:   make(map[string]ports.Purchase),
		idempotency: make(map[string]ports.IdempotencyRecord),
	}
}

func (s *Store) ListProducts(ctx context.Context, filter ports.ProductFilter) ([]ports.Product, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ports.Product, 0, len(s.products))
	for _, item := range s.products {
		if filter.CreatorID != "" && item.CreatorID != filter.CreatorID {
			continue
		}
		if filter.ProductType != "" && item.ProductType != filter.ProductType {
			continue
		}
		if filter.Visibility != "" && item.Visibility != filter.Visibility {
			continue
		}
		items = append(items, cloneProduct(item))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := len(items)
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	start := (page - 1) * limit
	if start >= total {
		return []ports.Product{}, total, nil
	}
	end := start + limit
	if end > total {
		end = total
	}
	return append([]ports.Product(nil), items[start:end]...), total, nil
}

func (s *Store) CreateProduct(ctx context.Context, input ports.CreateProductInput, now time.Time) (ports.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	productID := "prod_" + s.nextID("new")
	currency := strings.TrimSpace(strings.ToUpper(input.Currency))
	if currency == "" {
		currency = "USD"
	}
	visibility := strings.TrimSpace(strings.ToLower(input.Visibility))
	if visibility == "" {
		visibility = "public"
	}
	product := ports.Product{
		ProductID:      productID,
		CreatorID:      input.CreatorID,
		Name:           input.Name,
		Description:    input.Description,
		ProductType:    strings.ToLower(input.ProductType),
		PricingModel:   strings.ToLower(input.PricingModel),
		PriceCents:     input.PriceCents,
		Currency:       currency,
		Visibility:     visibility,
		Status:         "draft",
		SalesCount:     0,
		Rating:         0,
		InventoryCount: 0,
		UnlimitedStock: true,
		MediaOrder:     []string{},
		CreatedAt:      now.UTC(),
		UpdatedAt:      now.UTC(),
	}
	if product.ProductType == "physical" || product.ProductType == "event" {
		product.UnlimitedStock = false
	}
	s.products[product.ProductID] = product
	return cloneProduct(product), nil
}

func (s *Store) GetProduct(ctx context.Context, productID string) (ports.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[productID]
	if !ok {
		return ports.Product{}, domainerrors.ErrProductNotFound
	}
	return cloneProduct(product), nil
}

func (s *Store) CheckAccess(ctx context.Context, userID string, productID string, now time.Time) (ports.AccessRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.products[productID]; !ok {
		return ports.AccessRecord{}, false, domainerrors.ErrProductNotFound
	}
	userAccess, ok := s.access[userID]
	if !ok {
		return ports.AccessRecord{}, false, nil
	}
	record, ok := userAccess[productID]
	if !ok {
		return ports.AccessRecord{}, false, nil
	}
	if record.Status != "active" {
		return record, false, nil
	}
	if record.ExpiresAt != nil && !record.ExpiresAt.UTC().After(now.UTC()) {
		return record, false, nil
	}
	return record, true, nil
}

func (s *Store) CreatePurchase(ctx context.Context, userID string, productID string, now time.Time) (ports.Purchase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.products[productID]
	if !ok {
		return ports.Purchase{}, domainerrors.ErrProductNotFound
	}
	if !product.UnlimitedStock {
		if product.InventoryCount <= 0 {
			return ports.Purchase{}, domainerrors.ErrSoldOut
		}
		product.InventoryCount--
	}
	product.SalesCount++
	product.UpdatedAt = now.UTC()
	s.products[productID] = product

	purchase := ports.Purchase{
		PurchaseID:        "pur_" + s.nextID("purchase"),
		UserID:            userID,
		ProductID:         productID,
		AmountCents:       product.PriceCents,
		Currency:          product.Currency,
		Status:            "completed",
		FulfillmentStatus: "pending",
		CreatedAt:         now.UTC(),
	}
	s.purchases[purchase.PurchaseID] = purchase

	userAccess := s.access[userID]
	if userAccess == nil {
		userAccess = make(map[string]ports.AccessRecord)
		s.access[userID] = userAccess
	}
	userAccess[productID] = ports.AccessRecord{
		UserID:     userID,
		ProductID:  productID,
		AccessType: "lifetime",
		Status:     "active",
		GrantedAt:  now.UTC(),
		ExpiresAt:  nil,
	}
	return purchase, nil
}

func (s *Store) FulfillPurchase(ctx context.Context, userID string, productID string, now time.Time) (ports.FulfillmentResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var matched *ports.Purchase
	for i := range s.purchases {
		p := s.purchases[i]
		if p.UserID == userID && p.ProductID == productID {
			copy := p
			matched = &copy
		}
	}
	if matched == nil {
		return ports.FulfillmentResult{}, domainerrors.ErrPurchaseNotFound
	}
	fulfilledAt := now.UTC()
	matched.FulfillmentStatus = "completed"
	matched.FulfilledAt = &fulfilledAt
	s.purchases[matched.PurchaseID] = *matched
	return ports.FulfillmentResult{
		PurchaseID:      matched.PurchaseID,
		ProductID:       matched.ProductID,
		Status:          "completed",
		FulfillmentType: "digital",
		ProcessedAt:     now.UTC(),
	}, nil
}

func (s *Store) AdjustInventory(ctx context.Context, adminID string, productID string, newCount int, reason string, now time.Time) (ports.InventoryAdjustment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.products[productID]
	if !ok {
		return ports.InventoryAdjustment{}, domainerrors.ErrProductNotFound
	}
	oldCount := product.InventoryCount
	product.InventoryCount = newCount
	product.UnlimitedStock = false
	product.UpdatedAt = now.UTC()
	s.products[productID] = product
	return ports.InventoryAdjustment{
		ProductID: productID,
		OldCount:  oldCount,
		NewCount:  newCount,
		ChangedBy: adminID,
		Reason:    reason,
		ChangedAt: now.UTC(),
	}, nil
}

func (s *Store) ReorderMedia(ctx context.Context, productID string, mediaOrder []string, now time.Time) (ports.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product, ok := s.products[productID]
	if !ok {
		return ports.Product{}, domainerrors.ErrProductNotFound
	}
	product.MediaOrder = append([]string(nil), mediaOrder...)
	product.UpdatedAt = now.UTC()
	s.products[productID] = product
	return cloneProduct(product), nil
}

func (s *Store) DiscoverProducts(ctx context.Context, limit int) ([]ports.Product, error) {
	return s.SearchProducts(ctx, "", "", limit)
}

func (s *Store) SearchProducts(ctx context.Context, query string, productType string, limit int) ([]ports.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.TrimSpace(strings.ToLower(query))
	productType = strings.TrimSpace(strings.ToLower(productType))
	if limit <= 0 {
		limit = 20
	}

	items := make([]ports.Product, 0)
	for _, item := range s.products {
		if item.Visibility != "public" {
			continue
		}
		if productType != "" && strings.ToLower(item.ProductType) != productType {
			continue
		}
		if query != "" {
			candidate := strings.ToLower(item.Name + " " + item.Description)
			if !strings.Contains(candidate, query) {
				continue
			}
		}
		items = append(items, cloneProduct(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].SalesCount > items[j].SalesCount
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) ExportUserData(ctx context.Context, userID string, now time.Time) (ports.UserDataExport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	export := ports.UserDataExport{
		UserID:        userID,
		Purchases:     make([]ports.Purchase, 0),
		AccessRecords: make([]ports.AccessRecord, 0),
		GeneratedAt:   now.UTC(),
	}
	for _, p := range s.purchases {
		if p.UserID == userID {
			export.Purchases = append(export.Purchases, p)
		}
	}
	if userAccess, ok := s.access[userID]; ok {
		for _, a := range userAccess {
			export.AccessRecords = append(export.AccessRecords, a)
		}
	}
	return export, nil
}

func (s *Store) DeleteUserData(ctx context.Context, userID string, now time.Time) (ports.UserDeleteResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	revoked := 0
	if userAccess, ok := s.access[userID]; ok {
		for productID, rec := range userAccess {
			rec.Status = "revoked"
			rec.ExpiresAt = nil
			userAccess[productID] = rec
			revoked++
		}
	}

	anonymized := 0
	for purchaseID, purchase := range s.purchases {
		if purchase.UserID == userID {
			purchase.UserID = "anonymized"
			s.purchases[purchaseID] = purchase
			anonymized++
		}
	}
	return ports.UserDeleteResult{
		UserID:             userID,
		RevokedAccessCount: revoked,
		AnonymizedCount:    anonymized,
		ProcessedAt:        now.UTC(),
	}, nil
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
	return s.nextID("m60"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func cloneProduct(in ports.Product) ports.Product {
	out := in
	out.MediaOrder = append([]string(nil), in.MediaOrder...)
	return out
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
