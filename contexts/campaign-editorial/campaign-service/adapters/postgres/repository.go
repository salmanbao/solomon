package postgresadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"

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

func (r *Repository) CreateCampaign(ctx context.Context, campaign entities.Campaign) error {
	row := campaignModelFromEntity(campaign)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrInvalidCampaignInput
		}
		return err
	}
	return nil
}

func (r *Repository) UpdateCampaign(ctx context.Context, campaign entities.Campaign) error {
	result := r.db.WithContext(ctx).
		Model(&campaignModel{}).
		Where("campaign_id = ?", strings.TrimSpace(campaign.CampaignID)).
		Updates(campaignUpdatesFromEntity(campaign))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrCampaignNotFound
	}
	return nil
}

func (r *Repository) GetCampaign(ctx context.Context, campaignID string) (entities.Campaign, error) {
	var row campaignModel
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", strings.TrimSpace(campaignID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Campaign{}, domainerrors.ErrCampaignNotFound
		}
		return entities.Campaign{}, err
	}
	return row.toEntity(), nil
}

func (r *Repository) ListCampaigns(ctx context.Context, filter ports.CampaignFilter) ([]entities.Campaign, error) {
	tx := r.db.WithContext(ctx).Model(&campaignModel{})
	if strings.TrimSpace(filter.BrandID) != "" {
		tx = tx.Where("brand_id = ?", strings.TrimSpace(filter.BrandID))
	}
	if filter.Status != "" {
		tx = tx.Where("status = ?", string(filter.Status))
	}

	var rows []campaignModel
	if err := tx.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]entities.Campaign, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) AddMedia(ctx context.Context, media entities.Media) error {
	row := mediaModelFromEntity(media)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrMediaAlreadyConfirmed
		}
		return err
	}
	return nil
}

func (r *Repository) GetMedia(ctx context.Context, mediaID string) (entities.Media, error) {
	var row mediaModel
	err := r.db.WithContext(ctx).
		Where("media_id = ?", strings.TrimSpace(mediaID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Media{}, domainerrors.ErrMediaNotFound
		}
		return entities.Media{}, err
	}
	return row.toEntity(), nil
}

func (r *Repository) UpdateMedia(ctx context.Context, media entities.Media) error {
	result := r.db.WithContext(ctx).
		Model(&mediaModel{}).
		Where("media_id = ?", strings.TrimSpace(media.MediaID)).
		Updates(map[string]any{
			"asset_path":   strings.TrimSpace(media.AssetPath),
			"content_type": strings.TrimSpace(media.ContentType),
			"status":       string(media.Status),
			"updated_at":   media.UpdatedAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrMediaNotFound
	}
	return nil
}

func (r *Repository) ListMediaByCampaign(ctx context.Context, campaignID string) ([]entities.Media, error) {
	var rows []mediaModel
	if err := r.db.WithContext(ctx).
		Where("campaign_id = ?", strings.TrimSpace(campaignID)).
		Order("created_at ASC").
		Find(&rows).
		Error; err != nil {
		return nil, err
	}

	items := make([]entities.Media, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) AppendState(ctx context.Context, item entities.StateHistory) error {
	row := stateHistoryModel{
		HistoryID:    strings.TrimSpace(item.HistoryID),
		CampaignID:   strings.TrimSpace(item.CampaignID),
		FromState:    string(item.FromState),
		ToState:      string(item.ToState),
		ChangedBy:    strings.TrimSpace(item.ChangedBy),
		ChangeReason: strings.TrimSpace(item.ChangeReason),
		CreatedAt:    item.CreatedAt.UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrInvalidCampaignInput
		}
		return err
	}
	return nil
}

func (r *Repository) AppendBudget(ctx context.Context, item entities.BudgetLog) error {
	row := budgetLogModel{
		LogID:       strings.TrimSpace(item.LogID),
		CampaignID:  strings.TrimSpace(item.CampaignID),
		AmountDelta: item.AmountDelta,
		Reason:      strings.TrimSpace(item.Reason),
		CreatedAt:   item.CreatedAt.UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrInvalidCampaignInput
		}
		return err
	}
	return nil
}

