package commands

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/identity-access/authorization-service/application"
	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"
)

// GrantRoleCommand contains transport-agnostic input for role assignment.
type GrantRoleCommand struct {
	IdempotencyKey string
	UserID         string
	RoleID         string
	AdminID        string
	Reason         string
	ExpiresAt      *time.Time
}

// GrantRoleResult captures assignment/audit identifiers and replay status.
type GrantRoleResult struct {
	Assignment entities.RoleAssignment `json:"assignment"`
	AuditLogID string                  `json:"audit_log_id"`
	Replayed   bool                    `json:"replayed"`
}

// GrantRoleUseCase coordinates idempotent role assignment workflow.
type GrantRoleUseCase struct {
	Repository      ports.Repository
	Idempotency     ports.IdempotencyStore
	PermissionCache ports.PermissionCache
	Clock           ports.Clock
	IDGenerator     ports.IDGenerator
	IdempotencyTTL  time.Duration
	Logger          *slog.Logger
}

// Execute validates command input, enforces idempotency, writes mutation, and stores replay payload.
func (u GrantRoleUseCase) Execute(ctx context.Context, cmd GrantRoleCommand) (GrantRoleResult, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("grant role started",
		"event", "authz_grant_role_started",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", cmd.UserID,
		"admin_id", cmd.AdminID,
		"role_id", cmd.RoleID,
	)

	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return GrantRoleResult{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(cmd.UserID) == "" {
		return GrantRoleResult{}, domainerrors.ErrInvalidUserID
	}
	if strings.TrimSpace(cmd.RoleID) == "" {
		return GrantRoleResult{}, domainerrors.ErrInvalidRoleID
	}
	if strings.TrimSpace(cmd.AdminID) == "" {
		return GrantRoleResult{}, domainerrors.ErrInvalidAdminID
	}

	requestHash, err := hashRequest(struct {
		UserID    string     `json:"user_id"`
		RoleID    string     `json:"role_id"`
		AdminID   string     `json:"admin_id"`
		Reason    string     `json:"reason"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}{
		UserID:    cmd.UserID,
		RoleID:    cmd.RoleID,
		AdminID:   cmd.AdminID,
		Reason:    cmd.Reason,
		ExpiresAt: cmd.ExpiresAt,
	})
	if err != nil {
		logger.Error("grant role request hash failed",
			"event", "authz_grant_role_hash_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return GrantRoleResult{}, err
	}

	idempotencyKey := "authz_idempotency:" + cmd.IdempotencyKey
	now := u.now()

	existing, found, err := u.Idempotency.GetRecord(ctx, idempotencyKey, now)
	if err != nil {
		logger.Error("grant role idempotency lookup failed",
			"event", "authz_grant_role_idempotency_get_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return GrantRoleResult{}, err
	}
	if found {
		if existing.RequestHash != requestHash {
			return GrantRoleResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replay GrantRoleResult
		if err := json.Unmarshal(existing.ResponsePayload, &replay); err != nil {
			logger.Error("grant role replay decode failed",
				"event", "authz_grant_role_replay_decode_failed",
				"module", "identity-access/authorization-service",
				"layer", "application",
				"user_id", cmd.UserID,
				"admin_id", cmd.AdminID,
				"role_id", cmd.RoleID,
				"error", err.Error(),
			)
			return GrantRoleResult{}, err
		}
		replay.Replayed = true
		logger.Info("grant role replayed",
			"event", "authz_grant_role_replayed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
		)
		return replay, nil
	}

	if err := ensureActorPermission(ctx, u.Repository, cmd.AdminID, "user.grant_role", now); err != nil {
		return GrantRoleResult{}, err
	}

	assignmentID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return GrantRoleResult{}, err
	}
	auditLogID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return GrantRoleResult{}, err
	}
	outboxID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return GrantRoleResult{}, err
	}

	mutation, err := u.Repository.GrantRole(ctx, ports.GrantRoleInput{
		AssignmentID: assignmentID,
		AuditLogID:   auditLogID,
		OutboxID:     outboxID,
		UserID:       cmd.UserID,
		RoleID:       cmd.RoleID,
		AdminID:      cmd.AdminID,
		Reason:       cmd.Reason,
		AssignedAt:   now,
		ExpiresAt:    cmd.ExpiresAt,
	})
	if err != nil {
		logger.Error("grant role write failed",
			"event", "authz_grant_role_write_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return GrantRoleResult{}, err
	}

	if u.PermissionCache != nil {
		if err := u.PermissionCache.Invalidate(ctx, cmd.UserID); err != nil {
			logger.Warn("permission cache invalidate failed after role grant",
				"event", "authz_cache_invalidation_failed",
				"module", "identity-access/authorization-service",
				"layer", "application",
				"user_id", cmd.UserID,
				"error", err.Error(),
			)
		}
	}

	result := GrantRoleResult{
		Assignment: mutation.Assignment,
		AuditLogID: mutation.AuditLogID,
	}
	responsePayload, err := json.Marshal(result)
	if err != nil {
		logger.Error("grant role response encode failed",
			"event", "authz_grant_role_response_encode_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return GrantRoleResult{}, err
	}

	if err := u.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             idempotencyKey,
		Operation:       "grant_role",
		RequestHash:     requestHash,
		ResponsePayload: responsePayload,
		ExpiresAt:       now.Add(u.idempotencyTTL()),
	}); err != nil {
		logger.Error("grant role idempotency save failed",
			"event", "authz_grant_role_idempotency_put_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"admin_id", cmd.AdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return GrantRoleResult{}, err
	}

	logger.Info("grant role completed",
		"event", "authz_grant_role_completed",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"user_id", cmd.UserID,
		"admin_id", cmd.AdminID,
		"role_id", cmd.RoleID,
		"assignment_id", result.Assignment.AssignmentID,
	)

	return result, nil
}

func (u GrantRoleUseCase) idempotencyTTL() time.Duration {
	if u.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return u.IdempotencyTTL
}

func (u GrantRoleUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
