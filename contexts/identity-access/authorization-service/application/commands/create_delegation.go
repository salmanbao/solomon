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

// CreateDelegationCommand contains input for temporary admin delegation.
type CreateDelegationCommand struct {
	IdempotencyKey string
	FromAdminID    string
	ToAdminID      string
	RoleID         string
	ExpiresAt      time.Time
	Reason         string
}

// CreateDelegationResult captures delegation identifiers and replay status.
type CreateDelegationResult struct {
	Delegation entities.Delegation `json:"delegation"`
	AuditLogID string              `json:"audit_log_id"`
	Replayed   bool                `json:"replayed"`
}

// CreateDelegationUseCase coordinates idempotent delegation creation.
type CreateDelegationUseCase struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	IDGenerator    ports.IDGenerator
	Clock          ports.Clock
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

// Execute validates delegation invariants, writes mutation, and records replay payload.
func (u CreateDelegationUseCase) Execute(ctx context.Context, cmd CreateDelegationCommand) (CreateDelegationResult, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("create delegation started",
		"event", "authz_create_delegation_started",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"from_admin_id", cmd.FromAdminID,
		"to_admin_id", cmd.ToAdminID,
		"role_id", cmd.RoleID,
	)

	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return CreateDelegationResult{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if strings.TrimSpace(cmd.FromAdminID) == "" || strings.TrimSpace(cmd.ToAdminID) == "" {
		return CreateDelegationResult{}, domainerrors.ErrInvalidAdminID
	}
	if strings.TrimSpace(cmd.RoleID) == "" {
		return CreateDelegationResult{}, domainerrors.ErrInvalidRoleID
	}
	if cmd.FromAdminID == cmd.ToAdminID {
		return CreateDelegationResult{}, domainerrors.ErrInvalidDelegation
	}
	if !cmd.ExpiresAt.After(u.now()) {
		return CreateDelegationResult{}, domainerrors.ErrInvalidDelegation
	}

	requestHash, err := hashRequest(struct {
		FromAdminID string    `json:"from_admin_id"`
		ToAdminID   string    `json:"to_admin_id"`
		RoleID      string    `json:"role_id"`
		ExpiresAt   time.Time `json:"expires_at"`
		Reason      string    `json:"reason"`
	}{
		FromAdminID: cmd.FromAdminID,
		ToAdminID:   cmd.ToAdminID,
		RoleID:      cmd.RoleID,
		ExpiresAt:   cmd.ExpiresAt.UTC(),
		Reason:      cmd.Reason,
	})
	if err != nil {
		return CreateDelegationResult{}, err
	}

	idempotencyKey := "authz_idempotency:" + cmd.IdempotencyKey
	now := u.now()
	existing, found, err := u.Idempotency.GetRecord(ctx, idempotencyKey, now)
	if err != nil {
		logger.Error("create delegation idempotency lookup failed",
			"event", "authz_create_delegation_idempotency_get_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"from_admin_id", cmd.FromAdminID,
			"to_admin_id", cmd.ToAdminID,
			"error", err.Error(),
		)
		return CreateDelegationResult{}, err
	}
	if found {
		if existing.RequestHash != requestHash {
			return CreateDelegationResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replay CreateDelegationResult
		if err := json.Unmarshal(existing.ResponsePayload, &replay); err != nil {
			logger.Error("create delegation replay decode failed",
				"event", "authz_create_delegation_replay_decode_failed",
				"module", "identity-access/authorization-service",
				"layer", "application",
				"from_admin_id", cmd.FromAdminID,
				"to_admin_id", cmd.ToAdminID,
				"error", err.Error(),
			)
			return CreateDelegationResult{}, err
		}
		replay.Replayed = true
		logger.Info("create delegation replayed",
			"event", "authz_create_delegation_replayed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"from_admin_id", cmd.FromAdminID,
			"to_admin_id", cmd.ToAdminID,
			"role_id", cmd.RoleID,
		)
		return replay, nil
	}

	delegationID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return CreateDelegationResult{}, err
	}
	auditLogID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return CreateDelegationResult{}, err
	}
	outboxID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return CreateDelegationResult{}, err
	}

	mutation, err := u.Repository.CreateDelegation(ctx, ports.DelegationInput{
		DelegationID: delegationID,
		AuditLogID:   auditLogID,
		OutboxID:     outboxID,
		FromAdminID:  cmd.FromAdminID,
		ToAdminID:    cmd.ToAdminID,
		RoleID:       cmd.RoleID,
		Reason:       cmd.Reason,
		DelegatedAt:  now,
		ExpiresAt:    cmd.ExpiresAt.UTC(),
	})
	if err != nil {
		logger.Error("create delegation write failed",
			"event", "authz_create_delegation_write_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"from_admin_id", cmd.FromAdminID,
			"to_admin_id", cmd.ToAdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return CreateDelegationResult{}, err
	}

	result := CreateDelegationResult{
		Delegation: mutation.Delegation,
		AuditLogID: mutation.AuditLogID,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		logger.Error("create delegation response encode failed",
			"event", "authz_create_delegation_response_encode_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"from_admin_id", cmd.FromAdminID,
			"to_admin_id", cmd.ToAdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return CreateDelegationResult{}, err
	}

	if err := u.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             idempotencyKey,
		Operation:       "create_delegation",
		RequestHash:     requestHash,
		ResponsePayload: payload,
		ExpiresAt:       now.Add(u.idempotencyTTL()),
	}); err != nil {
		logger.Error("create delegation idempotency save failed",
			"event", "authz_create_delegation_idempotency_put_failed",
			"module", "identity-access/authorization-service",
			"layer", "application",
			"from_admin_id", cmd.FromAdminID,
			"to_admin_id", cmd.ToAdminID,
			"role_id", cmd.RoleID,
			"error", err.Error(),
		)
		return CreateDelegationResult{}, err
	}

	logger.Info("create delegation completed",
		"event", "authz_create_delegation_completed",
		"module", "identity-access/authorization-service",
		"layer", "application",
		"delegation_id", result.Delegation.DelegationID,
		"from_admin_id", cmd.FromAdminID,
		"to_admin_id", cmd.ToAdminID,
		"role_id", cmd.RoleID,
	)

	return result, nil
}

func (u CreateDelegationUseCase) idempotencyTTL() time.Duration {
	if u.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return u.IdempotencyTTL
}

func (u CreateDelegationUseCase) now() time.Time {
	if u.Clock != nil {
		return u.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
