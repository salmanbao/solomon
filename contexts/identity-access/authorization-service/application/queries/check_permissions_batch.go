package queries

import (
	"context"
	"log/slog"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/domain/entities"
)

// CheckPermissionsBatchQuery groups multiple permission checks for one user.
type CheckPermissionsBatchQuery struct {
	UserID      string
	Permissions []string
}

// CheckPermissionsBatchUseCase reuses CheckPermissionUseCase per requested permission.
type CheckPermissionsBatchUseCase struct {
	CheckPermission CheckPermissionUseCase
	Logger          *slog.Logger
}

// Execute returns one decision per input permission preserving input order.
func (u CheckPermissionsBatchUseCase) Execute(
	ctx context.Context,
	query CheckPermissionsBatchQuery,
) ([]entities.PermissionDecision, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("check permission batch started",
		"event", "authz_check_batch_started",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", query.UserID,
		"permission_count", len(query.Permissions),
	)

	results := make([]entities.PermissionDecision, 0, len(query.Permissions))
	for _, permission := range query.Permissions {
		decision, err := u.CheckPermission.Execute(ctx, CheckPermissionQuery{
			UserID:     query.UserID,
			Permission: permission,
		})
		if err != nil {
			logger.Error("check permission batch failed",
				"event", "authz_check_batch_failed",
				"module", "identity-access/authorization-service",
				"layer", "application",
				"user_id", query.UserID,
				"permission", permission,
				"error", err.Error(),
			)
			return nil, err
		}
		results = append(results, decision)
	}

	logger.Info("check permission batch completed",
		"event", "authz_check_batch_completed",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", query.UserID,
		"permission_count", len(query.Permissions),
	)
	return results, nil
}
