package queries

import (
	"context"
	"strings"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"
)

type ListUserRolesUseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
}

func (u ListUserRolesUseCase) Execute(ctx context.Context, userID string) ([]entities.RoleAssignment, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, domainerrors.ErrInvalidUserID
	}
	return u.Repository.ListUserRoles(ctx, userID, u.now())
}

func (u ListUserRolesUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
