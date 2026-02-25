package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	authzOutboxPending   = "pending"
	authzOutboxPublished = "published"
)

type Repository struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewRepository builds the GORM-backed authorization repository adapter.
func NewRepository(db *gorm.DB, logger *slog.Logger) *Repository {
	if logger == nil {
		logger = slog.Default()
	}
	return &Repository{db: db, logger: logger}
}

// ListEffectivePermissions resolves active role + delegation permissions for a user.
func (r *Repository) ListEffectivePermissions(ctx context.Context, userID string, now time.Time) ([]string, error) {
	type permissionRow struct {
		PermissionKey string `gorm:"column:permission_key"`
	}

	assignmentRows := make([]permissionRow, 0)
	if err := r.db.WithContext(ctx).
		Table("role_assignments AS ra").
		Select("DISTINCT rp.permission_key").
		Joins("JOIN role_permissions rp ON rp.role_id = ra.role_id").
		Where("ra.user_id = ? AND ra.is_active = ? AND (ra.expires_at IS NULL OR ra.expires_at > ?)", userID, true, now.UTC()).
		Scan(&assignmentRows).Error; err != nil {
		return nil, err
	}

	delegationRows := make([]permissionRow, 0)
	if err := r.db.WithContext(ctx).
		Table("role_delegations AS rd").
		Select("DISTINCT rp.permission_key").
		Joins("JOIN role_permissions rp ON rp.role_id = rd.role_id").
		Where("rd.to_admin_id = ? AND rd.is_active = ? AND rd.expires_at > ?", userID, true, now.UTC()).
		Scan(&delegationRows).Error; err != nil {
		return nil, err
	}

	merged := make(map[string]struct{}, len(assignmentRows)+len(delegationRows))
	for _, row := range assignmentRows {
		merged[row.PermissionKey] = struct{}{}
	}
	for _, row := range delegationRows {
		merged[row.PermissionKey] = struct{}{}
	}

	permissions := make([]string, 0, len(merged))
	for permission := range merged {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions, nil
}

// ListUserRoles returns role assignments used by role-management endpoints.
func (r *Repository) ListUserRoles(ctx context.Context, userID string, now time.Time) ([]entities.RoleAssignment, error) {
	var rows []roleAssignmentModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("(is_active = ? AND (expires_at IS NULL OR expires_at > ?)) OR is_active = ?", true, now.UTC(), false).
		Order("assigned_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]entities.RoleAssignment, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

// GrantRole persists role assignment, audit log, and outbox row in one transaction.
func (r *Repository) GrantRole(ctx context.Context, input ports.GrantRoleInput) (ports.RoleMutationResult, error) {
	var result ports.RoleMutationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role roleModel
		if err := tx.Where("role_id = ?", input.RoleID).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.ErrRoleNotFound
			}
			return err
		}

		var activeCount int64
		if err := tx.Model(&roleAssignmentModel{}).
			Where("user_id = ? AND role_id = ? AND is_active = ? AND (expires_at IS NULL OR expires_at > ?)", input.UserID, input.RoleID, true, input.AssignedAt.UTC()).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return domainerrors.ErrRoleAlreadyAssigned
		}

		assignment := roleAssignmentModel{
			AssignmentID: input.AssignmentID,
			UserID:       input.UserID,
			RoleID:       input.RoleID,
			RoleName:     role.RoleName,
			AssignedBy:   input.AdminID,
			Reason:       input.Reason,
			AssignedAt:   input.AssignedAt.UTC(),
			ExpiresAt:    input.ExpiresAt,
			IsActive:     true,
		}
		if err := tx.Create(&assignment).Error; err != nil {
			return mapWriteError(err)
		}

		if err := tx.Create(&permissionAuditModel{
			AuditID:     input.AuditLogID,
			ActionType:  "role_granted",
			UserID:      input.UserID,
			AdminID:     input.AdminID,
			RoleID:      input.RoleID,
			Reason:      input.Reason,
			PerformedAt: input.AssignedAt.UTC(),
		}).Error; err != nil {
			return mapWriteError(err)
		}

		eventPayload, err := buildPolicyChangedPayload(input.UserID, input.RoleID, "role_granted")
		if err != nil {
			return err
		}
		outbox, err := buildOutboxMessage(input.OutboxID, "authz.policy_changed", input.UserID, eventPayload, input.AssignedAt.UTC())
		if err != nil {
			return err
		}
		if err := tx.Create(&outbox).Error; err != nil {
			return mapWriteError(err)
		}

		result = ports.RoleMutationResult{
			Assignment: assignment.toEntity(),
			AuditLogID: input.AuditLogID,
		}
		return nil
	})
	if err != nil {
		return ports.RoleMutationResult{}, err
	}
	return result, nil
}

