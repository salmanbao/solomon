package httpadapter

import (
	"context"
	"log/slog"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/application/commands"
	"solomon/contexts/identity-access/authorization-service/application/queries"
	"solomon/contexts/identity-access/authorization-service/transport/http"
)

// Handler maps HTTP DTOs to application commands/queries.
type Handler struct {
	CheckPermission queries.CheckPermissionUseCase
	CheckBatch      queries.CheckPermissionsBatchUseCase
	ListRoles       queries.ListUserRolesUseCase
	GrantRole       commands.GrantRoleUseCase
	RevokeRole      commands.RevokeRoleUseCase
	DelegateRole    commands.CreateDelegationUseCase
	Logger          *slog.Logger
}

// CheckPermissionHandler evaluates one permission for one user.
func (h Handler) CheckPermissionHandler(
	ctx context.Context,
	userID string,
	request httptransport.CheckPermissionRequest,
) (httptransport.CheckPermissionResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Debug("http authz check received",
		"event", "authz_http_check_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"user_id", userID,
		"permission", request.Permission,
	)

	decision, err := h.CheckPermission.Execute(ctx, queries.CheckPermissionQuery{
		UserID:       userID,
		Permission:   request.Permission,
		ResourceType: request.ResourceType,
		ResourceID:   request.ResourceID,
	})
	if err != nil {
		logger.Error("http authz check failed",
			"event", "authz_http_check_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"user_id", userID,
			"permission", request.Permission,
			"error", err.Error(),
		)
		return httptransport.CheckPermissionResponse{}, err
	}
	return httptransport.CheckPermissionResponse{
		UserID:     decision.UserID,
		Permission: decision.Permission,
		Allowed:    decision.Allowed,
		Reason:     decision.Reason,
		CheckedAt:  decision.CheckedAt,
		CacheHit:   decision.CacheHit,
	}, nil
}

// CheckBatchHandler evaluates multiple permissions in a single request.
func (h Handler) CheckBatchHandler(
	ctx context.Context,
	userID string,
	request httptransport.CheckBatchRequest,
) (httptransport.CheckBatchResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Debug("http authz check batch received",
		"event", "authz_http_check_batch_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"user_id", userID,
		"permission_count", len(request.Permissions),
	)

	decisions, err := h.CheckBatch.Execute(ctx, queries.CheckPermissionsBatchQuery{
		UserID:      userID,
		Permissions: request.Permissions,
	})
	if err != nil {
		logger.Error("http authz check batch failed",
			"event", "authz_http_check_batch_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"user_id", userID,
			"permission_count", len(request.Permissions),
			"error", err.Error(),
		)
		return httptransport.CheckBatchResponse{}, err
	}

	items := make([]httptransport.CheckPermissionResponse, 0, len(decisions))
	for _, decision := range decisions {
		items = append(items, httptransport.CheckPermissionResponse{
			UserID:     decision.UserID,
			Permission: decision.Permission,
			Allowed:    decision.Allowed,
			Reason:     decision.Reason,
			CheckedAt:  decision.CheckedAt,
			CacheHit:   decision.CacheHit,
		})
	}
	return httptransport.CheckBatchResponse{Results: items}, nil
}

// ListUserRolesHandler returns active and historical role assignments for a user.
func (h Handler) ListUserRolesHandler(ctx context.Context, userID string) (httptransport.ListUserRolesResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("http authz list roles received",
		"event", "authz_http_list_roles_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"user_id", userID,
	)

	roles, err := h.ListRoles.Execute(ctx, userID)
	if err != nil {
		logger.Error("http authz list roles failed",
			"event", "authz_http_list_roles_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"user_id", userID,
			"error", err.Error(),
		)
		return httptransport.ListUserRolesResponse{}, err
	}

	items := make([]httptransport.RoleAssignmentDTO, 0, len(roles))
	for _, role := range roles {
		items = append(items, httptransport.RoleAssignmentDTO{
			AssignmentID: role.AssignmentID,
			UserID:       role.UserID,
			RoleID:       role.RoleID,
			RoleName:     role.RoleName,
			AssignedBy:   role.AssignedBy,
			Reason:       role.Reason,
			AssignedAt:   role.AssignedAt,
			ExpiresAt:    role.ExpiresAt,
			IsActive:     role.IsActive,
			RevokedAt:    role.RevokedAt,
		})
	}
	return httptransport.ListUserRolesResponse{
		UserID: userID,
		Roles:  items,
	}, nil
}

