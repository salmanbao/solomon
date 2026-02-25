package postgresadapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxStatusPending = "pending"
	outboxStatusSent    = "sent"
)

type Repository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewRepository(db *gorm.DB, logger *slog.Logger) *Repository {
	if logger == nil {
		logger = slog.Default()
	}
	return &Repository{
		db:     db,
		logger: logger,
	}
}

func (r *Repository) ListClips(ctx context.Context, filter ports.ClipListFilter) ([]entities.Clip, string, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	tx := r.db.WithContext(ctx).Model(&clipModel{})
	if filter.Status != "" {
		tx = tx.Where("status = ?", string(filter.Status))
	}

	if len(filter.Niches) > 0 {
		niches := make([]string, 0, len(filter.Niches))
		for _, niche := range filter.Niches {
			value := strings.ToLower(strings.TrimSpace(niche))
			if value != "" {
				niches = append(niches, value)
			}
		}
		if len(niches) > 0 {
			tx = tx.Where("LOWER(niche) IN ?", niches)
		}
	}
	tx = applyDurationBucket(tx, filter.DurationBucket)

	tx = applyClipSort(tx, filter.Popularity)

	offset := decodeCursor(filter.Cursor)
	if offset < 0 {
		offset = 0
	}

	var rows []clipModel
	if err := tx.Offset(offset).Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(rows) > limit {
		nextCursor = encodeCursor(offset + limit)
		rows = rows[:limit]
	}

	items := make([]entities.Clip, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}

	return items, nextCursor, nil
}

func (r *Repository) GetClip(ctx context.Context, clipID string) (entities.Clip, error) {
	var row clipModel
	err := r.db.WithContext(ctx).
		Where("clip_id = ?", clipID).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Clip{}, domainerrors.ErrClipNotFound
		}
		return entities.Clip{}, err
	}
	return row.toEntity(), nil
}

func (r *Repository) ListClaimsByUser(ctx context.Context, userID string) ([]entities.Claim, error) {
	var rows []claimModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("claimed_at DESC").
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	items := make([]entities.Claim, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) ListClaimsByClip(ctx context.Context, clipID string) ([]entities.Claim, error) {
	var rows []claimModel
	if err := r.db.WithContext(ctx).
		Where("clip_id = ?", clipID).
		Order("claimed_at DESC").
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	items := make([]entities.Claim, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) GetClaim(ctx context.Context, claimID string) (entities.Claim, error) {
	var row claimModel
	err := r.db.WithContext(ctx).
		Where("claim_id = ?", claimID).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Claim{}, domainerrors.ErrClaimNotFound
		}
		return entities.Claim{}, err
	}
	return row.toEntity(), nil
}

func (r *Repository) GetClaimByRequestID(ctx context.Context, requestID string) (entities.Claim, bool, error) {
	var row claimModel
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Claim{}, false, nil
		}
		return entities.Claim{}, false, err
	}
	return row.toEntity(), true, nil
}

func (r *Repository) CreateClaimWithOutbox(ctx context.Context, claim entities.Claim, event ports.ClaimedEvent) error {
	envelope, err := buildClaimedEnvelope(event)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claimRow := claimModelFromEntity(claim)
		if err := tx.Create(&claimRow).Error; err != nil {
			if isUniqueViolation(err) {
				if constraintName(err) == "clip_claims_unique_request" {
					return domainerrors.ErrDuplicateRequestID
				}
				return domainerrors.ErrRepositoryInvariantBroke
			}
			return err
		}

		outboxRow := outboxModel{
			OutboxID:     event.EventID,
			EventType:    event.EventType,
			PartitionKey: event.PartitionKey,
			Payload:      payload,
			Status:       outboxStatusPending,
			CreatedAt:    event.OccurredAt.UTC(),
		}
		if err := tx.Create(&outboxRow).Error; err != nil {
			if isUniqueViolation(err) {
				return domainerrors.ErrRepositoryInvariantBroke
			}
			return err
		}
		return nil
	})
}

func (r *Repository) UpdateClaimStatus(
	ctx context.Context,
	claimID string,
	status entities.ClaimStatus,
	updatedAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&claimModel{}).
		Where("claim_id = ?", claimID).
		Updates(map[string]any{
			"status":     string(status),
			"updated_at": updatedAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrClaimNotFound
	}
	return nil
}

func (r *Repository) ExpireActiveClaims(ctx context.Context, now time.Time) (int, error) {
	result := r.db.WithContext(ctx).
		Model(&claimModel{}).
		Where("status = ? AND expires_at < ?", string(entities.ClaimStatusActive), now.UTC()).
		Updates(map[string]any{
			"status":     string(entities.ClaimStatusExpired),
			"updated_at": now.UTC(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (r *Repository) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	var row idempotencyModel
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.IdempotencyRecord{}, false, nil
		}
		return ports.IdempotencyRecord{}, false, err
	}

	if !row.ExpiresAt.IsZero() && now.UTC().After(row.ExpiresAt.UTC()) {
		if err := r.db.WithContext(ctx).
			Where("key = ?", key).
			Delete(&idempotencyModel{}).
			Error; err != nil {
			return ports.IdempotencyRecord{}, false, err
		}
		return ports.IdempotencyRecord{}, false, nil
	}

	return row.toPort(), true, nil
}

func (r *Repository) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	row := idempotencyModelFromPort(record)
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
	if err := r.db.WithContext(ctx).
		Where("key = ?", record.Key).
		First(&existing).
		Error; err != nil {
		return err
	}
	if existing.RequestHash != record.RequestHash {
		return domainerrors.ErrIdempotencyKeyConflict
	}
	return nil
}

