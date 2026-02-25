package postgresadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	"solomon/contexts/campaign-editorial/distribution-service/ports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxStatusPending   = "pending"
	outboxStatusPublished = "published"
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

func (r *Repository) CreateItem(ctx context.Context, item entities.DistributionItem) error {
	if strings.TrimSpace(item.ID) == "" ||
		strings.TrimSpace(item.InfluencerID) == "" ||
		strings.TrimSpace(item.ClipID) == "" ||
		strings.TrimSpace(item.CampaignID) == "" {
		r.logWarn("distribution_repo_create_item_invalid_input",
			"item_id", strings.TrimSpace(item.ID),
			"influencer_id", strings.TrimSpace(item.InfluencerID),
			"clip_id", strings.TrimSpace(item.ClipID),
			"campaign_id", strings.TrimSpace(item.CampaignID),
		)
		return domainerrors.ErrInvalidDistributionInput
	}

	var duplicates int64
	if err := r.db.WithContext(ctx).
		Model(&distributionItemModel{}).
		Where("influencer_id = ?", strings.TrimSpace(item.InfluencerID)).
		Where("clip_id = ?", strings.TrimSpace(item.ClipID)).
		Where("campaign_id = ?", strings.TrimSpace(item.CampaignID)).
		Count(&duplicates).
		Error; err != nil {
		return r.logError("distribution_repo_create_item_duplicate_check_failed", err,
			"item_id", strings.TrimSpace(item.ID),
			"influencer_id", strings.TrimSpace(item.InfluencerID),
			"clip_id", strings.TrimSpace(item.ClipID),
			"campaign_id", strings.TrimSpace(item.CampaignID),
		)
	}
	if duplicates > 0 {
		r.logWarn("distribution_repo_create_item_duplicate_detected",
			"item_id", strings.TrimSpace(item.ID),
			"influencer_id", strings.TrimSpace(item.InfluencerID),
			"clip_id", strings.TrimSpace(item.ClipID),
			"campaign_id", strings.TrimSpace(item.CampaignID),
		)
		return domainerrors.ErrDistributionItemExists
	}

	row := distributionItemModelFromEntity(item)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			r.logWarn("distribution_repo_create_item_unique_conflict",
				"item_id", strings.TrimSpace(item.ID),
				"influencer_id", strings.TrimSpace(item.InfluencerID),
				"clip_id", strings.TrimSpace(item.ClipID),
				"campaign_id", strings.TrimSpace(item.CampaignID),
			)
			return domainerrors.ErrDistributionItemExists
		}
		return r.logError("distribution_repo_create_item_failed", err,
			"item_id", strings.TrimSpace(item.ID),
			"influencer_id", strings.TrimSpace(item.InfluencerID),
			"clip_id", strings.TrimSpace(item.ClipID),
			"campaign_id", strings.TrimSpace(item.CampaignID),
		)
	}
	return nil
}

func (r *Repository) UpdateItem(ctx context.Context, item entities.DistributionItem) error {
	result := r.db.WithContext(ctx).
		Model(&distributionItemModel{}).
		Where("id = ?", strings.TrimSpace(item.ID)).
		Updates(distributionItemUpdatesFromEntity(item))
	if result.Error != nil {
		return r.logError("distribution_repo_update_item_failed", result.Error,
			"item_id", strings.TrimSpace(item.ID),
		)
	}
	if result.RowsAffected == 0 {
		r.logWarn("distribution_repo_update_item_not_found",
			"item_id", strings.TrimSpace(item.ID),
		)
		return domainerrors.ErrDistributionItemNotFound
	}
	return nil
}

func (r *Repository) GetItem(ctx context.Context, itemID string) (entities.DistributionItem, error) {
	var row distributionItemModel
	err := r.db.WithContext(ctx).
		Where("id = ?", strings.TrimSpace(itemID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.DistributionItem{}, domainerrors.ErrDistributionItemNotFound
		}
		return entities.DistributionItem{}, r.logError("distribution_repo_get_item_failed", err,
			"item_id", strings.TrimSpace(itemID),
		)
	}
	return row.toEntity(), nil
}