func (r *Repository) GetRecord(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	var row idempotencyModel
	err := r.db.WithContext(ctx).
		Where("key = ?", strings.TrimSpace(key)).
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
			Where("key = ?", strings.TrimSpace(key)).
			Delete(&idempotencyModel{}).
			Error; err != nil {
			return ports.IdempotencyRecord{}, false, err
		}
		return ports.IdempotencyRecord{}, false, nil
	}

	return ports.IdempotencyRecord{
		Key:             row.Key,
		RequestHash:     row.RequestHash,
		ResponsePayload: append([]byte(nil), row.ResponsePayload...),
		ExpiresAt:       row.ExpiresAt.UTC(),
	}, true, nil
}

func (r *Repository) PutRecord(ctx context.Context, record ports.IdempotencyRecord) error {
	row := idempotencyModel{
		Key:             strings.TrimSpace(record.Key),
		RequestHash:     record.RequestHash,
		ResponsePayload: append([]byte(nil), record.ResponsePayload...),
		ExpiresAt:       record.ExpiresAt.UTC(),
	}
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
		Where("key = ?", row.Key).
		First(&existing).
		Error; err != nil {
		return err
	}
	if existing.RequestHash != row.RequestHash || !bytes.Equal(existing.ResponsePayload, row.ResponsePayload) {
		return domainerrors.ErrIdempotencyKeyConflict
	}
	return nil
}

func (r *Repository) AppendOutbox(ctx context.Context, envelope ports.EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	row := outboxModel{
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

	createResult := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "outbox_id"}},
			DoNothing: true,
		}).
		Create(&row)
	if createResult.Error != nil {
		return createResult.Error
	}
	if createResult.RowsAffected > 0 {
		return nil
	}

	var existing outboxModel
	if err := r.db.WithContext(ctx).
		Select("payload").
		Where("outbox_id = ?", row.OutboxID).
		First(&existing).
		Error; err != nil {
		return err
	}
	if !bytes.Equal(existing.Payload, row.Payload) {
		return domainerrors.ErrIdempotencyKeyConflict
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
		Model(&outboxModel{}).
		Where("outbox_id = ?", strings.TrimSpace(outboxID)).
		Updates(map[string]any{
			"status":       outboxStatusPublished,
			"published_at": publishedAt.UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrInvalidCampaignInput
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
		EventID:     strings.TrimSpace(eventID),
		PayloadHash: strings.TrimSpace(payloadHash),
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
		Where("event_id = ?", row.EventID).
		First(&existing).
		Error; err != nil {
		return false, err
	}
	if existing.PayloadHash != row.PayloadHash {
		return false, domainerrors.ErrIdempotencyKeyConflict
	}
	return true, nil
}

func (r *Repository) ApplySubmissionCreated(
	ctx context.Context,
	campaignID string,
	eventID string,
	occurredAt time.Time,
) (ports.SubmissionCreatedResult, error) {
	_ = eventID
	result := ports.SubmissionCreatedResult{}
	now := occurredAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row campaignModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("campaign_id = ?", strings.TrimSpace(campaignID)).
			First(&row).
			Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domainerrors.ErrCampaignNotFound
			}
			return err
		}

		campaign := row.toEntity()
		result.CampaignID = campaign.CampaignID
		result.BudgetRemaining = campaign.BudgetRemaining
		result.NewStatus = campaign.Status
		if campaign.Status != entities.CampaignStatusActive {
			return nil
		}

		reserveAmount := campaign.RatePer1KViews
		campaign.SubmissionCount++
		campaign.BudgetReserved += reserveAmount
		campaign.BudgetRemaining = campaign.BudgetTotal - campaign.BudgetSpent - campaign.BudgetReserved
		campaign.UpdatedAt = now
		result.BudgetReservedDelta = reserveAmount
		result.BudgetRemaining = campaign.BudgetRemaining

		autoPaused := campaign.BudgetRemaining < entities.BudgetAutoPauseThreshold(campaign.RatePer1KViews)
		if autoPaused {
			campaign.Status = entities.CampaignStatusPaused
			result.AutoPaused = true
			result.NewStatus = entities.CampaignStatusPaused
		} else {
			result.NewStatus = campaign.Status
		}

		if err := tx.Model(&campaignModel{}).
			Where("campaign_id = ?", campaign.CampaignID).
			Updates(campaignUpdatesFromEntity(campaign)).
			Error; err != nil {
			return err
		}

		budgetLog := budgetLogModel{
			LogID:       uuid.NewString(),
			CampaignID:  campaign.CampaignID,
			AmountDelta: reserveAmount,
			Reason:      "submission_created_reserve",
			CreatedAt:   now,
		}
		if err := tx.Create(&budgetLog).Error; err != nil {
			return err
		}

		budgetEnvelope, err := campaignEnvelopeFromMap(
			uuid.NewString(),
			"campaign.budget_updated",
			campaign.CampaignID,
			now,
			map[string]any{
				"campaign_id":      campaign.CampaignID,
				"budget_total":     campaign.BudgetTotal,
				"budget_spent":     campaign.BudgetSpent,
				"budget_reserved":  campaign.BudgetReserved,
				"budget_remaining": campaign.BudgetRemaining,
			},
		)
		if err != nil {
			return err
		}
		if err := insertOutboxEnvelopeTx(tx, budgetEnvelope); err != nil {
			return err
		}

		if autoPaused {
			stateRow := stateHistoryModel{
				HistoryID:    uuid.NewString(),
				CampaignID:   campaign.CampaignID,
				FromState:    string(entities.CampaignStatusActive),
				ToState:      string(entities.CampaignStatusPaused),
				ChangedBy:    "system",
				ChangeReason: "budget_exhausted",
				CreatedAt:    now,
			}
			if err := tx.Create(&stateRow).Error; err != nil {
				return err
			}

			pausedEnvelope, err := campaignEnvelopeFromMap(
				uuid.NewString(),
				"campaign.paused",
				campaign.CampaignID,
				now,
				map[string]any{
					"campaign_id":      campaign.CampaignID,
					"reason":           "budget_exhausted",
					"budget_remaining": campaign.BudgetRemaining,
				},
			)
			if err != nil {
				return err
			}
			if err := insertOutboxEnvelopeTx(tx, pausedEnvelope); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ports.SubmissionCreatedResult{}, err
	}
	return result, nil
}