// RevokeRole deactivates role assignment, writes audit, and appends outbox row.
func (r *Repository) RevokeRole(ctx context.Context, input ports.RevokeRoleInput) (ports.RoleMutationResult, error) {
	var result ports.RoleMutationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var assignment roleAssignmentModel
		if err := tx.Where("user_id = ? AND role_id = ? AND is_active = ?", input.UserID, input.RoleID, true).
			Order("assigned_at DESC").
			First(&assignment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.ErrRoleNotAssigned
			}
			return err
		}

		revokedAt := input.RevokedAt.UTC()
		if err := tx.Model(&roleAssignmentModel{}).
			Where("assignment_id = ?", assignment.AssignmentID).
			Updates(map[string]any{
				"is_active":  false,
				"revoked_at": revokedAt,
				"updated_at": revokedAt,
			}).Error; err != nil {
			return err
		}
		assignment.IsActive = false
		assignment.RevokedAt = &revokedAt
		assignment.UpdatedAt = revokedAt

		if err := tx.Create(&permissionAuditModel{
			AuditID:     input.AuditLogID,
			ActionType:  "role_revoked",
			UserID:      input.UserID,
			AdminID:     input.AdminID,
			RoleID:      input.RoleID,
			Reason:      input.Reason,
			PerformedAt: revokedAt,
		}).Error; err != nil {
			return mapWriteError(err)
		}

		eventPayload, err := buildPolicyChangedPayload(input.UserID, input.RoleID, "role_revoked")
		if err != nil {
			return err
		}
		outbox, err := buildOutboxMessage(input.OutboxID, "authz.policy_changed", input.UserID, eventPayload, revokedAt)
		if err != nil {
			return err
		}
		if err := tx.Create(&outbox).Error; err != nil {
			return mapWriteError(err)
		}

		result = ports.RoleMutationResult{
			Assignment: assignment.toEntity(),
			AuditLogID: input.AuditLogID,
		}
		return nil
	})
	if err != nil {
		return ports.RoleMutationResult{}, err
	}
	return result, nil
}

// CreateDelegation persists delegation, writes audit, and appends outbox row.
func (r *Repository) CreateDelegation(ctx context.Context, input ports.DelegationInput) (ports.DelegationMutationResult, error) {
	var result ports.DelegationMutationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.FromAdminID == input.ToAdminID || !input.ExpiresAt.After(input.DelegatedAt) {
			return domainerrors.ErrInvalidDelegation
		}

		var role roleModel
		if err := tx.Where("role_id = ?", input.RoleID).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.ErrRoleNotFound
			}
			return err
		}
		_ = role

		delegation := roleDelegationModel{
			DelegationID: input.DelegationID,
			FromAdminID:  input.FromAdminID,
			ToAdminID:    input.ToAdminID,
			RoleID:       input.RoleID,
			Reason:       input.Reason,
			DelegatedAt:  input.DelegatedAt.UTC(),
			ExpiresAt:    input.ExpiresAt.UTC(),
			IsActive:     true,
		}
		if err := tx.Create(&delegation).Error; err != nil {
			return mapWriteError(err)
		}

		if err := tx.Create(&permissionAuditModel{
			AuditID:     input.AuditLogID,
			ActionType:  "delegated",
			UserID:      input.ToAdminID,
			AdminID:     input.FromAdminID,
			RoleID:      input.RoleID,
			Reason:      input.Reason,
			PerformedAt: input.DelegatedAt.UTC(),
		}).Error; err != nil {
			return mapWriteError(err)
		}

		eventPayload, err := buildPolicyChangedPayload(input.ToAdminID, input.RoleID, "delegated")
		if err != nil {
			return err
		}
		outbox, err := buildOutboxMessage(input.OutboxID, "authz.policy_changed", input.ToAdminID, eventPayload, input.DelegatedAt.UTC())
		if err != nil {
			return err
		}
		if err := tx.Create(&outbox).Error; err != nil {
			return mapWriteError(err)
		}

		result = ports.DelegationMutationResult{
			Delegation: delegation.toEntity(),
			AuditLogID: input.AuditLogID,
		}
		return nil
	})
	if err != nil {
		return ports.DelegationMutationResult{}, err
	}
	return result, nil
}

