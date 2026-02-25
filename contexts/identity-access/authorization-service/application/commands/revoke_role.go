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

type RevokeRoleCommand struct {
	IdempotencyKey string
	UserID         string
	RoleID         string
	AdminID        string
	Reason         string
}

type RevokeRoleResult struct {
	Assignment entities.RoleAssignment `json:"assignment"`
	AuditLogID string                  `json:"audit_log_id"`
	Replayed   bool                    `json:"replayed"`
}

type RevokeRoleUseCase struct {
	Repository      ports.Repository
	Idempotency     ports.IdempotencyStore
	PermissionCache ports.PermissionCache
	Clock           ports.Clock
	IDGenerator     ports.IDGenerator
	IdempotencyTTL  time.Duration
	Logger          *slog.Logger
}

func (u RevokeRoleUseCase) Execute(ctx context.Context, cmd RevokeRoleCommand) (RevokeRoleResult, error) {
	logger := application.ResolveLogger(u.Logger)

	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return RevokeRoleResult{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(cmd.UserID) == "" {
		return RevokeRoleResult{}, domainerrors.ErrInvalidUserID
	}
	if strings.TrimSpace(cmd.RoleID) == "" {
		return RevokeRoleResult{}, domainerrors.ErrInvalidRoleID
	}
	if strings.TrimSpace(cmd.AdminID) == "" {
		return RevokeRoleResult{}, domainerrors.ErrInvalidAdminID
	}

	requestHash, err := hashRequest(struct {
		UserID  string `json:"user_id"`
		RoleID  string `json:"role_id"`
		AdminID string `json:"admin_id"`
		Reason  string `json:"reason"`
	}{
		UserID:  cmd.UserID,
		RoleID:  cmd.RoleID,
		AdminID: cmd.AdminID,
		Reason:  cmd.Reason,
	})
	if err != nil {
		return RevokeRoleResult{}, err
	}

	idempotencyKey := "authz_idempotency:" + cmd.IdempotencyKey
	now := u.now()

	existing, found, err := u.Idempotency.GetRecord(ctx, idempotencyKey, now)
	if err != nil {
		return RevokeRoleResult{}, err
	}
	if found {
		if existing.RequestHash != requestHash {
			return RevokeRoleResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replay RevokeRoleResult
		if err := json.Unmarshal(existing.ResponsePayload, &replay); err != nil {
			return RevokeRoleResult{}, err
		}
		replay.Replayed = true
		return replay, nil
	}

	auditLogID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return RevokeRoleResult{}, err
	}
	outboxID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return RevokeRoleResult{}, err
	}

	mutation, err := u.Repository.RevokeRole(ctx, ports.RevokeRoleInput{
		AuditLogID: auditLogID,
		OutboxID:   outboxID,
		UserID:     cmd.UserID,
		RoleID:     cmd.RoleID,
		AdminID:    cmd.AdminID,
		Reason:     cmd.Reason,
		RevokedAt:  now,
	})
	if err != nil {
		return RevokeRoleResult{}, err
	}

	if err := u.PermissionCache.Invalidate(ctx, cmd.UserID); err != nil {
		logger.Warn("permission cache invalidate failed after role revoke",
			"event", "authz_cache_invalidation_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"user_id", cmd.UserID,
			"error", err.Error(),
		)
	}

	result := RevokeRoleResult{
		Assignment: mutation.Assignment,
		AuditLogID: mutation.AuditLogID,
	}
	responsePayload, err := json.Marshal(result)
	if err != nil {
		return RevokeRoleResult{}, err
	}

	if err := u.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             idempotencyKey,
		Operation:       "revoke_role",
		RequestHash:     requestHash,
		ResponsePayload: responsePayload,
		ExpiresAt:       now.Add(u.idempotencyTTL()),
	}); err != nil {
		return RevokeRoleResult{}, err
	}

	return result, nil
}

func (u RevokeRoleUseCase) idempotencyTTL() time.Duration {
	if u.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return u.IdempotencyTTL
}

func (u RevokeRoleUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