func (r *Repository) CompleteCampaignsPastDeadline(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]ports.DeadlineCompletionResult, error) {
	if limit <= 0 {
		limit = 100
	}
	timestamp := now.UTC()
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	results := make([]ports.DeadlineCompletionResult, 0)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []campaignModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND deadline IS NOT NULL AND deadline < ?", string(entities.CampaignStatusActive), timestamp).
			Order("deadline ASC").
			Limit(limit).
			Find(&rows).
			Error; err != nil {
			return err
		}

		for _, row := range rows {
			campaign := row.toEntity()
			campaign.Status = entities.CampaignStatusCompleted
			campaign.UpdatedAt = timestamp
			completedAt := timestamp
			campaign.CompletedAt = &completedAt

			if err := tx.Model(&campaignModel{}).
				Where("campaign_id = ?", campaign.CampaignID).
				Updates(campaignUpdatesFromEntity(campaign)).
				Error; err != nil {
				return err
			}

			stateRow := stateHistoryModel{
				HistoryID:    uuid.NewString(),
				CampaignID:   campaign.CampaignID,
				FromState:    string(entities.CampaignStatusActive),
				ToState:      string(entities.CampaignStatusCompleted),
				ChangedBy:    "system",
				ChangeReason: "deadline_reached",
				CreatedAt:    timestamp,
			}
			if err := tx.Create(&stateRow).Error; err != nil {
				return err
			}

			envelope, err := campaignEnvelopeFromMap(
				uuid.NewString(),
				"campaign.completed",
				campaign.CampaignID,
				timestamp,
				map[string]any{
					"campaign_id": campaign.CampaignID,
					"brand_id":    campaign.BrandID,
					"status":      string(campaign.Status),
					"reason":      "deadline_reached",
				},
			)
			if err != nil {
				return err
			}
			if err := insertOutboxEnvelopeTx(tx, envelope); err != nil {
				return err
			}

			results = append(results, ports.DeadlineCompletionResult{
				CampaignID: campaign.CampaignID,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func insertOutboxEnvelopeTx(tx *gorm.DB, envelope ports.EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	row := outboxModel{
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
	createResult := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "outbox_id"}},
		DoNothing: true,
	}).Create(&row)
	if createResult.Error != nil {
		return createResult.Error
	}
	if createResult.RowsAffected == 0 {
		var existing outboxModel
		if err := tx.Select("payload").Where("outbox_id = ?", row.OutboxID).First(&existing).Error; err != nil {
			return err
		}
		if !bytes.Equal(existing.Payload, row.Payload) {
			return domainerrors.ErrIdempotencyKeyConflict
		}
	}
	return nil
}

func campaignEnvelopeFromMap(
	eventID string,
	eventType string,
	campaignID string,
	occurredAt time.Time,
	data map[string]any,
) (ports.EventEnvelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return ports.EventEnvelope{}, err
	}
	return ports.EventEnvelope{
		EventID:          eventID,
		EventType:        eventType,
		OccurredAt:       occurredAt.UTC(),
		SourceService:    "campaign-service",
		SchemaVersion:    1,
		PartitionKeyPath: "campaign_id",
		PartitionKey:     campaignID,
		Data:             payload,
	}, nil
}

