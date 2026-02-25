package postgresadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"

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

func (r *Repository) CreateSubmission(ctx context.Context, submission entities.Submission) error {
	createdAt := submission.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	var duplicateCount int64
	if err := r.db.WithContext(ctx).
		Model(&submissionModel{}).
		Where("campaign_id = ?", strings.TrimSpace(submission.CampaignID)).
		Where("creator_id = ?", strings.TrimSpace(submission.CreatorID)).
		Where("post_url = ?", strings.TrimSpace(submission.PostURL)).
		Where("status <> ?", string(entities.SubmissionStatusCancelled)).
		Where("created_at >= ?", createdAt.Add(-24*time.Hour)).
		Count(&duplicateCount).
		Error; err != nil {
		return err
	}
	if duplicateCount > 0 {
		return domainerrors.ErrDuplicateSubmission
	}

	row := submissionModelFromEntity(submission)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if isUniqueViolation(err) {
			return domainerrors.ErrDuplicateSubmission
		}
		return err
	}
	return nil
}

func (r *Repository) UpdateSubmission(ctx context.Context, submission entities.Submission) error {
	result := r.db.WithContext(ctx).
		Model(&submissionModel{}).
		Where("submission_id = ?", strings.TrimSpace(submission.SubmissionID)).
		Updates(submissionUpdatesFromEntity(submission))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrSubmissionNotFound
	}
	return nil
}

func (r *Repository) GetSubmission(ctx context.Context, submissionID string) (entities.Submission, error) {
	var row submissionModel
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Submission{}, domainerrors.ErrSubmissionNotFound
		}
		return entities.Submission{}, err
	}
	return row.toEntity(), nil
}