// GetRecord loads an idempotency record and evicts expired entries.
func (r *Repository) GetRecord(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	var row idempotencyModel
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.IdempotencyRecord{}, false, nil
		}
		return ports.IdempotencyRecord{}, false, err
	}
	if !row.ExpiresAt.After(now.UTC()) {
		if err := r.db.WithContext(ctx).Where("key = ?", key).Delete(&idempotencyModel{}).Error; err != nil {
			return ports.IdempotencyRecord{}, false, err
		}
		return ports.IdempotencyRecord{}, false, nil
	}
	return row.toPort(), true, nil
}

// PutRecord inserts a new idempotency record and checks request-hash collisions.
func (r *Repository) PutRecord(ctx context.Context, record ports.IdempotencyRecord) error {
	row := idempotencyFromPort(record)
	createResult := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).
		Create(&row)
	if createResult.Error != nil {
		return createResult.Error
	}
	if createResult.RowsAffected > 0 {
		return nil
	}

	var existing idempotencyModel
	if err := r.db.WithContext(ctx).Where("key = ?", row.Key).First(&existing).Error; err != nil {
		return err
	}
	if existing.RequestHash != row.RequestHash {
		return domainerrors.ErrIdempotencyConflict
	}
	return nil
}

// ListPendingOutbox loads unsent outbox rows oldest-first.
func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]ports.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []authzOutboxModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", authzOutboxPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ports.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, nil
}

// MarkOutboxPublished marks one outbox row as published.
func (r *Repository) MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&authzOutboxModel{}).
		Where("outbox_id = ?", outboxID).
		Updates(map[string]any{
			"status":       authzOutboxPublished,
			"published_at": publishedAt.UTC(),
			"updated_at":   publishedAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("outbox row not found")
	}
	return nil
}

// ReserveEvent inserts event dedupe record or validates duplicate payload hash.
func (r *Repository) ReserveEvent(
	ctx context.Context,
	eventID string,
	payloadHash string,
	expiresAt time.Time,
) (bool, error) {
	row := eventDedupModel{
		EventID:     eventID,
		PayloadHash: payloadHash,
		ProcessedAt: time.Now().UTC(),
		ExpiresAt:   expiresAt.UTC(),
	}
	createResult := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(&row)
	if createResult.Error != nil {
		return false, createResult.Error
	}
	if createResult.RowsAffected > 0 {
		return false, nil
	}

	var existing eventDedupModel
	if err := r.db.WithContext(ctx).
		Select("payload_hash").
		Where("event_id = ?", eventID).
		First(&existing).Error; err != nil {
		return false, err
	}
	if existing.PayloadHash != payloadHash {
		return false, domainerrors.ErrIdempotencyConflict
	}
	return true, nil
}

type roleModel struct {
	RoleID   string `gorm:"column:role_id;primaryKey"`
	RoleName string `gorm:"column:role_name"`
}

func (roleModel) TableName() string {
	return "roles"
}