type campaignModel struct {
	CampaignID              string     `gorm:"column:campaign_id;primaryKey"`
	BrandID                 string     `gorm:"column:brand_id"`
	Title                   string     `gorm:"column:title"`
	Description             string     `gorm:"column:description"`
	Instructions            string     `gorm:"column:instructions"`
	Niche                   string     `gorm:"column:niche"`
	AllowedPlatforms        []string   `gorm:"column:allowed_platforms;type:text[]"`
	RequiredHashtags        []string   `gorm:"column:required_hashtags;type:text[]"`
	RequiredTags            []string   `gorm:"column:required_tags;type:text[]"`
	OptionalHashtags        []string   `gorm:"column:optional_hashtags;type:text[]"`
	UsageGuidelines         string     `gorm:"column:usage_guidelines"`
	DosAndDonts             string     `gorm:"column:dos_and_donts"`
	CampaignType            string     `gorm:"column:campaign_type"`
	DeadlineAt              *time.Time `gorm:"column:deadline"`
	TargetSubmissions       *int       `gorm:"column:target_submissions"`
	BannerImageURL          string     `gorm:"column:banner_image_url"`
	ExternalURL             string     `gorm:"column:external_url"`
	BudgetTotal             float64    `gorm:"column:budget_total"`
	BudgetSpent             float64    `gorm:"column:budget_spent"`
	BudgetReserved          float64    `gorm:"column:budget_reserved"`
	BudgetRemaining         float64    `gorm:"column:budget_remaining"`
	RatePer1KViews          float64    `gorm:"column:rate_per_1k_views"`
	SubmissionCount         int        `gorm:"column:submission_count"`
	ApprovedSubmissionCount int        `gorm:"column:approved_submission_count"`
	TotalViews              int64      `gorm:"column:total_views"`
	Status                  string     `gorm:"column:status"`
	CreatedAt               time.Time  `gorm:"column:created_at"`
	UpdatedAt               time.Time  `gorm:"column:updated_at"`
	LaunchedAt              *time.Time `gorm:"column:launched_at"`
	CompletedAt             *time.Time `gorm:"column:completed_at"`
}

func (campaignModel) TableName() string {
	return "campaigns"
}