func (r *Repository) CountUserClipDownloadsSince(
	ctx context.Context,
	userID string,
	clipID string,
	since time.Time,
) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&clipDownloadModel{}).
		Where("user_id = ? AND clip_id = ? AND downloaded_at >= ?", userID, clipID, since.UTC()).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *Repository) CreateDownload(ctx context.Context, download ports.ClipDownload) error {
	row := clipDownloadModel{
		DownloadID:   download.DownloadID,
		ClipID:       download.ClipID,
		UserID:       download.UserID,
		IPAddress:    download.IPAddress,
		UserAgent:    download.UserAgent,
		DownloadedAt: download.DownloadedAt.UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrRepositoryInvariantBroke
		}
		return err
	}
	return nil
}

func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]ports.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}

	var rows []outboxModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", outboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}

	items := make([]ports.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toPort())
	}
	return items, nil
}

func (r *Repository) MarkOutboxSent(ctx context.Context, outboxID string, sentAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&outboxModel{}).
		Where("outbox_id = ?", outboxID).
		Updates(map[string]any{
			"status":  outboxStatusSent,
			"sent_at": sentAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrRepositoryInvariantBroke
	}
	return nil
}

func (r *Repository) ReserveEvent(
	ctx context.Context,
	eventID string,
	payloadHash string,
	expiresAt time.Time,
) (bool, error) {
	row := eventDedupModel{
		EventID:     eventID,
		PayloadHash: payloadHash,
		ExpiresAt:   expiresAt.UTC(),
		ProcessedAt: time.Now().UTC(),
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
		First(&existing).
		Error; err != nil {
		return false, err
	}
	if existing.PayloadHash != payloadHash {
		return false, domainerrors.ErrIdempotencyKeyConflict
	}
	return true, nil
}

type clipModel struct {
	ClipID          string    `gorm:"column:clip_id;primaryKey"`
	CampaignID      string    `gorm:"column:campaign_id"`
	SubmissionID    string    `gorm:"column:submission_id"`
	Title           string    `gorm:"column:title"`
	Description     string    `gorm:"column:description"`
	Niche           string    `gorm:"column:niche"`
	DurationSeconds int       `gorm:"column:duration_seconds"`
	PreviewURL      string    `gorm:"column:preview_url"`
	DownloadAssetID string    `gorm:"column:download_asset_id"`
	Exclusivity     string    `gorm:"column:exclusivity"`
	ClaimLimit      int       `gorm:"column:claim_limit"`
	Views7d         int       `gorm:"column:views_7d"`
	Votes7d         int       `gorm:"column:votes_7d"`
	EngagementRate  float64   `gorm:"column:engagement_rate"`
	Status          string    `gorm:"column:status"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (clipModel) TableName() string {
	return "clips"
}

func (m clipModel) toEntity() entities.Clip {
	return entities.Clip{
		ClipID:          m.ClipID,
		CampaignID:      m.CampaignID,
		SubmissionID:    m.SubmissionID,
		Title:           m.Title,
		Description:     m.Description,
		Niche:           m.Niche,
		DurationSeconds: m.DurationSeconds,
		PreviewURL:      m.PreviewURL,
		DownloadAssetID: m.DownloadAssetID,
		Exclusivity:     entities.ClipExclusivity(m.Exclusivity),
		ClaimLimit:      m.ClaimLimit,
		Views7d:         m.Views7d,
		Votes7d:         m.Votes7d,
		EngagementRate:  m.EngagementRate,
		Status:          entities.ClipStatus(m.Status),
		CreatedAt:       m.CreatedAt.UTC(),
		UpdatedAt:       m.UpdatedAt.UTC(),
	}
}

type claimModel struct {
	ClaimID   string    `gorm:"column:claim_id;primaryKey"`
	ClipID    string    `gorm:"column:clip_id"`
	UserID    string    `gorm:"column:user_id"`
	ClaimType string    `gorm:"column:claim_type"`
	Status    string    `gorm:"column:status"`
	RequestID string    `gorm:"column:request_id"`
	ClaimedAt time.Time `gorm:"column:claimed_at"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (claimModel) TableName() string {
	return "clip_claims"
}

func claimModelFromEntity(claim entities.Claim) claimModel {
	return claimModel{
		ClaimID:   claim.ClaimID,
		ClipID:    claim.ClipID,
		UserID:    claim.UserID,
		ClaimType: string(claim.ClaimType),
		Status:    string(claim.Status),
		RequestID: claim.RequestID,
		ClaimedAt: claim.ClaimedAt.UTC(),
		ExpiresAt: claim.ExpiresAt.UTC(),
		UpdatedAt: claim.UpdatedAt.UTC(),
	}
}

func (m claimModel) toEntity() entities.Claim {
	return entities.Claim{
		ClaimID:   m.ClaimID,
		ClipID:    m.ClipID,
		UserID:    m.UserID,
		ClaimType: entities.ClaimType(m.ClaimType),
		Status:    entities.ClaimStatus(m.Status),
		RequestID: m.RequestID,
		ClaimedAt: m.ClaimedAt.UTC(),
		ExpiresAt: m.ExpiresAt.UTC(),
		UpdatedAt: m.UpdatedAt.UTC(),
	}
}

type idempotencyModel struct {
	Key         string    `gorm:"column:key;primaryKey"`
	RequestHash string    `gorm:"column:request_hash"`
	ClaimID     string    `gorm:"column:claim_id"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
}

func (idempotencyModel) TableName() string {
	return "content_marketplace_idempotency"
}

func idempotencyModelFromPort(record ports.IdempotencyRecord) idempotencyModel {
	return idempotencyModel{
		Key:         record.Key,
		RequestHash: record.RequestHash,
		ClaimID:     record.ClaimID,
		ExpiresAt:   record.ExpiresAt.UTC(),
	}
}

func (m idempotencyModel) toPort() ports.IdempotencyRecord {
	return ports.IdempotencyRecord{
		Key:         m.Key,
		RequestHash: m.RequestHash,
		ClaimID:     m.ClaimID,
		ExpiresAt:   m.ExpiresAt.UTC(),
	}
}

type outboxModel struct {
	OutboxID     string     `gorm:"column:outbox_id;primaryKey"`
	EventType    string     `gorm:"column:event_type"`
	PartitionKey string     `gorm:"column:partition_key"`
	Payload      []byte     `gorm:"column:payload"`
	Status       string     `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	SentAt       *time.Time `gorm:"column:sent_at"`
}

type clipDownloadModel struct {
	DownloadID   string    `gorm:"column:download_id;primaryKey"`
	ClipID       string    `gorm:"column:clip_id"`
	UserID       string    `gorm:"column:user_id"`
	IPAddress    string    `gorm:"column:ip_address"`
	UserAgent    string    `gorm:"column:user_agent"`
	DownloadedAt time.Time `gorm:"column:downloaded_at"`
}

func (clipDownloadModel) TableName() string {
	return "clip_downloads"
}

func (outboxModel) TableName() string {
	return "content_marketplace_outbox"
}

func (m outboxModel) toPort() ports.OutboxMessage {
	return ports.OutboxMessage{
		OutboxID:     m.OutboxID,
		EventType:    m.EventType,
		PartitionKey: m.PartitionKey,
		Payload:      append([]byte(nil), m.Payload...),
		CreatedAt:    m.CreatedAt.UTC(),
	}
}

type eventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	PayloadHash string    `gorm:"column:payload_hash"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (eventDedupModel) TableName() string {
	return "content_marketplace_event_dedup"
}

func applyClipSort(tx *gorm.DB, popularity string) *gorm.DB {
	switch popularity {
	case "views_7d":
		return tx.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "views_7d"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "clip_id"}, Desc: false})
	case "votes_7d":
		return tx.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "votes_7d"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "clip_id"}, Desc: false})
	case "engagement_rate":
		return tx.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "engagement_rate"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "clip_id"}, Desc: false})
	default:
		return tx.
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "clip_id"}, Desc: false})
	}
}

func applyDurationBucket(tx *gorm.DB, bucket string) *gorm.DB {
	switch bucket {
	case "0-15":
		return tx.Where("duration_seconds BETWEEN ? AND ?", 0, 15)
	case "16-30":
		return tx.Where("duration_seconds BETWEEN ? AND ?", 16, 30)
	case "31-60":
		return tx.Where("duration_seconds BETWEEN ? AND ?", 31, 60)
	case "60+":
		return tx.Where("duration_seconds >= ?", 61)
	default:
		return tx
	}
}

func buildClaimedEnvelope(event ports.ClaimedEvent) (ports.EventEnvelope, error) {
	data, err := json.Marshal(map[string]string{
		"claim_id":   event.ClaimID,
		"clip_id":    event.ClipID,
		"user_id":    event.UserID,
		"claim_type": event.ClaimType,
	})
	if err != nil {
		return ports.EventEnvelope{}, err
	}
	return ports.EventEnvelope{
		EventID:          event.EventID,
		EventType:        event.EventType,
		OccurredAt:       event.OccurredAt.UTC(),
		SourceService:    "content-marketplace-service",
		SchemaVersion:    1,
		PartitionKeyPath: "clip_id",
		PartitionKey:     event.PartitionKey,
		Data:             data,
	}, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

func decodeCursor(cursor string) int {
	if strings.TrimSpace(cursor) == "" {
		return 0
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	index, err := strconv.Atoi(string(raw))
	if err != nil || index < 0 {
		return 0
	}
	return index
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}
