package ports

import (
	"context"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
	contractsv1 "solomon/contracts/gen/events/v1"
)

// Clock abstracts current time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts UUID generation for commands/outbox rows.
type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

// PermissionCache stores effective permissions with TTL semantics.
type PermissionCache interface {
	Get(ctx context.Context, userID string, now time.Time) ([]string, bool, error)
	Set(ctx context.Context, userID string, permissions []string, expiresAt time.Time) error
	Invalidate(ctx context.Context, userID string) error
}

// IdempotencyRecord stores request hash and previous response payload.
type IdempotencyRecord struct {
	Key             string
	Operation       string
	RequestHash     string
	ResponsePayload []byte
	ExpiresAt       time.Time
}

// IdempotencyStore guarantees replay/conflict behavior for mutating endpoints.
type IdempotencyStore interface {
	GetRecord(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	PutRecord(ctx context.Context, record IdempotencyRecord) error
}

// GrantRoleInput is persisted atomically with audit and outbox records.
type GrantRoleInput struct {
	AssignmentID string
	AuditLogID   string
	OutboxID     string
	UserID       string
	RoleID       string
	AdminID      string
	Reason       string
	AssignedAt   time.Time
	ExpiresAt    *time.Time
}

// RevokeRoleInput captures revoke metadata and audit context.
type RevokeRoleInput struct {
	AuditLogID string
	OutboxID   string
	UserID     string
	RoleID     string
	AdminID    string
	Reason     string
	RevokedAt  time.Time
}

// DelegationInput captures temporary admin delegation metadata.
type DelegationInput struct {
	DelegationID string
	AuditLogID   string
	OutboxID     string
	FromAdminID  string
	ToAdminID    string
	RoleID       string
	Reason       string
	DelegatedAt  time.Time
	ExpiresAt    time.Time
}

// RoleMutationResult is returned by grant/revoke repository operations.
type RoleMutationResult struct {
	Assignment entities.RoleAssignment
	AuditLogID string
}

// DelegationMutationResult is returned by delegation repository operations.
type DelegationMutationResult struct {
	Delegation entities.Delegation
	AuditLogID string
}

// Repository is the write/read boundary for authorization domain state.
type Repository interface {
	ListEffectivePermissions(ctx context.Context, userID string, now time.Time) ([]string, error)
	ListUserRoles(ctx context.Context, userID string, now time.Time) ([]entities.RoleAssignment, error)
	GrantRole(ctx context.Context, input GrantRoleInput) (RoleMutationResult, error)
	RevokeRole(ctx context.Context, input RevokeRoleInput) (RoleMutationResult, error)
	CreateDelegation(ctx context.Context, input DelegationInput) (DelegationMutationResult, error)
}

// OutboxMessage represents a pending relay message.
type OutboxMessage struct {
	OutboxID  string
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

// OutboxRepository supports worker relay polling and acknowledgement.
type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error
}

// PolicyChangedEvent reuses the canonical cross-runtime envelope contract.
type PolicyChangedEvent = contractsv1.Envelope

// PolicyChangedPublisher emits policy change events to the event bus adapter.
type PolicyChangedPublisher interface {
	PublishPolicyChanged(ctx context.Context, event PolicyChangedEvent) error
}

// EventDedupStore enforces idempotent processing for consumed events.
type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}