func (r *Repository) ListSubmissions(ctx context.Context, filter ports.SubmissionFilter) ([]entities.Submission, error) {
	tx := r.db.WithContext(ctx).Model(&submissionModel{})
	if strings.TrimSpace(filter.CreatorID) != "" {
		tx = tx.Where("creator_id = ?", strings.TrimSpace(filter.CreatorID))
	}
	if strings.TrimSpace(filter.CampaignID) != "" {
		tx = tx.Where("campaign_id = ?", strings.TrimSpace(filter.CampaignID))
	}
	if filter.Status != "" {
		tx = tx.Where("status = ?", string(filter.Status))
	}

	var rows []submissionModel
	if err := tx.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]entities.Submission, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) AddReport(ctx context.Context, report entities.SubmissionReport) error {
	reportedBy := strings.TrimSpace(report.ReportedByID)
	if reportedBy != "" {
		var existingCount int64
		if err := r.db.WithContext(ctx).
			Model(&submissionReportModel{}).
			Where("submission_id = ?", strings.TrimSpace(report.SubmissionID)).
			Where("reported_by_user_id = ?", reportedBy).
			Count(&existingCount).
			Error; err != nil {
			return err
		}
		if existingCount > 0 {
			return domainerrors.ErrAlreadyReported
		}
	}
	row := submissionReportModel{
		ReportID:         strings.TrimSpace(report.ReportID),
		SubmissionID:     strings.TrimSpace(report.SubmissionID),
		ReportedByUserID: reportedBy,
		Reason:           strings.TrimSpace(report.Reason),
		Description:      strings.TrimSpace(report.Description),
		CreatedAt:        report.ReportedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) AddFlag(ctx context.Context, flag entities.SubmissionFlag) error {
	details, err := json.Marshal(flag.Details)
	if err != nil {
		return err
	}
	row := submissionFlagModel{
		FlagID:       strings.TrimSpace(flag.FlagID),
		SubmissionID: strings.TrimSpace(flag.SubmissionID),
		FlagType:     strings.TrimSpace(flag.FlagType),
		Severity:     strings.TrimSpace(flag.Severity),
		Details:      details,
		IsResolved:   flag.IsResolved,
		ResolvedAt:   normalizeOptionalTime(flag.ResolvedAt),
		CreatedAt:    flag.CreatedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) AddAudit(ctx context.Context, audit entities.SubmissionAudit) error {
	row := submissionAuditModel{
		AuditID:      strings.TrimSpace(audit.AuditID),
		SubmissionID: strings.TrimSpace(audit.SubmissionID),
		Action:       strings.TrimSpace(audit.Action),
		OldStatus:    string(audit.OldStatus),
		NewStatus:    string(audit.NewStatus),
		ActorID:      strings.TrimSpace(audit.ActorID),
		ActorRole:    strings.TrimSpace(audit.ActorRole),
		ReasonCode:   strings.TrimSpace(audit.ReasonCode),
		ReasonNotes:  strings.TrimSpace(audit.ReasonNotes),
		IPAddress:    strings.TrimSpace(audit.IPAddress),
		UserAgent:    strings.TrimSpace(audit.UserAgent),
		CreatedAt:    audit.CreatedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) AddBulkOperation(ctx context.Context, operation entities.BulkSubmissionOperation) error {
	row := bulkSubmissionOperationModel{
		OperationID:       strings.TrimSpace(operation.OperationID),
		CampaignID:        strings.TrimSpace(operation.CampaignID),
		OperationType:     strings.TrimSpace(operation.OperationType),
		SubmissionIDs:     append([]string(nil), operation.SubmissionIDs...),
		PerformedByUserID: strings.TrimSpace(operation.PerformedByUserID),
		SucceededCount:    operation.SucceededCount,
		FailedCount:       operation.FailedCount,
		ReasonCode:        strings.TrimSpace(operation.ReasonCode),
		ReasonNotes:       strings.TrimSpace(operation.ReasonNotes),
		CreatedAt:         operation.CreatedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) AddViewSnapshot(ctx context.Context, snapshot entities.ViewSnapshot) error {
	metrics, err := json.Marshal(snapshot.PlatformMetricsJSON)
	if err != nil {
		return err
	}
	row := viewSnapshotModel{
		SnapshotID:         strings.TrimSpace(snapshot.SnapshotID),
		SubmissionID:       strings.TrimSpace(snapshot.SubmissionID),
		ViewsCount:         snapshot.ViewsCount,
		EngagementEstimate: snapshot.EngagementEstimate,
		PlatformMetrics:    metrics,
		SyncedAt:           snapshot.SyncedAt.UTC(),
		IsAnomaly:          snapshot.IsAnomaly,
		AnomalyReason:      strings.TrimSpace(snapshot.AnomalyReason),
	}
	if row.SyncedAt.IsZero() {
		row.SyncedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Create(&row).Error
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
		return domainerrors.ErrInvalidSubmissionInput
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

func (r *Repository) ListPendingAutoApprove(
	ctx context.Context,
	threshold time.Time,
	limit int,
) ([]entities.Submission, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []submissionModel
	if err := r.db.WithContext(ctx).
		Where("status = ?", string(entities.SubmissionStatusPending)).
		Where("reported_count = 0").
		Where("created_at <= ?", threshold.UTC()).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	items := make([]entities.Submission, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) ListDueViewLock(
	ctx context.Context,
	threshold time.Time,
	limit int,
) ([]entities.Submission, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows []submissionModel
	if err := r.db.WithContext(ctx).
		Where("status IN ?", []string{
			string(entities.SubmissionStatusApproved),
			string(entities.SubmissionStatusVerification),
		}).
		Where("verification_window_end IS NOT NULL").
		Where("verification_window_end <= ?", threshold.UTC()).
		Where("locked_views IS NULL").
		Order("verification_window_end ASC").
		Limit(limit).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}
	items := make([]entities.Submission, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) GetCampaignForSubmission(ctx context.Context, campaignID string) (ports.CampaignForSubmission, error) {
	var row campaignProjectionModel
	if err := r.db.WithContext(ctx).
		Where("campaign_id = ?", strings.TrimSpace(campaignID)).
		First(&row).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.CampaignForSubmission{}, domainerrors.ErrCampaignNotFound
		}
		return ports.CampaignForSubmission{}, err
	}
	return ports.CampaignForSubmission{
		CampaignID:       row.CampaignID,
		Status:           row.Status,
		AllowedPlatforms: append([]string(nil), row.AllowedPlatforms...),
		RatePer1KViews:   row.RatePer1KViews,
	}, nil
}

type submissionModel struct {
	SubmissionID          string     `gorm:"column:submission_id;primaryKey"`
	CampaignID            string     `gorm:"column:campaign_id"`
	CreatorID             string     `gorm:"column:creator_id"`
	Platform              string     `gorm:"column:platform"`
	PostURL               string     `gorm:"column:post_url"`
	PostID                string     `gorm:"column:post_id"`
	CreatorPlatformHandle string     `gorm:"column:creator_platform_handle"`
	Status                string     `gorm:"column:status"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	ApprovedAt            *time.Time `gorm:"column:approved_at"`
	ApprovedByUserID      string     `gorm:"column:approved_by_user_id"`
	ApprovalReason        string     `gorm:"column:approval_reason"`
	RejectedAt            *time.Time `gorm:"column:rejected_at"`
	RejectionReason       string     `gorm:"column:rejection_reason"`
	RejectionNotes        string     `gorm:"column:rejection_notes"`
	VerificationStart     *time.Time `gorm:"column:verification_start"`
	VerificationWindowEnd *time.Time `gorm:"column:verification_window_end"`
	ViewsCount            int        `gorm:"column:views_count"`
	LockedViews           *int       `gorm:"column:locked_views"`
	LockedAt              *time.Time `gorm:"column:locked_at"`
	LastViewSync          *time.Time `gorm:"column:last_view_sync"`
	CpvRate               float64    `gorm:"column:cpv_rate"`
	GrossAmount           float64    `gorm:"column:gross_amount"`
	PlatformFee           float64    `gorm:"column:platform_fee"`
	NetAmount             float64    `gorm:"column:net_amount"`
	Metadata              []byte     `gorm:"column:metadata"`
	ReportedCount         int        `gorm:"column:reported_count"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
}

func (submissionModel) TableName() string {
	return "submissions"
}

func submissionModelFromEntity(item entities.Submission) submissionModel {
	metadata := map[string]any{}
	if item.Metadata != nil {
		metadata = item.Metadata
	}
	metadataRaw, _ := json.Marshal(metadata)
	return submissionModel{
		SubmissionID:          strings.TrimSpace(item.SubmissionID),
		CampaignID:            strings.TrimSpace(item.CampaignID),
		CreatorID:             strings.TrimSpace(item.CreatorID),
		Platform:              strings.TrimSpace(item.Platform),
		PostURL:               strings.TrimSpace(item.PostURL),
		PostID:                strings.TrimSpace(item.PostID),
		CreatorPlatformHandle: strings.TrimSpace(item.CreatorPlatformHandle),
		Status:                string(item.Status),
		CreatedAt:             item.CreatedAt.UTC(),
		ApprovedAt:            normalizeOptionalTime(item.ApprovedAt),
		ApprovedByUserID:      strings.TrimSpace(item.ApprovedByUserID),
		ApprovalReason:        strings.TrimSpace(item.ApprovalReason),
		RejectedAt:            normalizeOptionalTime(item.RejectedAt),
		RejectionReason:       strings.TrimSpace(item.RejectionReason),
		RejectionNotes:        strings.TrimSpace(item.RejectionNotes),
		VerificationStart:     normalizeOptionalTime(item.VerificationStart),
		VerificationWindowEnd: normalizeOptionalTime(item.VerificationWindowEnd),
		ViewsCount:            item.ViewsCount,
		LockedViews:           item.LockedViews,
		LockedAt:              normalizeOptionalTime(item.LockedAt),
		LastViewSync:          normalizeOptionalTime(item.LastViewSync),
		CpvRate:               item.CpvRate,
		GrossAmount:           item.GrossAmount,
		PlatformFee:           item.PlatformFee,
		NetAmount:             item.NetAmount,
		Metadata:              metadataRaw,
		ReportedCount:         item.ReportedCount,
		UpdatedAt:             item.UpdatedAt.UTC(),
	}
}

func submissionUpdatesFromEntity(item entities.Submission) map[string]any {
	row := submissionModelFromEntity(item)
	return map[string]any{
		"campaign_id":             row.CampaignID,
		"creator_id":              row.CreatorID,
		"platform":                row.Platform,
		"post_url":                row.PostURL,
		"post_id":                 row.PostID,
		"creator_platform_handle": row.CreatorPlatformHandle,
		"status":                  row.Status,
		"created_at":              row.CreatedAt,
		"approved_at":             row.ApprovedAt,
		"approved_by_user_id":     row.ApprovedByUserID,
		"approval_reason":         row.ApprovalReason,
		"rejected_at":             row.RejectedAt,
		"rejection_reason":        row.RejectionReason,
		"rejection_notes":         row.RejectionNotes,
		"verification_start":      row.VerificationStart,
		"verification_window_end": row.VerificationWindowEnd,
		"views_count":             row.ViewsCount,
		"locked_views":            row.LockedViews,
		"locked_at":               row.LockedAt,
		"last_view_sync":          row.LastViewSync,
		"cpv_rate":                row.CpvRate,
		"gross_amount":            row.GrossAmount,
		"platform_fee":            row.PlatformFee,
		"net_amount":              row.NetAmount,
		"metadata":                row.Metadata,
		"reported_count":          row.ReportedCount,
		"updated_at":              row.UpdatedAt,
	}
}

func (m submissionModel) toEntity() entities.Submission {
	metadata := map[string]any{}
	if len(m.Metadata) > 0 {
		_ = json.Unmarshal(m.Metadata, &metadata)
	}
	return entities.Submission{
		SubmissionID:          m.SubmissionID,
		CampaignID:            m.CampaignID,
		CreatorID:             m.CreatorID,
		Platform:              m.Platform,
		PostURL:               m.PostURL,
		PostID:                m.PostID,
		CreatorPlatformHandle: m.CreatorPlatformHandle,
		Status:                entities.SubmissionStatus(m.Status),
		CreatedAt:             m.CreatedAt.UTC(),
		ApprovedAt:            normalizeOptionalTime(m.ApprovedAt),
		ApprovedByUserID:      m.ApprovedByUserID,
		ApprovalReason:        m.ApprovalReason,
		RejectedAt:            normalizeOptionalTime(m.RejectedAt),
		RejectionReason:       m.RejectionReason,
		RejectionNotes:        m.RejectionNotes,
		VerificationStart:     normalizeOptionalTime(m.VerificationStart),
		VerificationWindowEnd: normalizeOptionalTime(m.VerificationWindowEnd),
		ViewsCount:            m.ViewsCount,
		LockedViews:           m.LockedViews,
		LockedAt:              normalizeOptionalTime(m.LockedAt),
		LastViewSync:          normalizeOptionalTime(m.LastViewSync),
		CpvRate:               m.CpvRate,
		GrossAmount:           m.GrossAmount,
		PlatformFee:           m.PlatformFee,
		NetAmount:             m.NetAmount,
		Metadata:              metadata,
		ReportedCount:         m.ReportedCount,
		UpdatedAt:             m.UpdatedAt.UTC(),
	}
}

type submissionAuditModel struct {
	AuditID      string    `gorm:"column:audit_id;primaryKey"`
	SubmissionID string    `gorm:"column:submission_id"`
	Action       string    `gorm:"column:action"`
	OldStatus    string    `gorm:"column:old_status"`
	NewStatus    string    `gorm:"column:new_status"`
	ActorID      string    `gorm:"column:actor_id"`
	ActorRole    string    `gorm:"column:actor_role"`
	ReasonCode   string    `gorm:"column:reason_code"`
	ReasonNotes  string    `gorm:"column:reason_notes"`
	IPAddress    string    `gorm:"column:ip_address"`
	UserAgent    string    `gorm:"column:user_agent"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (submissionAuditModel) TableName() string {
	return "submissions_audit"
}

type submissionFlagModel struct {
	FlagID       string     `gorm:"column:flag_id;primaryKey"`
	SubmissionID string     `gorm:"column:submission_id"`
	FlagType     string     `gorm:"column:flag_type"`
	Severity     string     `gorm:"column:severity"`
	Details      []byte     `gorm:"column:details"`
	IsResolved   bool       `gorm:"column:is_resolved"`
	ResolvedAt   *time.Time `gorm:"column:resolved_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (submissionFlagModel) TableName() string {
	return "submission_flags"
}

type submissionReportModel struct {
	ReportID         string    `gorm:"column:report_id;primaryKey"`
	SubmissionID     string    `gorm:"column:submission_id"`
	ReportedByUserID string    `gorm:"column:reported_by_user_id"`
	Reason           string    `gorm:"column:reason"`
	Description      string    `gorm:"column:description"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

func (submissionReportModel) TableName() string {
	return "submission_reports"
}

type bulkSubmissionOperationModel struct {
	OperationID       string    `gorm:"column:operation_id;primaryKey"`
	CampaignID        string    `gorm:"column:campaign_id"`
	OperationType     string    `gorm:"column:operation_type"`
	SubmissionIDs     []string  `gorm:"column:submission_ids;type:uuid[]"`
	PerformedByUserID string    `gorm:"column:performed_by_user_id"`
	SucceededCount    int       `gorm:"column:succeeded_count"`
	FailedCount       int       `gorm:"column:failed_count"`
	ReasonCode        string    `gorm:"column:reason_code"`
	ReasonNotes       string    `gorm:"column:reason_notes"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

func (bulkSubmissionOperationModel) TableName() string {
	return "bulk_submission_operations"
}

type viewSnapshotModel struct {
	SnapshotID         string    `gorm:"column:snapshot_id;primaryKey"`
	SubmissionID       string    `gorm:"column:submission_id"`
	ViewsCount         int       `gorm:"column:views_count"`
	EngagementEstimate int       `gorm:"column:engagement_estimate"`
	PlatformMetrics    []byte    `gorm:"column:platform_metrics"`
	SyncedAt           time.Time `gorm:"column:synced_at"`
	IsAnomaly          bool      `gorm:"column:is_anomaly"`
	AnomalyReason      string    `gorm:"column:anomaly_reason"`
}

func (viewSnapshotModel) TableName() string {
	return "view_snapshots"
}

type idempotencyModel struct {
	Key             string    `gorm:"column:key;primaryKey"`
	RequestHash     string    `gorm:"column:request_hash"`
	ResponsePayload []byte    `gorm:"column:response_payload"`
	ExpiresAt       time.Time `gorm:"column:expires_at"`
}

func (idempotencyModel) TableName() string {
	return "submission_idempotency"
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
	return "submission_outbox"
}

type eventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	PayloadHash string    `gorm:"column:payload_hash"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (eventDedupModel) TableName() string {
	return "submission_event_dedup"
}

type campaignProjectionModel struct {
	CampaignID       string   `gorm:"column:campaign_id;primaryKey"`
	Status           string   `gorm:"column:status"`
	AllowedPlatforms []string `gorm:"column:allowed_platforms;type:text[]"`
	RatePer1KViews   float64  `gorm:"column:rate_per_1k_views"`
}

func (campaignProjectionModel) TableName() string {
	return "campaigns"
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
