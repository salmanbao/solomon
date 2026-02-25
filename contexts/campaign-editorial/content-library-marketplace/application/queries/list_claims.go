package queries

import (
	"context"
	"log/slog"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type ListClaimsQuery struct {
	UserID string
}

type ListClaimsResult struct {
	Items []entities.Claim
}

type ListClaimsUseCase struct {
	Claims ports.ClaimRepository
	Logger *slog.Logger
}

func (u ListClaimsUseCase) Execute(ctx context.Context, query ListClaimsQuery) (ListClaimsResult, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("list claims started",
		"event", "list_claims_started",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"user_id", query.UserID,
	)

	items, err := u.Claims.ListClaimsByUser(ctx, query.UserID)
	if err != nil {
		logger.Error("list claims failed",
			"event", "list_claims_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"user_id", query.UserID,
			"error", err.Error(),
		)
		return ListClaimsResult{}, err
	}

	logger.Info("list claims completed",
		"event", "list_claims_completed",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"user_id", query.UserID,
		"items_count", len(items),
	)

	return ListClaimsResult{
		Items: items,
	}, nil
}