func (r *Repository) ListItemsByInfluencer(ctx context.Context, influencerID string) ([]entities.DistributionItem, error) {
	var rows []distributionItemModel
	if err := r.db.WithContext(ctx).
		Where("influencer_id = ?", strings.TrimSpace(influencerID)).
		Order("claimed_at DESC").
		Find(&rows).Error; err != nil {
		return nil, r.logError("distribution_repo_list_items_by_influencer_failed", err,
			"influencer_id", strings.TrimSpace(influencerID),
		)
	}
	items := make([]entities.DistributionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) ListDueScheduled(
	ctx context.Context,
	threshold time.Time,
	limit int,
) ([]entities.DistributionItem, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []distributionItemModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(entities.DistributionStatusScheduled)).
		Where("scheduled_for_utc IS NOT NULL").
		Where("scheduled_for_utc <= ?", threshold.UTC()).
		Order("scheduled_for_utc ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, r.logError("distribution_repo_list_due_scheduled_failed", err,
			"threshold_utc", threshold.UTC().Format(time.RFC3339),
			"limit", limit,
		)
	}
	items := make([]entities.DistributionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) GetCampaignIDByClip(ctx context.Context, clipID string) (string, error) {
	var row clipProjectionModel
	// Canonical DBR allowance for M31 -> M09 is internal_sql_readonly.
	// This method intentionally performs a read-only projection lookup only.
	err := r.db.WithContext(ctx).
		Select("campaign_id").
		Where("clip_id = ?", strings.TrimSpace(clipID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domainerrors.ErrDistributionItemNotFound
		}
		return "", r.logError("distribution_repo_get_campaign_by_clip_failed", err,
			"clip_id", strings.TrimSpace(clipID),
		)
	}
	return strings.TrimSpace(row.CampaignID), nil
}

func (r *Repository) AddOverlay(ctx context.Context, overlay entities.Overlay) error {
	row := distributionOverlayModel{
		ID:                 strings.TrimSpace(overlay.ID),
		DistributionItemID: strings.TrimSpace(overlay.DistributionItemID),
		OverlayType:        strings.TrimSpace(overlay.OverlayType),
		AssetPath:          strings.TrimSpace(overlay.AssetPath),
		DurationSeconds:    overlay.DurationSeconds,
		CreatedAt:          overlay.CreatedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return r.logError("distribution_repo_add_overlay_failed", err,
			"item_id", row.DistributionItemID,
			"overlay_id", row.ID,
			"overlay_type", row.OverlayType,
		)
	}
	return nil
}

func (r *Repository) UpsertCaption(ctx context.Context, caption entities.Caption) error {
	row := distributionCaptionModel{
		ID:                 strings.TrimSpace(caption.ID),
		DistributionItemID: strings.TrimSpace(caption.DistributionItemID),
		Platform:           strings.TrimSpace(caption.Platform),
		CaptionText:        strings.TrimSpace(caption.CaptionText),
		Hashtags:           append([]string(nil), caption.Hashtags...),
		CreatedAt:          caption.CreatedAt.UTC(),
		UpdatedAt:          caption.UpdatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "distribution_item_id"}, {Name: "platform"}},
		DoUpdates: clause.AssignmentColumns([]string{"caption_text", "hashtags", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return r.logError("distribution_repo_upsert_caption_failed", err,
			"item_id", row.DistributionItemID,
			"caption_id", row.ID,
			"platform", row.Platform,
		)
	}
	return nil
}

