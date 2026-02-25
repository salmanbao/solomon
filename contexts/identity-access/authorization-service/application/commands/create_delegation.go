package commands

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"
)

type CreateDelegationCommand struct {
	IdempotencyKey string
	FromAdminID    string
	ToAdminID      string
	RoleID         string
	ExpiresAt      time.Time
	Reason         string
}

type CreateDelegationResult struct {
	Delegation entities.Delegation `json:"delegation"`
	AuditLogID string              `json:"audit_log_id"`
	Replayed   bool                `json:"replayed"`
}

type CreateDelegationUseCase struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	IDGenerator    ports.IDGenerator
	Clock          ports.Clock
	IdempotencyTTL time.Duration
}

func (u CreateDelegationUseCase) Execute(ctx context.Context, cmd CreateDelegationCommand) (CreateDelegationResult, error) {
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
		return CreateDelegationResult{}, err
	}
	if found {
		if existing.RequestHash != requestHash {
			return CreateDelegationResult{}, domainerrors.ErrIdempotencyConflict
		}
		var replay CreateDelegationResult
		if err := json.Unmarshal(existing.ResponsePayload, &replay); err != nil {
			return CreateDelegationResult{}, err
		}
		replay.Replayed = true
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
		return CreateDelegationResult{}, err
	}

	result := CreateDelegationResult{
		Delegation: mutation.Delegation,
		AuditLogID: mutation.AuditLogID,
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return CreateDelegationResult{}, err
	}

	if err := u.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             idempotencyKey,
		Operation:       "create_delegation",
		RequestHash:     requestHash,
		ResponsePayload: payload,
		ExpiresAt:       now.Add(u.idempotencyTTL()),
	}); err != nil {
		return CreateDelegationResult{}, err
	}

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
