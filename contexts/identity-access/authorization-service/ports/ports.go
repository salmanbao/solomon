package ports

import (
	"context"
	"encoding/json"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type PermissionCache interface {
	Get(ctx context.Context, userID string, now time.Time) ([]string, bool, error)
	Set(ctx context.Context, userID string, permissions []string, expiresAt time.Time) error
	Invalidate(ctx context.Context, userID string) error
}

type IdempotencyRecord struct {
	Key             string
	Operation       string
	RequestHash     string
	ResponsePayload []byte
	ExpiresAt       time.Time
}

type IdempotencyStore interface {
	GetRecord(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	PutRecord(ctx context.Context, record IdempotencyRecord) error
}

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

type RevokeRoleInput struct {
	AuditLogID string
	OutboxID   string
	UserID     string
	RoleID     string
	AdminID    string
	Reason     string
	RevokedAt  time.Time
}

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

type RoleMutationResult struct {
	Assignment entities.RoleAssignment
	AuditLogID string
}

type DelegationMutationResult struct {
	Delegation entities.Delegation
	AuditLogID string
}

type Repository interface {
	ListEffectivePermissions(ctx context.Context, userID string, now time.Time) ([]string, error)
	ListUserRoles(ctx context.Context, userID string, now time.Time) ([]entities.RoleAssignment, error)
	GrantRole(ctx context.Context, input GrantRoleInput) (RoleMutationResult, error)
	RevokeRole(ctx context.Context, input RevokeRoleInput) (RoleMutationResult, error)
	CreateDelegation(ctx context.Context, input DelegationInput) (DelegationMutationResult, error)
}

type OutboxMessage struct {
	OutboxID  string
	EventType string
	Payload   []byte
	CreatedAt time.Time
}

type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error
}

type PolicyChangedEvent struct {
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	OccurredAt   time.Time       `json:"occurred_at"`
	Source       string          `json:"source"`
	Schema       int             `json:"schema_version"`
	PartitionKey string          `json:"partition_key"`
	Data         json.RawMessage `json:"data"`
}

type PolicyChangedPublisher interface {
	PublishPolicyChanged(ctx context.Context, event PolicyChangedEvent) error
}

type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}