func (r *Repository) UpsertPlatformStatus(ctx context.Context, status entities.PlatformStatus) error {
	row := distributionPlatformStatusModel{
		ID:                 strings.TrimSpace(status.ID),
		DistributionItemID: strings.TrimSpace(status.DistributionItemID),
		Platform:           strings.TrimSpace(status.Platform),
		Status:             strings.TrimSpace(status.Status),
		PlatformPostURL:    strings.TrimSpace(status.PlatformPostURL),
		ErrorMessage:       strings.TrimSpace(status.ErrorMessage),
		RetryCount:         status.RetryCount,
		UpdatedAt:          status.UpdatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "distribution_item_id"}, {Name: "platform"}},
		DoUpdates: clause.Assignments(map[string]any{
			"status":            row.Status,
			"platform_post_url": row.PlatformPostURL,
			"error_message":     row.ErrorMessage,
			"retry_count":       row.RetryCount,
			"updated_at":        row.UpdatedAt,
		}),
	}).Create(&row).Error; err != nil {
		return r.logError("distribution_repo_upsert_platform_status_failed", err,
			"item_id", row.DistributionItemID,
			"platform", row.Platform,
			"status", row.Status,
		)
	}
	return nil
}

func (r *Repository) AddPublishingAnalytics(ctx context.Context, analytics entities.PublishingAnalytics) error {
	row := publishingAnalyticsModel{
		ID:                   strings.TrimSpace(analytics.ID),
		DistributionItemID:   strings.TrimSpace(analytics.DistributionItemID),
		InfluencerID:         strings.TrimSpace(analytics.InfluencerID),
		CampaignID:           strings.TrimSpace(analytics.CampaignID),
		Platform:             strings.TrimSpace(analytics.Platform),
		Success:              analytics.Success,
		ErrorCode:            strings.TrimSpace(analytics.ErrorCode),
		ErrorMessage:         strings.TrimSpace(analytics.ErrorMessage),
		TimeToPublishSeconds: analytics.TimeToPublishSeconds,
		CreatedAt:            analytics.CreatedAt.UTC(),
	}
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return r.logError("distribution_repo_add_publishing_analytics_failed", err,
			"item_id", row.DistributionItemID,
			"analytics_id", row.ID,
			"platform", row.Platform,
		)
	}
	return nil
}

func (r *Repository) AppendOutbox(ctx context.Context, envelope ports.EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return r.logError("distribution_repo_append_outbox_marshal_failed", err,
			"event_id", strings.TrimSpace(envelope.EventID),
			"event_type", strings.TrimSpace(envelope.EventType),
		)
	}
	row := distributionOutboxModel{
		OutboxID:     strings.TrimSpace(envelope.EventID),
		EventType:    strings.TrimSpace(envelope.EventType),
		PartitionKey: strings.TrimSpace(envelope.PartitionKey),
		Payload:      payload,
		Status:       outboxStatusPending,
		CreatedAt:    envelope.OccurredAt.UTC(),
	}
	if row.OutboxID == "" {
		row.OutboxID = uuid.NewString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}

	createResult := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "outbox_id"}},
		DoNothing: true,
	}).Create(&row)
	if createResult.Error != nil {
		return r.logError("distribution_repo_append_outbox_insert_failed", createResult.Error,
			"outbox_id", row.OutboxID,
			"event_type", row.EventType,
		)
	}
	if createResult.RowsAffected > 0 {
		return nil
	}

	var existing distributionOutboxModel
	if err := r.db.WithContext(ctx).
		Select("payload").
		Where("outbox_id = ?", row.OutboxID).
		First(&existing).
		Error; err != nil {
		return r.logError("distribution_repo_append_outbox_load_existing_failed", err,
			"outbox_id", row.OutboxID,
		)
	}
	if !bytes.Equal(existing.Payload, row.Payload) {
		r.logWarn("distribution_repo_append_outbox_payload_conflict",
			"outbox_id", row.OutboxID,
			"event_type", row.EventType,
		)
		return domainerrors.ErrInvalidDistributionInput
	}
	return nil
}

