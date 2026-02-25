package httpadapter

import (
	"context"
	"log/slog"

	"solomon/contexts/identity-access/authorization-service/application/commands"
	"solomon/contexts/identity-access/authorization-service/application/queries"
	"solomon/contexts/identity-access/authorization-service/transport/http"
)

type Handler struct {
	CheckPermission queries.CheckPermissionUseCase
	CheckBatch      queries.CheckPermissionsBatchUseCase
	ListRoles       queries.ListUserRolesUseCase
	GrantRole       commands.GrantRoleUseCase
	RevokeRole      commands.RevokeRoleUseCase
	DelegateRole    commands.CreateDelegationUseCase
	Logger          *slog.Logger
}

func (h Handler) CheckPermissionHandler(
	ctx context.Context,
	userID string,
	request httptransport.CheckPermissionRequest,
) (httptransport.CheckPermissionResponse, error) {
	decision, err := h.CheckPermission.Execute(ctx, queries.CheckPermissionQuery{
		UserID:       userID,
		Permission:   request.Permission,
		ResourceType: request.ResourceType,
		ResourceID:   request.ResourceID,
	})
	if err != nil {
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

func (h Handler) CheckBatchHandler(
	ctx context.Context,
	userID string,
	request httptransport.CheckBatchRequest,
) (httptransport.CheckBatchResponse, error) {
	decisions, err := h.CheckBatch.Execute(ctx, queries.CheckPermissionsBatchQuery{
		UserID:      userID,
		Permissions: request.Permissions,
	})
	if err != nil {
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

func (h Handler) ListUserRolesHandler(ctx context.Context, userID string) (httptransport.ListUserRolesResponse, error) {
	roles, err := h.ListRoles.Execute(ctx, userID)
	if err != nil {
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

func (h Handler) GrantRoleHandler(
	ctx context.Context,
	userID string,
	adminID string,
	idempotencyKey string,
	request httptransport.GrantRoleRequest,
) (httptransport.GrantRoleResponse, error) {
	result, err := h.GrantRole.Execute(ctx, commands.GrantRoleCommand{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		RoleID:         request.RoleID,
		AdminID:        adminID,
		Reason:         request.Reason,
		ExpiresAt:      request.ExpiresAt,
	})
	if err != nil {
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

func (h Handler) RevokeRoleHandler(
	ctx context.Context,
	userID string,
	adminID string,
	idempotencyKey string,
	request httptransport.RevokeRoleRequest,
) (httptransport.RevokeRoleResponse, error) {
	result, err := h.RevokeRole.Execute(ctx, commands.RevokeRoleCommand{
		IdempotencyKey: idempotencyKey,
		UserID:         userID,
		RoleID:         request.RoleID,
		AdminID:        adminID,
		Reason:         request.Reason,
	})
	if err != nil {
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

func (h Handler) CreateDelegationHandler(
	ctx context.Context,
	idempotencyKey string,
	request httptransport.CreateDelegationRequest,
) (httptransport.CreateDelegationResponse, error) {
	result, err := h.DelegateRole.Execute(ctx, commands.CreateDelegationCommand{
		IdempotencyKey: idempotencyKey,
		FromAdminID:    request.FromAdminID,
		ToAdminID:      request.ToAdminID,
		RoleID:         request.RoleID,
		ExpiresAt:      request.ExpiresAt,
		Reason:         request.Reason,
	})
	if err != nil {
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