type roleAssignmentModel struct {
	AssignmentID string     `gorm:"column:assignment_id;primaryKey"`
	UserID       string     `gorm:"column:user_id"`
	RoleID       string     `gorm:"column:role_id"`
	RoleName     string     `gorm:"column:role_name"`
	AssignedBy   string     `gorm:"column:assigned_by"`
	Reason       string     `gorm:"column:reason"`
	AssignedAt   time.Time  `gorm:"column:assigned_at"`
	ExpiresAt    *time.Time `gorm:"column:expires_at"`
	IsActive     bool       `gorm:"column:is_active"`
	RevokedAt    *time.Time `gorm:"column:revoked_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (roleAssignmentModel) TableName() string {
	return "role_assignments"
}

func (m roleAssignmentModel) toEntity() entities.RoleAssignment {
	return entities.RoleAssignment{
		AssignmentID: m.AssignmentID,
		UserID:       m.UserID,
		RoleID:       m.RoleID,
		RoleName:     m.RoleName,
		AssignedBy:   m.AssignedBy,
		Reason:       m.Reason,
		AssignedAt:   m.AssignedAt.UTC(),
		ExpiresAt:    m.ExpiresAt,
		IsActive:     m.IsActive,
		RevokedAt:    m.RevokedAt,
	}
}

type roleDelegationModel struct {
	DelegationID string    `gorm:"column:delegation_id;primaryKey"`
	FromAdminID  string    `gorm:"column:from_admin_id"`
	ToAdminID    string    `gorm:"column:to_admin_id"`
	RoleID       string    `gorm:"column:role_id"`
	Reason       string    `gorm:"column:reason"`
	DelegatedAt  time.Time `gorm:"column:delegated_at"`
	ExpiresAt    time.Time `gorm:"column:expires_at"`
	IsActive     bool      `gorm:"column:is_active"`
}

func (roleDelegationModel) TableName() string {
	return "role_delegations"
}

func (m roleDelegationModel) toEntity() entities.Delegation {
	return entities.Delegation{
		DelegationID: m.DelegationID,
		FromAdminID:  m.FromAdminID,
		ToAdminID:    m.ToAdminID,
		RoleID:       m.RoleID,
		Reason:       m.Reason,
		DelegatedAt:  m.DelegatedAt.UTC(),
		ExpiresAt:    m.ExpiresAt.UTC(),
		IsActive:     m.IsActive,
	}
}

type permissionAuditModel struct {
	AuditID     string    `gorm:"column:audit_id;primaryKey"`
	ActionType  string    `gorm:"column:action_type"`
	UserID      string    `gorm:"column:user_id"`
	AdminID     string    `gorm:"column:admin_id"`
	RoleID      string    `gorm:"column:role_id"`
	Reason      string    `gorm:"column:reason"`
	PerformedAt time.Time `gorm:"column:performed_at"`
}

func (permissionAuditModel) TableName() string {
	return "permission_audit"
}

type authzOutboxModel struct {
	OutboxID     string     `gorm:"column:outbox_id;primaryKey"`
	EventType    string     `gorm:"column:event_type"`
	PartitionKey string     `gorm:"column:partition_key"`
	Payload      []byte     `gorm:"column:payload"`
	Status       string     `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
	RetryCount   int        `gorm:"column:retry_count"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (authzOutboxModel) TableName() string {
	return "authz_outbox"
}

func (m authzOutboxModel) toPort() ports.OutboxMessage {
	return ports.OutboxMessage{
		OutboxID:  m.OutboxID,
		EventType: m.EventType,
		Payload:   append([]byte(nil), m.Payload...),
		CreatedAt: m.CreatedAt.UTC(),
	}
}

type idempotencyModel struct {
	Key             string    `gorm:"column:key;primaryKey"`
	Operation       string    `gorm:"column:operation"`
	RequestHash     string    `gorm:"column:request_hash"`
	ResponsePayload []byte    `gorm:"column:response_payload"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (idempotencyModel) TableName() string {
	return "authz_idempotency"
}

func idempotencyFromPort(record ports.IdempotencyRecord) idempotencyModel {
	return idempotencyModel{
		Key:             record.Key,
		Operation:       record.Operation,
		RequestHash:     record.RequestHash,
		ResponsePayload: append([]byte(nil), record.ResponsePayload...),
		ExpiresAt:       record.ExpiresAt.UTC(),
		CreatedAt:       time.Now().UTC(),
	}
}

func (m idempotencyModel) toPort() ports.IdempotencyRecord {
	return ports.IdempotencyRecord{
		Key:             m.Key,
		Operation:       m.Operation,
		RequestHash:     m.RequestHash,
		ResponsePayload: append([]byte(nil), m.ResponsePayload...),
		ExpiresAt:       m.ExpiresAt.UTC(),
	}
}

type eventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	PayloadHash string    `gorm:"column:payload_hash"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
}

func (eventDedupModel) TableName() string {
	return "authz_event_dedup"
}

func buildPolicyChangedPayload(userID string, roleID string, action string) ([]byte, error) {
	return json.Marshal(map[string]string{
		"user_id":     userID,
		"role_id":     roleID,
		"action_type": action,
	})
}

func buildOutboxMessage(outboxID string, eventType string, partitionKey string, payload []byte, at time.Time) (authzOutboxModel, error) {
	event := ports.PolicyChangedEvent{
		EventID:          outboxID,
		EventType:        eventType,
		OccurredAt:       at.UTC(),
		SourceService:    "authorization-service",
		SchemaVersion:    1,
		PartitionKeyPath: "user_id",
		PartitionKey:     partitionKey,
		Data:             payload,
	}
	envelope, err := json.Marshal(event)
	if err != nil {
		return authzOutboxModel{}, err
	}
	return authzOutboxModel{
		OutboxID:     outboxID,
		EventType:    eventType,
		PartitionKey: partitionKey,
		Payload:      envelope,
		Status:       authzOutboxPending,
		CreatedAt:    at.UTC(),
		RetryCount:   0,
		UpdatedAt:    at.UTC(),
	}, nil
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503":
			if pgErr.ConstraintName == "role_assignments_role_id_fkey" || pgErr.ConstraintName == "role_delegations_role_id_fkey" {
				return domainerrors.ErrRoleNotFound
			}
			return domainerrors.ErrUserNotFound
		case "23505":
			if pgErr.ConstraintName == "role_assignments_unique_active" {
				return domainerrors.ErrRoleAlreadyAssigned
			}
			return domainerrors.ErrIdempotencyConflict
		}
	}
	return err
}