func (r *Repository) ListPendingOutbox(ctx context.Context, limit int) ([]ports.OutboxMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []distributionOutboxModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", outboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, r.logError("distribution_repo_list_pending_outbox_failed", err,
			"limit", limit,
		)
	}
	items := make([]ports.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		items = append(items, ports.OutboxMessage{
			OutboxID:     row.OutboxID,
			EventType:    row.EventType,
			PartitionKey: row.PartitionKey,
			Payload:      append([]byte(nil), row.Payload...),
			CreatedAt:    row.CreatedAt.UTC(),
		})
	}
	return items, nil
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&distributionOutboxModel{}).
		Where("outbox_id = ?", strings.TrimSpace(outboxID)).
		Updates(map[string]any{
			"status":       outboxStatusPublished,
			"published_at": publishedAt.UTC(),
		})
	if result.Error != nil {
		return r.logError("distribution_repo_mark_outbox_published_failed", result.Error,
			"outbox_id", strings.TrimSpace(outboxID),
		)
	}
	if result.RowsAffected == 0 {
		r.logWarn("distribution_repo_mark_outbox_published_not_found",
			"outbox_id", strings.TrimSpace(outboxID),
		)
		return domainerrors.ErrInvalidDistributionInput
	}
	return nil
}

func (r *Repository) logError(event string, err error, attrs ...any) error {
	fields := make([]any, 0, len(attrs)+7)
	fields = append(fields,
		"event", event,
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"error", err.Error(),
	)
	fields = append(fields, attrs...)
	r.logger.Error("distribution repository operation failed", fields...)
	return err
}

func (r *Repository) logWarn(event string, attrs ...any) {
	fields := make([]any, 0, len(attrs)+5)
	fields = append(fields,
		"event", event,
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
	)
	fields = append(fields, attrs...)
	r.logger.Warn("distribution repository warning", fields...)
}

type distributionItemModel struct {
	ID             string     `gorm:"column:id;primaryKey"`
	InfluencerID   string     `gorm:"column:influencer_id"`
	ClipID         string     `gorm:"column:clip_id"`
	CampaignID     string     `gorm:"column:campaign_id"`
	Status         string     `gorm:"column:status"`
	ClaimedAt      time.Time  `gorm:"column:claimed_at"`
	ClaimExpiresAt time.Time  `gorm:"column:claim_expires_at"`
	ScheduledFor   *time.Time `gorm:"column:scheduled_for_utc"`
	Timezone       string     `gorm:"column:timezone"`
	Platforms      []string   `gorm:"column:platforms;type:text[]"`
	CaptionText    string     `gorm:"column:caption_text"`
	RetryCount     int        `gorm:"column:retry_count"`
	LastError      string     `gorm:"column:last_error"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (distributionItemModel) TableName() string {
	return "distribution_items"
}

func distributionItemModelFromEntity(item entities.DistributionItem) distributionItemModel {
	return distributionItemModel{
		ID:             strings.TrimSpace(item.ID),
		InfluencerID:   strings.TrimSpace(item.InfluencerID),
		ClipID:         strings.TrimSpace(item.ClipID),
		CampaignID:     strings.TrimSpace(item.CampaignID),
		Status:         string(item.Status),
		ClaimedAt:      item.ClaimedAt.UTC(),
		ClaimExpiresAt: item.ClaimExpiresAt.UTC(),
		ScheduledFor:   normalizeOptionalTime(item.ScheduledForUTC),
		Timezone:       strings.TrimSpace(item.Timezone),
		Platforms:      append([]string(nil), item.Platforms...),
		CaptionText:    strings.TrimSpace(item.Caption),
		RetryCount:     item.RetryCount,
		LastError:      strings.TrimSpace(item.LastError),
		PublishedAt:    normalizeOptionalTime(item.PublishedAt),
		UpdatedAt:      item.UpdatedAt.UTC(),
	}
}

func distributionItemUpdatesFromEntity(item entities.DistributionItem) map[string]any {
	row := distributionItemModelFromEntity(item)
	return map[string]any{
		"influencer_id":     row.InfluencerID,
		"clip_id":           row.ClipID,
		"campaign_id":       row.CampaignID,
		"status":            row.Status,
		"claimed_at":        row.ClaimedAt,
		"claim_expires_at":  row.ClaimExpiresAt,
		"scheduled_for_utc": row.ScheduledFor,
		"timezone":          row.Timezone,
		"platforms":         row.Platforms,
		"caption_text":      row.CaptionText,
		"retry_count":       row.RetryCount,
		"last_error":        row.LastError,
		"published_at":      row.PublishedAt,
		"updated_at":        row.UpdatedAt,
	}
}

func (m distributionItemModel) toEntity() entities.DistributionItem {
	return entities.DistributionItem{
		ID:              m.ID,
		InfluencerID:    m.InfluencerID,
		ClipID:          m.ClipID,
		CampaignID:      m.CampaignID,
		Status:          entities.DistributionStatus(m.Status),
		ClaimedAt:       m.ClaimedAt.UTC(),
		ClaimExpiresAt:  m.ClaimExpiresAt.UTC(),
		ScheduledForUTC: normalizeOptionalTime(m.ScheduledFor),
		Timezone:        m.Timezone,
		Platforms:       append([]string(nil), m.Platforms...),
		Caption:         m.CaptionText,
		RetryCount:      m.RetryCount,
		LastError:       m.LastError,
		PublishedAt:     normalizeOptionalTime(m.PublishedAt),
		UpdatedAt:       m.UpdatedAt.UTC(),
	}
}

type distributionCaptionModel struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	DistributionItemID string    `gorm:"column:distribution_item_id"`
	Platform           string    `gorm:"column:platform"`
	CaptionText        string    `gorm:"column:caption_text"`
	Hashtags           []string  `gorm:"column:hashtags;type:text[]"`
	CreatedAt          time.Time `gorm:"column:created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (distributionCaptionModel) TableName() string {
	return "distribution_captions"
}

