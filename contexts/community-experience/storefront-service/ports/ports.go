package ports

import (
	"context"
	"time"
)

var allowedCategories = map[string]struct{}{
	"Business":  {},
	"Tech":      {},
	"Fitness":   {},
	"Finance":   {},
	"Gaming":    {},
	"Lifestyle": {},
	"Education": {},
	"Other":     {},
}

func IsValidCategory(value string) bool {
	_, ok := allowedCategories[value]
	return ok
}

func NormalizeVisibilityMode(value string) string {
	switch value {
	case "public", "unlisted", "private":
		return value
	default:
		return ""
	}
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type CreateStorefrontInput struct {
	DisplayName string
	Category    string
}

type UpdateStorefrontInput struct {
	Headline       string
	Bio            string
	VisibilityMode string
	Password       string
}

type Storefront struct {
	StorefrontID     string
	CreatorUserID    string
	Subdomain        string
	DisplayName      string
	Headline         string
	Bio              string
	Status           string
	Category         string
	VisibilityMode   string
	PasswordHash     string
	DiscoverEligible bool
	DiscoverReasons  []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReportInput struct {
	Reason string
	Type   string
}

type ReportResult struct {
	StorefrontID string
	Status       string
	ReportedAt   time.Time
}

type ProductPublishedEvent struct {
	EventID      string
	StorefrontID string
	ProductID    string
	OccurredAt   time.Time
}

type ProductProjectionResult struct {
	StorefrontID string
	ProductID    string
	Accepted     bool
}

type SubscriptionProjectionInput struct {
	UserID string
	Active bool
}

type Repository interface {
	CreateStorefront(ctx context.Context, actorUserID string, input CreateStorefrontInput, now time.Time) (Storefront, error)
	UpdateStorefront(ctx context.Context, actorUserID string, storefrontID string, input UpdateStorefrontInput, now time.Time) (Storefront, error)
	GetStorefrontByID(ctx context.Context, storefrontID string, actorUserID string) (Storefront, error)
	GetStorefrontBySlug(ctx context.Context, slug string) (Storefront, error)
	PublishStorefront(ctx context.Context, actorUserID string, storefrontID string, now time.Time) (Storefront, error)
	ReportStorefront(ctx context.Context, actorUserID string, storefrontID string, input ReportInput, now time.Time) (ReportResult, error)
	ConsumeProductPublishedEvent(ctx context.Context, event ProductPublishedEvent, now time.Time) (ProductProjectionResult, error)
	UpsertSubscriptionProjection(ctx context.Context, input SubscriptionProjectionInput, now time.Time) error
}
