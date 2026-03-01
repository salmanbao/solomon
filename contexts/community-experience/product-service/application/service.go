package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/product-service/domain/errors"
	"solomon/contexts/community-experience/product-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) ListProducts(ctx context.Context, filter ports.ProductFilter) ([]ports.Product, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.Repo.ListProducts(ctx, filter)
}

func (s Service) CreateProduct(
	ctx context.Context,
	idempotencyKey string,
	input ports.CreateProductInput,
) (ports.Product, error) {
	var out ports.Product
	if strings.TrimSpace(input.CreatorID) == "" ||
		strings.TrimSpace(input.Name) == "" ||
		strings.TrimSpace(input.ProductType) == "" ||
		strings.TrimSpace(input.PricingModel) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if input.PriceCents < 0 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}

	payload, _ := json.Marshal(input)
	requestHash := hashStrings("create_product", string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CreateProduct(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) CheckAccess(ctx context.Context, userID string, productID string) (ports.AccessRecord, bool, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(productID) == "" {
		return ports.AccessRecord{}, false, domainerrors.ErrInvalidRequest
	}
	return s.Repo.CheckAccess(ctx, userID, productID, s.now())
}

func (s Service) PurchaseProduct(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	productID string,
) (ports.Purchase, error) {
	var out ports.Purchase
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(productID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("purchase_product", userID, productID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CreatePurchase(ctx, userID, productID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) FulfillProduct(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	productID string,
) (ports.FulfillmentResult, error) {
	var out ports.FulfillmentResult
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(productID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("fulfill_product", userID, productID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.FulfillPurchase(ctx, userID, productID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) AdjustInventory(
	ctx context.Context,
	idempotencyKey string,
	adminID string,
	productID string,
	newCount int,
	reason string,
) (ports.InventoryAdjustment, error) {
	var out ports.InventoryAdjustment
	if strings.TrimSpace(adminID) == "" ||
		strings.TrimSpace(productID) == "" ||
		strings.TrimSpace(reason) == "" ||
		newCount < 0 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings(
		"adjust_inventory",
		adminID,
		productID,
		fmt.Sprintf("%d", newCount),
		reason,
	)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.AdjustInventory(ctx, adminID, productID, newCount, reason, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) ReorderMedia(
	ctx context.Context,
	idempotencyKey string,
	productID string,
	mediaOrder []string,
) (ports.Product, error) {
	var out ports.Product
	if strings.TrimSpace(productID) == "" || len(mediaOrder) == 0 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(mediaOrder)
	requestHash := hashStrings("reorder_media", productID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.ReorderMedia(ctx, productID, mediaOrder, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) DiscoverProducts(ctx context.Context, limit int) ([]ports.Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.Repo.DiscoverProducts(ctx, limit)
}

func (s Service) SearchProducts(ctx context.Context, query string, productType string, limit int) ([]ports.Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.Repo.SearchProducts(ctx, query, productType, limit)
}

func (s Service) ExportUserData(ctx context.Context, userID string) (ports.UserDataExport, error) {
	if strings.TrimSpace(userID) == "" {
		return ports.UserDataExport{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.ExportUserData(ctx, userID, s.now())
}

func (s Service) DeleteUserData(
	ctx context.Context,
	idempotencyKey string,
	userID string,
) (ports.UserDeleteResult, error) {
	var out ports.UserDeleteResult
	if strings.TrimSpace(userID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("delete_user_data", userID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.DeleteUserData(ctx, userID, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) requireIdempotency(key string) error {
	if strings.TrimSpace(key) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	return nil
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	logger := ResolveLogger(s.Logger)
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

	logger.Debug("product service idempotent operation committed",
		"event", "product_service_idempotent_operation_committed",
		"module", "community-experience/product-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