type distributionOverlayModel struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	DistributionItemID string    `gorm:"column:distribution_item_id"`
	OverlayType        string    `gorm:"column:overlay_type"`
	AssetPath          string    `gorm:"column:asset_path"`
	DurationSeconds    float64   `gorm:"column:duration_seconds"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (distributionOverlayModel) TableName() string {
	return "distribution_overlays"
}

type distributionPlatformStatusModel struct {
	ID                 string    `gorm:"column:id;primaryKey"`
	DistributionItemID string    `gorm:"column:distribution_item_id"`
	Platform           string    `gorm:"column:platform"`
	Status             string    `gorm:"column:status"`
	PlatformPostURL    string    `gorm:"column:platform_post_url"`
	ErrorMessage       string    `gorm:"column:error_message"`
	RetryCount         int       `gorm:"column:retry_count"`
	UpdatedAt          time.Time `gorm:"column:updated_at"`
}

func (distributionPlatformStatusModel) TableName() string {
	return "distribution_platform_status"
}

type publishingAnalyticsModel struct {
	ID                   string    `gorm:"column:id;primaryKey"`
	DistributionItemID   string    `gorm:"column:distribution_item_id"`
	InfluencerID         string    `gorm:"column:influencer_id"`
	CampaignID           string    `gorm:"column:campaign_id"`
	Platform             string    `gorm:"column:platform"`
	Success              bool      `gorm:"column:success"`
	ErrorCode            string    `gorm:"column:error_code"`
	ErrorMessage         string    `gorm:"column:error_message"`
	TimeToPublishSeconds *int      `gorm:"column:time_to_publish_seconds"`
	CreatedAt            time.Time `gorm:"column:created_at"`
}

func (publishingAnalyticsModel) TableName() string {
	return "publishing_analytics"
}

type distributionOutboxModel struct {
	OutboxID     string     `gorm:"column:outbox_id;primaryKey"`
	EventType    string     `gorm:"column:event_type"`
	PartitionKey string     `gorm:"column:partition_key"`
	Payload      []byte     `gorm:"column:payload"`
	Status       string     `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
}

func (distributionOutboxModel) TableName() string {
	return "distribution_outbox"
}

type clipProjectionModel struct {
	ClipID     string `gorm:"column:clip_id;primaryKey"`
	CampaignID string `gorm:"column:campaign_id"`
}

func (clipProjectionModel) TableName() string {
	return "clips"
}

func normalizeOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	t := value.UTC()
	return &t
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

var _ ports.Repository = (*Repository)(nil)
var _ ports.OutboxWriter = (*Repository)(nil)
var _ ports.OutboxRepository = (*Repository)(nil)