func campaignModelFromEntity(item entities.Campaign) campaignModel {
	return campaignModel{
		CampaignID:              strings.TrimSpace(item.CampaignID),
		BrandID:                 strings.TrimSpace(item.BrandID),
		Title:                   strings.TrimSpace(item.Title),
		Description:             strings.TrimSpace(item.Description),
		Instructions:            strings.TrimSpace(item.Instructions),
		Niche:                   strings.TrimSpace(item.Niche),
		AllowedPlatforms:        copyOrEmpty(item.AllowedPlatforms),
		RequiredHashtags:        copyOrEmpty(item.RequiredHashtags),
		RequiredTags:            copyOrEmpty(item.RequiredTags),
		OptionalHashtags:        copyOrEmpty(item.OptionalHashtags),
		UsageGuidelines:         strings.TrimSpace(item.UsageGuidelines),
		DosAndDonts:             strings.TrimSpace(item.DosAndDonts),
		CampaignType:            string(item.CampaignType),
		DeadlineAt:              normalizeOptionalTime(item.DeadlineAt),
		TargetSubmissions:       item.TargetSubmissions,
		BannerImageURL:          strings.TrimSpace(item.BannerImageURL),
		ExternalURL:             strings.TrimSpace(item.ExternalURL),
		BudgetTotal:             item.BudgetTotal,
		BudgetSpent:             item.BudgetSpent,
		BudgetReserved:          item.BudgetReserved,
		BudgetRemaining:         item.BudgetRemaining,
		RatePer1KViews:          item.RatePer1KViews,
		SubmissionCount:         item.SubmissionCount,
		ApprovedSubmissionCount: item.ApprovedSubmissionCount,
		TotalViews:              item.TotalViews,
		Status:                  string(item.Status),
		CreatedAt:               item.CreatedAt.UTC(),
		UpdatedAt:               item.UpdatedAt.UTC(),
		LaunchedAt:              normalizeOptionalTime(item.LaunchedAt),
		CompletedAt:             normalizeOptionalTime(item.CompletedAt),
	}
}

func campaignUpdatesFromEntity(item entities.Campaign) map[string]any {
	row := campaignModelFromEntity(item)
	return map[string]any{
		"brand_id":                  row.BrandID,
		"title":                     row.Title,
		"description":               row.Description,
		"instructions":              row.Instructions,
		"niche":                     row.Niche,
		"allowed_platforms":         row.AllowedPlatforms,
		"required_hashtags":         row.RequiredHashtags,
		"required_tags":             row.RequiredTags,
		"optional_hashtags":         row.OptionalHashtags,
		"usage_guidelines":          row.UsageGuidelines,
		"dos_and_donts":             row.DosAndDonts,
		"campaign_type":             row.CampaignType,
		"deadline":                  row.DeadlineAt,
		"target_submissions":        row.TargetSubmissions,
		"banner_image_url":          row.BannerImageURL,
		"external_url":              row.ExternalURL,
		"budget_total":              row.BudgetTotal,
		"budget_spent":              row.BudgetSpent,
		"budget_reserved":           row.BudgetReserved,
		"budget_remaining":          row.BudgetRemaining,
		"rate_per_1k_views":         row.RatePer1KViews,
		"submission_count":          row.SubmissionCount,
		"approved_submission_count": row.ApprovedSubmissionCount,
		"total_views":               row.TotalViews,
		"status":                    row.Status,
		"updated_at":                row.UpdatedAt,
		"launched_at":               row.LaunchedAt,
		"completed_at":              row.CompletedAt,
	}
}

