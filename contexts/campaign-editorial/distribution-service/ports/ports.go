package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
)

type Repository interface {
	CreateItem(ctx context.Context, item entities.DistributionItem) error
	UpdateItem(ctx context.Context, item entities.DistributionItem) error
	GetItem(ctx context.Context, itemID string) (entities.DistributionItem, error)
	ListItemsByInfluencer(ctx context.Context, influencerID string) ([]entities.DistributionItem, error)
	AddOverlay(ctx context.Context, overlay entities.Overlay) error
	UpsertPlatformStatus(ctx context.Context, status entities.PlatformStatus) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
