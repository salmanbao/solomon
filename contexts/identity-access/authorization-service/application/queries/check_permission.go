package queries

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/domain/services"
	"solomon/contexts/identity-access/authorization-service/ports"
)

// CheckPermissionQuery is the request model for single-permission evaluation.
type CheckPermissionQuery struct {
	UserID       string
	Permission   string
	ResourceType string
	ResourceID   string
}

// CheckPermissionUseCase orchestrates cache-first permission evaluation.
type CheckPermissionUseCase struct {
	Repository         ports.Repository
	PermissionCache    ports.PermissionCache
	Clock              ports.Clock
	PermissionCacheTTL time.Duration
	Logger             *slog.Logger
}

// Execute evaluates a permission and returns deny-by-default on lookup failures.
func (u CheckPermissionUseCase) Execute(ctx context.Context, query CheckPermissionQuery) (entities.PermissionDecision, error) {
	if strings.TrimSpace(query.UserID) == "" {
		return entities.PermissionDecision{}, domainerrors.ErrInvalidUserID
	}
	if strings.TrimSpace(query.Permission) == "" {
		return entities.PermissionDecision{}, domainerrors.ErrInvalidPermission
	}

	logger := application.ResolveLogger(u.Logger)
	now := u.now()
	logger.Debug("check permission started",
		"event", "authz_check_started",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", query.UserID,
		"permission", query.Permission,
		"resource_type", query.ResourceType,
		"resource_id", query.ResourceID,
	)

	permissions, cacheHit, err := u.loadPermissions(ctx, query.UserID, now)
	if err != nil {
		logger.Error("permission lookup failed, deny by default",
			"event", "authz_permission_lookup_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", query.UserID,
			"permission", query.Permission,
			"error", err.Error(),
		)
		return entities.PermissionDecision{
			UserID:     query.UserID,
			Permission: query.Permission,
			Allowed:    false,
			Reason:     "deny_by_default",
			CheckedAt:  now,
			CacheHit:   false,
		}, nil
	}

	allowed := services.GrantsPermission(permissions, query.Permission)
	reason := "permission_granted"
	if !allowed {
		reason = "permission_missing"
		logger.Warn("check permission denied",
			"event", "authz_check_denied",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", query.UserID,
			"permission", query.Permission,
			"resource_type", query.ResourceType,
			"resource_id", query.ResourceID,
			"cache_hit", cacheHit,
		)
	} else {
		logger.Debug("check permission allowed",
			"event", "authz_check_allowed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", query.UserID,
			"permission", query.Permission,
			"resource_type", query.ResourceType,
			"resource_id", query.ResourceID,
			"cache_hit", cacheHit,
		)
	}

	return entities.PermissionDecision{
		UserID:     query.UserID,
		Permission: query.Permission,
		Allowed:    allowed,
		Reason:     reason,
		CheckedAt:  now,
		CacheHit:   cacheHit,
	}, nil
}

func (u CheckPermissionUseCase) loadPermissions(
	ctx context.Context,
	userID string,
	now time.Time,
) ([]string, bool, error) {
	if u.PermissionCache != nil {
		items, hit, err := u.PermissionCache.Get(ctx, userID, now)
		if err != nil {
			return nil, false, err
		}
		if hit {
			return items, true, nil
		}
	}

	permissions, err := u.Repository.ListEffectivePermissions(ctx, userID, now)
	if err != nil {
		return nil, false, err
	}

	if u.PermissionCache != nil {
		_ = u.PermissionCache.Set(ctx, userID, permissions, now.Add(u.cacheTTL()))
	}
	return permissions, false, nil
}

func (u CheckPermissionUseCase) cacheTTL() time.Duration {
	if u.PermissionCacheTTL <= 0 {
		return 5 * time.Minute
	}
	return u.PermissionCacheTTL
}

func (u CheckPermissionUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
