package postgresadapter

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	domainerrors "solomon/contexts/moderation-safety/abuse-prevention-service/domain/errors"
	"solomon/contexts/moderation-safety/abuse-prevention-service/ports"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	lockoutStatusActive   = "active"
	lockoutStatusReleased = "released"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ReleaseLockout(
	ctx context.Context,
	userID string,
	releasedAt time.Time,
) (ports.LockoutRelease, error) {
	userID = strings.TrimSpace(userID)
	releasedAt = releasedAt.UTC()

	var row lockoutHistoryModel
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND status = ?", userID, lockoutStatusActive).
			Order("locked_at DESC").
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.ErrThreatNotFound
			}
			return err
		}

		if err := tx.Model(&lockoutHistoryModel{}).
			Where("lockout_id = ?", row.LockoutID).
			Updates(map[string]any{
				"status":      lockoutStatusReleased,
				"released_at": releasedAt,
				"updated_at":  releasedAt,
			}).Error; err != nil {
			return err
		}

		row.Status = lockoutStatusReleased
		row.ReleasedAt = &releasedAt
		row.UpdatedAt = releasedAt
		return nil
	})
	if err != nil {
		return ports.LockoutRelease{}, err
	}

	return ports.LockoutRelease{
		ThreatID:   row.ThreatID,
		UserID:     row.UserID,
		Status:     row.Status,
		ReleasedAt: releasedAt,
	}, nil
}

func (r *Repository) AppendAuditLog(ctx context.Context, row ports.AuditLog) error {
	model := auditLogModel{
		AuditID:       strings.TrimSpace(row.AuditID),
		ActorID:       strings.TrimSpace(row.ActorID),
		Action:        strings.TrimSpace(row.Action),
		TargetID:      strings.TrimSpace(row.TargetID),
		Justification: strings.TrimSpace(row.Justification),
		OccurredAt:    row.OccurredAt.UTC(),
		SourceIP:      strings.TrimSpace(row.SourceIP),
		CorrelationID: strings.TrimSpace(row.CorrelationID),
	}
	if model.AuditID == "" {
		model.AuditID = "abuse_audit_" + model.OccurredAt.Format("20060102150405.000000000")
	}
	return r.db.WithContext(ctx).Create(&model).Error
}

func (r *Repository) ListRecentAuditLogs(ctx context.Context, limit int) ([]ports.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}

	rows := make([]auditLogModel, 0, limit)
	if err := r.db.WithContext(ctx).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]ports.AuditLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.AuditLog{
			AuditID:       row.AuditID,
			ActorID:       row.ActorID,
			Action:        row.Action,
			TargetID:      row.TargetID,
			Justification: row.Justification,
			OccurredAt:    row.OccurredAt.UTC(),
			SourceIP:      row.SourceIP,
			CorrelationID: row.CorrelationID,
		})
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}

	var row idempotencyModel
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if now.UTC().After(row.ExpiresAt.UTC()) {
		if err := r.db.WithContext(ctx).Where("key = ?", key).Delete(&idempotencyModel{}).Error; err != nil {
			return nil, err
		}
		return nil, nil
	}

	record := ports.IdempotencyRecord{
		Key:          row.Key,
		RequestHash:  row.RequestHash,
		ResponseBody: slices.Clone(row.ResponseBody),
		ExpiresAt:    row.ExpiresAt.UTC(),
	}
	return &record, nil
}

func (r *Repository) Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error {
	key = strings.TrimSpace(key)
	requestHash = strings.TrimSpace(requestHash)
	now := time.Now().UTC()
	expiresAt = expiresAt.UTC()

	createResult := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).
		Create(&idempotencyModel{
			Key:         key,
			RequestHash: requestHash,
			ExpiresAt:   expiresAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	if createResult.Error != nil {
		return createResult.Error
	}
	if createResult.RowsAffected > 0 {
		return nil
	}

	var existing idempotencyModel
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&existing).Error; err != nil {
		return err
	}
	if now.After(existing.ExpiresAt.UTC()) {
		return r.db.WithContext(ctx).Model(&idempotencyModel{}).
			Where("key = ?", key).
			Updates(map[string]any{
				"request_hash":  requestHash,
				"response_body": []byte{},
				"expires_at":    expiresAt,
				"updated_at":    now,
			}).Error
	}
	if existing.RequestHash != requestHash {
		return domainerrors.ErrIdempotencyConflict
	}
	return nil
}

func (r *Repository) Complete(ctx context.Context, key string, responseBody []byte, at time.Time) error {
	key = strings.TrimSpace(key)
	at = at.UTC()

	var existing idempotencyModel
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	updates := map[string]any{
		"response_body": slices.Clone(responseBody),
		"updated_at":    at,
	}
	if at.After(existing.ExpiresAt.UTC()) {
		updates["expires_at"] = at.Add(7 * 24 * time.Hour)
	}
	return r.db.WithContext(ctx).Model(&idempotencyModel{}).
		Where("key = ?", key).
		Updates(updates).Error
}

type lockoutHistoryModel struct {
	LockoutID  string     `gorm:"column:lockout_id;primaryKey"`
	ThreatID   string     `gorm:"column:threat_id"`
	UserID     string     `gorm:"column:user_id"`
	Status     string     `gorm:"column:status"`
	Reason     string     `gorm:"column:reason"`
	LockedAt   time.Time  `gorm:"column:locked_at"`
	ReleasedAt *time.Time `gorm:"column:released_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at"`
}

func (lockoutHistoryModel) TableName() string {
	return "abuse_lockout_history"
}

type auditLogModel struct {
	AuditID       string    `gorm:"column:audit_id;primaryKey"`
	ActorID       string    `gorm:"column:actor_id"`
	Action        string    `gorm:"column:action"`
	TargetID      string    `gorm:"column:target_id"`
	Justification string    `gorm:"column:justification"`
	OccurredAt    time.Time `gorm:"column:occurred_at"`
	SourceIP      string    `gorm:"column:source_ip"`
	CorrelationID string    `gorm:"column:correlation_id"`
}

func (auditLogModel) TableName() string {
	return "abuse_audit_log"
}

type idempotencyModel struct {
	Key          string    `gorm:"column:key;primaryKey"`
	RequestHash  string    `gorm:"column:request_hash"`
	ResponseBody []byte    `gorm:"column:response_body"`
	ExpiresAt    time.Time `gorm:"column:expires_at"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (idempotencyModel) TableName() string {
	return "abuse_idempotency"
}

var _ ports.Repository = (*Repository)(nil)
var _ ports.IdempotencyStore = (*Repository)(nil)
