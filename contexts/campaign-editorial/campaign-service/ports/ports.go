package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
)

type CampaignFilter struct {
	BrandID string
	Status  entities.CampaignStatus
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign entities.Campaign) error
	UpdateCampaign(ctx context.Context, campaign entities.Campaign) error
	GetCampaign(ctx context.Context, campaignID string) (entities.Campaign, error)
	ListCampaigns(ctx context.Context, filter CampaignFilter) ([]entities.Campaign, error)
}

type MediaRepository interface {
	AddMedia(ctx context.Context, media entities.Media) error
	GetMedia(ctx context.Context, mediaID string) (entities.Media, error)
	UpdateMedia(ctx context.Context, media entities.Media) error
	ListMediaByCampaign(ctx context.Context, campaignID string) ([]entities.Media, error)
}

type HistoryRepository interface {
	AppendState(ctx context.Context, item entities.StateHistory) error
	AppendBudget(ctx context.Context, item entities.BudgetLog) error
}

type IdempotencyRecord struct {
	Key             string
	RequestHash     string
	ResponsePayload []byte
	ExpiresAt       time.Time
}

type IdempotencyStore interface {
	GetRecord(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	PutRecord(ctx context.Context, record IdempotencyRecord) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