// GrantRoleHandler executes idempotent role assignment and returns command result DTO.
func (h Handler) GrantRoleHandler(
	ctx context.Context,
	userID string,
	adminID string,
	idempotencyKey string,
	request httptransport.GrantRoleRequest,
) (httptransport.GrantRoleResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("http authz grant role received",
		"event", "authz_http_grant_role_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"user_id", userID,
		"admin_id", adminID,
		"role_id", request.RoleID,
	)

	result, err := h.GrantRole.Execute(ctx, commands.GrantRoleCommand{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		RoleID:         request.RoleID,
		AdminID:        adminID,
		Reason:         request.Reason,
		ExpiresAt:      request.ExpiresAt,
	})
	if err != nil {
		logger.Error("http authz grant role failed",
			"event", "authz_http_grant_role_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"user_id", userID,
			"admin_id", adminID,
			"role_id", request.RoleID,
			"error", err.Error(),
		)
		return httptransport.GrantRoleResponse{}, err
	}
	return httptransport.GrantRoleResponse{
		AssignmentID: result.Assignment.AssignmentID,
		UserID:       result.Assignment.UserID,
		RoleID:       result.Assignment.RoleID,
		AssignedAt:   result.Assignment.AssignedAt,
		ExpiresAt:    result.Assignment.ExpiresAt,
		AuditLogID:   result.AuditLogID,
		Replayed:     result.Replayed,
	}, nil
}

// RevokeRoleHandler executes idempotent role revocation and returns command result DTO.
func (h Handler) RevokeRoleHandler(
	ctx context.Context,
	userID string,
	adminID string,
	idempotencyKey string,
	request httptransport.RevokeRoleRequest,
) (httptransport.RevokeRoleResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("http authz revoke role received",
		"event", "authz_http_revoke_role_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"user_id", userID,
		"admin_id", adminID,
		"role_id", request.RoleID,
	)

	result, err := h.RevokeRole.Execute(ctx, commands.RevokeRoleCommand{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		RoleID:         request.RoleID,
		AdminID:        adminID,
		Reason:         request.Reason,
	})
	if err != nil {
		logger.Error("http authz revoke role failed",
			"event", "authz_http_revoke_role_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"user_id", userID,
			"admin_id", adminID,
			"role_id", request.RoleID,
			"error", err.Error(),
		)
		return httptransport.RevokeRoleResponse{}, err
	}
	return httptransport.RevokeRoleResponse{
		UserID:     result.Assignment.UserID,
		RoleID:     result.Assignment.RoleID,
		RevokedAt:  result.Assignment.RevokedAt,
		AuditLogID: result.AuditLogID,
		Replayed:   result.Replayed,
	}, nil
}

// CreateDelegationHandler creates a temporary delegation between admins.
func (h Handler) CreateDelegationHandler(
	ctx context.Context,
	idempotencyKey string,
	request httptransport.CreateDelegationRequest,
) (httptransport.CreateDelegationResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("http authz create delegation received",
		"event", "authz_http_create_delegation_received",
		"module", "identity-access/authorization-service",
		"layer", "transport",
		"from_admin_id", request.FromAdminID,
		"to_admin_id", request.ToAdminID,
		"role_id", request.RoleID,
	)

	result, err := h.DelegateRole.Execute(ctx, commands.CreateDelegationCommand{
		IdempotencyKey: idempotencyKey,
		FromAdminID:    request.FromAdminID,
		ToAdminID:      request.ToAdminID,
		RoleID:         request.RoleID,
		ExpiresAt:      request.ExpiresAt,
		Reason:         request.Reason,
	})
	if err != nil {
		logger.Error("http authz create delegation failed",
			"event", "authz_http_create_delegation_failed",
			"module", "identity-access/authorization-service",
			"layer", "transport",
			"from_admin_id", request.FromAdminID,
			"to_admin_id", request.ToAdminID,
			"role_id", request.RoleID,
			"error", err.Error(),
		)
		return httptransport.CreateDelegationResponse{}, err
	}
	return httptransport.CreateDelegationResponse{
		DelegationID: result.Delegation.DelegationID,
		FromAdminID:  result.Delegation.FromAdminID,
		ToAdminID:    result.Delegation.ToAdminID,
		RoleID:       result.Delegation.RoleID,
		DelegatedAt:  result.Delegation.DelegatedAt,
		ExpiresAt:    result.Delegation.ExpiresAt,
		AuditLogID:   result.AuditLogID,
		Replayed:     result.Replayed,
	}, nil
}
