package queries

import (
	"context"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

type UseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
}

func (uc UseCase) GetItem(ctx context.Context, itemID string) (entities.DistributionItem, error) {
	return uc.Repository.GetItem(ctx, strings.TrimSpace(itemID))
}

func (uc UseCase) Preview(ctx context.Context, itemID string) (string, time.Time, error) {
	item, err := uc.Repository.GetItem(ctx, strings.TrimSpace(itemID))
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := uc.Clock.Now().UTC().Add(5 * time.Minute)
	url := "https://preview.viralforge.local/distribution/" + item.ID
	return url, expiresAt, nil
}
