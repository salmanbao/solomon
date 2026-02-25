package queries

import (
	"context"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
)

type CheckPermissionsBatchQuery struct {
	UserID      string
	Permissions []string
}

type CheckPermissionsBatchUseCase struct {
	CheckPermission CheckPermissionUseCase
}

func (u CheckPermissionsBatchUseCase) Execute(
	ctx context.Context,
	query CheckPermissionsBatchQuery,
) ([]entities.PermissionDecision, error) {
	results := make([]entities.PermissionDecision, 0, len(query.Permissions))
	for _, permission := range query.Permissions {
		decision, err := u.CheckPermission.Execute(ctx, CheckPermissionQuery{
			UserID:     query.UserID,
			Permission: permission,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, decision)
	}
	return results, nil
}
