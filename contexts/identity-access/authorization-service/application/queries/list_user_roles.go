package queries

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"
)

// ListUserRolesUseCase loads role assignments visible to authorization checks.
type ListUserRolesUseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	Logger     *slog.Logger
}

// Execute validates user identity then queries role assignments.
func (u ListUserRolesUseCase) Execute(ctx context.Context, userID string) ([]entities.RoleAssignment, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("list user roles started",
		"event", "authz_list_roles_started",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", userID,
	)

	if strings.TrimSpace(userID) == "" {
		return nil, domainerrors.ErrInvalidUserID
	}
	roles, err := u.Repository.ListUserRoles(ctx, userID, u.now())
	if err != nil {
		logger.Error("list user roles failed",
			"event", "authz_list_roles_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", userID,
			"error", err.Error(),
		)
		return nil, err
	}
	logger.Info("list user roles completed",
		"event", "authz_list_roles_completed",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", userID,
		"roles_count", len(roles),
	)
	return roles, nil
}

func (u ListUserRolesUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
