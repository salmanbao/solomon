package commands

import (
	"context"
	"time"

	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/domain/services"
	"solomon/contexts/identity-access/authorization-service/ports"
)

func ensureActorPermission(
	ctx context.Context,
	repository ports.Repository,
	actorID string,
	permission string,
	now time.Time,
) error {
	permissions, err := repository.ListEffectivePermissions(ctx, actorID, now)
	if err != nil {
		return err
	}
	if !services.GrantsPermission(permissions, permission) {
		return domainerrors.ErrForbidden
	}
	return nil
}