func (m campaignModel) toEntity() entities.Campaign {
	return entities.Campaign{
		CampaignID:              m.CampaignID,
		BrandID:                 m.BrandID,
		Title:                   m.Title,
		Description:             m.Description,
		Instructions:            m.Instructions,
		Niche:                   m.Niche,
		AllowedPlatforms:        copyOrEmpty(m.AllowedPlatforms),
		RequiredHashtags:        copyOrEmpty(m.RequiredHashtags),
		RequiredTags:            copyOrEmpty(m.RequiredTags),
		OptionalHashtags:        copyOrEmpty(m.OptionalHashtags),
		UsageGuidelines:         m.UsageGuidelines,
		DosAndDonts:             m.DosAndDonts,
		CampaignType:            entities.CampaignType(m.CampaignType),
		DeadlineAt:              normalizeOptionalTime(m.DeadlineAt),
		TargetSubmissions:       m.TargetSubmissions,
		BannerImageURL:          m.BannerImageURL,
		ExternalURL:             m.ExternalURL,
		BudgetTotal:             m.BudgetTotal,
		BudgetSpent:             m.BudgetSpent,
		BudgetReserved:          m.BudgetReserved,
		BudgetRemaining:         m.BudgetRemaining,
		RatePer1KViews:          m.RatePer1KViews,
		SubmissionCount:         m.SubmissionCount,
		ApprovedSubmissionCount: m.ApprovedSubmissionCount,
		TotalViews:              m.TotalViews,
		Status:                  entities.CampaignStatus(m.Status),
		CreatedAt:               m.CreatedAt.UTC(),
		UpdatedAt:               m.UpdatedAt.UTC(),
		LaunchedAt:              normalizeOptionalTime(m.LaunchedAt),
		CompletedAt:             normalizeOptionalTime(m.CompletedAt),
	}
}

type mediaModel struct {
	MediaID     string    `gorm:"column:media_id;primaryKey"`
	CampaignID  string    `gorm:"column:campaign_id"`
	AssetPath   string    `gorm:"column:asset_path"`
	ContentType string    `gorm:"column:content_type"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (mediaModel) TableName() string {
	return "campaign_media"
}

func mediaModelFromEntity(item entities.Media) mediaModel {
	return mediaModel{
		MediaID:     strings.TrimSpace(item.MediaID),
		CampaignID:  strings.TrimSpace(item.CampaignID),
		AssetPath:   strings.TrimSpace(item.AssetPath),
		ContentType: strings.TrimSpace(item.ContentType),
		Status:      string(item.Status),
		CreatedAt:   item.CreatedAt.UTC(),
		UpdatedAt:   item.UpdatedAt.UTC(),
	}
}

func (m mediaModel) toEntity() entities.Media {
	return entities.Media{
		MediaID:     m.MediaID,
		CampaignID:  m.CampaignID,
		AssetPath:   m.AssetPath,
		ContentType: m.ContentType,
		Status:      entities.MediaStatus(m.Status),
		CreatedAt:   m.CreatedAt.UTC(),
		UpdatedAt:   m.UpdatedAt.UTC(),
	}
}

type stateHistoryModel struct {
	HistoryID    string    `gorm:"column:history_id;primaryKey"`
	CampaignID   string    `gorm:"column:campaign_id"`
	FromState    string    `gorm:"column:from_state"`
	ToState      string    `gorm:"column:to_state"`
	ChangedBy    string    `gorm:"column:changed_by"`
	ChangeReason string    `gorm:"column:change_reason"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (stateHistoryModel) TableName() string {
	return "campaign_state_history"
}

type budgetLogModel struct {
	LogID       string    `gorm:"column:log_id;primaryKey"`
	CampaignID  string    `gorm:"column:campaign_id"`
	AmountDelta float64   `gorm:"column:amount_delta"`
	Reason      string    `gorm:"column:reason"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (budgetLogModel) TableName() string {
	return "campaign_budget_log"
}

type idempotencyModel struct {
	Key             string    `gorm:"column:key;primaryKey"`
	RequestHash     string    `gorm:"column:request_hash"`
	ResponsePayload []byte    `gorm:"column:response_payload"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
}

func (idempotencyModel) TableName() string {
	return "campaign_idempotency"
}

type outboxModel struct {
	OutboxID     string     `gorm:"column:outbox_id;primaryKey"`
	EventType    string     `gorm:"column:event_type"`
	PartitionKey string     `gorm:"column:partition_key"`
	Payload      []byte     `gorm:"column:payload"`
	Status       string     `gorm:"column:status"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
}

func (outboxModel) TableName() string {
	return "campaign_outbox"
}

type eventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	PayloadHash string    `gorm:"column:payload_hash"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (eventDedupModel) TableName() string {
	return "campaign_event_dedup"
}

func copyOrEmpty(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	return append([]string(nil), items...)
}

func normalizeOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	timestamp := value.UTC()
	return &timestamp
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
