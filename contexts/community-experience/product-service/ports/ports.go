package ports

import (
	"context"
	"time"
)

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

type Product struct {
	ProductID      string
	CreatorID      string
	Name           string
	Description    string
	ProductType    string
	PricingModel   string
	PriceCents     int64
	Currency       string
	CoverImageURL  string
	Visibility     string
	Status         string
	SalesCount     int64
	Rating         float64
	InventoryCount int
	UnlimitedStock bool
	MediaOrder     []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProductFilter struct {
	CreatorID   string
	ProductType string
	Visibility  string
	Page        int
	Limit       int
}

type CreateProductInput struct {
	CreatorID           string
	Name                string
	Description         string
	ProductType         string
	PricingModel        string
	PriceCents          int64
	Currency            string
	Category            string
	Visibility          string
	MetaTitle           string
	MetaDescription     string
	FulfillmentMetadata map[string]any
}

type AccessRecord struct {
	UserID     string
	ProductID  string
	AccessType string
	Status     string
	GrantedAt  time.Time
	ExpiresAt  *time.Time
}

type Purchase struct {
	PurchaseID        string
	UserID            string
	ProductID         string
	AmountCents       int64
	Currency          string
	Status            string
	FulfillmentStatus string
	CreatedAt         time.Time
	FulfilledAt       *time.Time
}

type FulfillmentResult struct {
	PurchaseID      string
	ProductID       string
	Status          string
	FulfillmentType string
	ProcessedAt     time.Time
}

type InventoryAdjustment struct {
	ProductID string
	OldCount  int
	NewCount  int
	ChangedBy string
	Reason    string
	ChangedAt time.Time
}

type UserDataExport struct {
	UserID        string
	Purchases     []Purchase
	AccessRecords []AccessRecord
	GeneratedAt   time.Time
}

type UserDeleteResult struct {
	UserID             string
	RevokedAccessCount int
	AnonymizedCount    int
	ProcessedAt        time.Time
}

type Repository interface {
	ListProducts(ctx context.Context, filter ProductFilter) ([]Product, int, error)
	CreateProduct(ctx context.Context, input CreateProductInput, now time.Time) (Product, error)
	GetProduct(ctx context.Context, productID string) (Product, error)
	CheckAccess(ctx context.Context, userID string, productID string, now time.Time) (AccessRecord, bool, error)
	CreatePurchase(ctx context.Context, userID string, productID string, now time.Time) (Purchase, error)
	FulfillPurchase(ctx context.Context, userID string, productID string, now time.Time) (FulfillmentResult, error)
	AdjustInventory(ctx context.Context, adminID string, productID string, newCount int, reason string, now time.Time) (InventoryAdjustment, error)
	ReorderMedia(ctx context.Context, productID string, mediaOrder []string, now time.Time) (Product, error)
	DiscoverProducts(ctx context.Context, limit int) ([]Product, error)
	SearchProducts(ctx context.Context, query string, productType string, limit int) ([]Product, error)
	ExportUserData(ctx context.Context, userID string, now time.Time) (UserDataExport, error)
	DeleteUserData(ctx context.Context, userID string, now time.Time) (UserDeleteResult, error)
}
