package postgresadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/voting-engine/domain/errors"
	"solomon/contexts/campaign-editorial/voting-engine/ports"

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

func (r *Repository) SaveVote(ctx context.Context, vote entities.Vote) error {
	row := voteModelFromEntity(vote)
	create := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"submission_id":             row.SubmissionID,
			"campaign_id":               row.CampaignID,
			"round_id":                  row.RoundID,
			"user_id":                   row.UserID,
			"vote_type":                 row.VoteType,
			"weight":                    row.Weight,
			"reputation_score_snapshot": row.ReputationScoreSnapshot,
			"ip_address":                row.IPAddress,
			"user_agent":                row.UserAgent,
			"retracted":                 row.Retracted,
			"updated_at":                row.UpdatedAt,
		}),
	}).Create(&row)
	if create.Error != nil {
		if isUniqueViolation(create.Error) {
			return domainerrors.ErrConflict
		}
		return r.logError("voting_repo_save_vote_failed", create.Error,
			"vote_id", strings.TrimSpace(vote.VoteID),
			"submission_id", strings.TrimSpace(vote.SubmissionID),
			"user_id", strings.TrimSpace(vote.UserID),
		)
	}
	return nil
}

func (r *Repository) GetVote(ctx context.Context, voteID string) (entities.Vote, error) {
	var row voteModel
	err := r.db.WithContext(ctx).
		Where("id = ?", strings.TrimSpace(voteID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Vote{}, domainerrors.ErrVoteNotFound
		}
		return entities.Vote{}, r.logError("voting_repo_get_vote_failed", err, "vote_id", strings.TrimSpace(voteID))
	}
	return row.toEntity(), nil
}

func (r *Repository) GetVoteByIdentity(
	ctx context.Context,
	submissionID string,
	userID string,
	roundID string,
) (entities.Vote, bool, error) {
	tx := r.db.WithContext(ctx).Model(&voteModel{}).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		Where("user_id = ?", strings.TrimSpace(userID))

	if strings.TrimSpace(roundID) == "" {
		tx = tx.Where("round_id IS NULL")
	} else {
		tx = tx.Where("round_id = ?", strings.TrimSpace(roundID))
	}

	var row voteModel
	err := tx.Order("updated_at DESC").First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.Vote{}, false, nil
		}
		return entities.Vote{}, false, r.logError("voting_repo_get_vote_by_identity_failed", err,
			"submission_id", strings.TrimSpace(submissionID),
			"user_id", strings.TrimSpace(userID),
			"round_id", strings.TrimSpace(roundID),
		)
	}
	return row.toEntity(), true, nil
}

func (r *Repository) ListVotesBySubmission(ctx context.Context, submissionID string) ([]entities.Vote, error) {
	var rows []voteModel
	if err := r.db.WithContext(ctx).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_votes_by_submission_failed", err,
			"submission_id", strings.TrimSpace(submissionID),
		)
	}
	return toVoteEntities(rows), nil
}

func (r *Repository) ListVotesByCampaign(ctx context.Context, campaignID string) ([]entities.Vote, error) {
	tx := r.db.WithContext(ctx).Model(&voteModel{})
	if strings.TrimSpace(campaignID) != "" {
		tx = tx.Where("campaign_id = ?", strings.TrimSpace(campaignID))
	}
	var rows []voteModel
	if err := tx.Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_votes_by_campaign_failed", err,
			"campaign_id", strings.TrimSpace(campaignID),
		)
	}
	return toVoteEntities(rows), nil
}

func (r *Repository) ListVotesByRound(ctx context.Context, roundID string) ([]entities.Vote, error) {
	var rows []voteModel
	if err := r.db.WithContext(ctx).
		Where("round_id = ?", strings.TrimSpace(roundID)).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_votes_by_round_failed", err,
			"round_id", strings.TrimSpace(roundID),
		)
	}
	return toVoteEntities(rows), nil
}

func (r *Repository) ListVotesByCreator(ctx context.Context, creatorID string) ([]entities.Vote, error) {
	var rows []voteModel
	err := r.db.WithContext(ctx).
		Table("votes AS v").
		Select("v.*").
		Joins("JOIN submissions AS s ON s.submission_id = v.submission_id").
		Where("s.creator_id = ?", strings.TrimSpace(creatorID)).
		Order("v.created_at ASC").
		Scan(&rows).
		Error
	if err != nil {
		return nil, r.logError("voting_repo_list_votes_by_creator_failed", err,
			"creator_id", strings.TrimSpace(creatorID),
		)
	}
	return toVoteEntities(rows), nil
}

func (r *Repository) ListVotes(ctx context.Context) ([]entities.Vote, error) {
	var rows []voteModel
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_votes_failed", err)
	}
	return toVoteEntities(rows), nil
}

func (r *Repository) GetSubmission(ctx context.Context, submissionID string) (ports.SubmissionProjection, error) {
	var row submissionProjectionModel
	err := r.db.WithContext(ctx).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.SubmissionProjection{}, domainerrors.ErrSubmissionNotFound
		}
		return ports.SubmissionProjection{}, r.logError("voting_repo_get_submission_failed", err,
			"submission_id", strings.TrimSpace(submissionID),
		)
	}
	return ports.SubmissionProjection{
		SubmissionID: row.SubmissionID,
		CampaignID:   row.CampaignID,
		CreatorID:    row.CreatorID,
		Status:       row.Status,
	}, nil
}

func (r *Repository) GetCampaign(ctx context.Context, campaignID string) (ports.CampaignProjection, error) {
	var row campaignProjectionModel
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", strings.TrimSpace(campaignID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.CampaignProjection{}, domainerrors.ErrCampaignNotFound
		}
		return ports.CampaignProjection{}, r.logError("voting_repo_get_campaign_failed", err,
			"campaign_id", strings.TrimSpace(campaignID),
		)
	}
	return ports.CampaignProjection{
		CampaignID: row.CampaignID,
		Status:     row.Status,
	}, nil
}

func (r *Repository) GetReputationScore(ctx context.Context, userID string) (float64, bool, error) {
	var row reputationScoreModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", strings.TrimSpace(userID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		if isUndefinedTable(err) {
			// M48 schema is optional in local development; callers fall back to weight 1.0x.
			return 0, false, nil
		}
		return 0, false, r.logError("voting_repo_get_reputation_score_failed", err,
			"user_id", strings.TrimSpace(userID),
		)
	}
	return row.OverallScore, true, nil
}

func (r *Repository) GetRound(ctx context.Context, roundID string) (entities.VotingRound, error) {
	var row votingRoundModel
	err := r.db.WithContext(ctx).
		Where("id = ?", strings.TrimSpace(roundID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.VotingRound{}, domainerrors.ErrRoundNotFound
		}
		return entities.VotingRound{}, r.logError("voting_repo_get_round_failed", err,
			"round_id", strings.TrimSpace(roundID),
		)
	}
	return row.toEntity(), nil
}

func (r *Repository) GetActiveRoundByCampaign(ctx context.Context, campaignID string) (entities.VotingRound, bool, error) {
	var row votingRoundModel
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", strings.TrimSpace(campaignID)).
		Where("status = ?", string(entities.RoundStatusActive)).
		Where("ends_at IS NULL OR ends_at > ?", time.Now().UTC()).
		Order("starts_at DESC").
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.VotingRound{}, false, nil
		}
		return entities.VotingRound{}, false, r.logError("voting_repo_get_active_round_failed", err,
			"campaign_id", strings.TrimSpace(campaignID),
		)
	}
	return row.toEntity(), true, nil
}

func (r *Repository) TransitionRoundsForCampaign(
	ctx context.Context,
	campaignID string,
	toStatus entities.RoundStatus,
	updatedAt time.Time,
) ([]entities.VotingRound, error) {
	var rows []votingRoundModel
	tx := r.db.WithContext(ctx).Model(&votingRoundModel{}).
		Where("campaign_id = ?", strings.TrimSpace(campaignID))

	switch toStatus {
	case entities.RoundStatusClosingSoon:
		tx = tx.Where("status = ?", string(entities.RoundStatusActive))
	case entities.RoundStatusClosed:
		tx = tx.Where("status IN ?", []string{
			string(entities.RoundStatusActive),
			string(entities.RoundStatusClosingSoon),
		})
	default:
		tx = tx.Where("status <> ?", string(toStatus))
	}

	if err := tx.Order("starts_at ASC").Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_transition_rounds_list_failed", err,
			"campaign_id", strings.TrimSpace(campaignID),
			"to_status", string(toStatus),
		)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	items := make([]entities.VotingRound, 0, len(rows))
	for _, row := range rows {
		updates := map[string]any{
			"status":     string(toStatus),
			"updated_at": updatedAt.UTC(),
		}
		if toStatus == entities.RoundStatusClosed && row.EndsAt == nil {
			updates["ends_at"] = updatedAt.UTC()
		}
		if err := r.db.WithContext(ctx).
			Model(&votingRoundModel{}).
			Where("id = ?", row.ID).
			Updates(updates).Error; err != nil {
			return nil, r.logError("voting_repo_transition_rounds_update_failed", err,
				"campaign_id", strings.TrimSpace(campaignID),
				"round_id", row.ID,
				"to_status", string(toStatus),
			)
		}
		row.Status = string(toStatus)
		row.UpdatedAt = updatedAt.UTC()
		if toStatus == entities.RoundStatusClosed && row.EndsAt == nil {
			endedAt := updatedAt.UTC()
			row.EndsAt = &endedAt
		}
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) GetQuarantine(ctx context.Context, quarantineID string) (entities.VoteQuarantine, error) {
	var row quarantineModel
	err := r.db.WithContext(ctx).
		Where("id = ?", strings.TrimSpace(quarantineID)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities.VoteQuarantine{}, domainerrors.ErrQuarantineNotFound
		}
		return entities.VoteQuarantine{}, r.logError("voting_repo_get_quarantine_failed", err,
			"quarantine_id", strings.TrimSpace(quarantineID),
		)
	}
	return row.toEntity(), nil
}

func (r *Repository) SaveQuarantine(ctx context.Context, quarantine entities.VoteQuarantine) error {
	row := quarantineModelFromEntity(quarantine)
	create := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"vote_id":    row.VoteID,
			"risk_score": row.RiskScore,
			"reason":     row.Reason,
			"status":     row.Status,
			"updated_at": row.UpdatedAt,
		}),
	}).Create(&row)
	if create.Error != nil {
		return r.logError("voting_repo_save_quarantine_failed", create.Error,
			"quarantine_id", strings.TrimSpace(quarantine.QuarantineID),
			"vote_id", strings.TrimSpace(quarantine.VoteID),
		)
	}
	return nil
}

func (r *Repository) ListQuarantines(ctx context.Context) ([]entities.VoteQuarantine, error) {
	var rows []quarantineModel
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_quarantines_failed", err)
	}
	items := make([]entities.VoteQuarantine, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) RetractVotesBySubmission(
	ctx context.Context,
	submissionID string,
	updatedAt time.Time,
) ([]entities.Vote, error) {
	var rows []voteModel
	if err := r.db.WithContext(ctx).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		Where("retracted = ?", false).
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_retract_votes_by_submission_list_failed", err,
			"submission_id", strings.TrimSpace(submissionID),
		)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	if err := r.db.WithContext(ctx).
		Model(&voteModel{}).
		Where("submission_id = ?", strings.TrimSpace(submissionID)).
		Where("retracted = ?", false).
		Updates(map[string]any{
			"retracted":  true,
			"updated_at": updatedAt.UTC(),
		}).Error; err != nil {
		return nil, r.logError("voting_repo_retract_votes_by_submission_update_failed", err,
			"submission_id", strings.TrimSpace(submissionID),
		)
	}

	items := make([]entities.Vote, 0, len(rows))
	for _, row := range rows {
		row.Retracted = true
		row.UpdatedAt = updatedAt.UTC()
		items = append(items, row.toEntity())
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	var row idempotencyModel
	err := r.db.WithContext(ctx).
		Where("key = ?", strings.TrimSpace(key)).
		First(&row).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.IdempotencyRecord{}, false, nil
		}
		return ports.IdempotencyRecord{}, false, r.logError("voting_repo_idempotency_get_failed", err,
			"idempotency_key", strings.TrimSpace(key),
		)
	}
	if !row.ExpiresAt.IsZero() && now.UTC().After(row.ExpiresAt.UTC()) {
		if err := r.db.WithContext(ctx).
			Where("key = ?", strings.TrimSpace(key)).
			Delete(&idempotencyModel{}).Error; err != nil {
			return ports.IdempotencyRecord{}, false, r.logError("voting_repo_idempotency_expire_delete_failed", err,
				"idempotency_key", strings.TrimSpace(key),
			)
		}
		return ports.IdempotencyRecord{}, false, nil
	}
	return ports.IdempotencyRecord{
		Key:         row.Key,
		RequestHash: row.RequestHash,
		VoteID:      row.VoteID,
		ExpiresAt:   row.ExpiresAt.UTC(),
	}, true, nil
}

func (r *Repository) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	row := idempotencyModel{
		Key:         strings.TrimSpace(record.Key),
		RequestHash: strings.TrimSpace(record.RequestHash),
		VoteID:      strings.TrimSpace(record.VoteID),
		ExpiresAt:   record.ExpiresAt.UTC(),
	}
	create := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoNothing: true,
	}).Create(&row)
	if create.Error != nil {
		return r.logError("voting_repo_idempotency_put_failed", create.Error, "idempotency_key", row.Key)
	}
	if create.RowsAffected > 0 {
		return nil
	}

	var existing idempotencyModel
	if err := r.db.WithContext(ctx).
		Where("key = ?", row.Key).
		First(&existing).Error; err != nil {
		return r.logError("voting_repo_idempotency_load_existing_failed", err, "idempotency_key", row.Key)
	}
	if existing.RequestHash != row.RequestHash || existing.VoteID != row.VoteID {
		return domainerrors.ErrIdempotencyConflict
	}
	return nil
}

func (r *Repository) AppendOutbox(ctx context.Context, envelope ports.EventEnvelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return r.logError("voting_repo_append_outbox_marshal_failed", err,
			"event_id", strings.TrimSpace(envelope.EventID),
			"event_type", strings.TrimSpace(envelope.EventType),
		)
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
	create := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "outbox_id"}},
		DoNothing: true,
	}).Create(&row)
	if create.Error != nil {
		return r.logError("voting_repo_append_outbox_insert_failed", create.Error,
			"outbox_id", row.OutboxID,
		)
	}
	if create.RowsAffected > 0 {
		return nil
	}

	var existing outboxModel
	if err := r.db.WithContext(ctx).
		Select("payload").
		Where("outbox_id = ?", row.OutboxID).
		First(&existing).Error; err != nil {
		return r.logError("voting_repo_append_outbox_load_existing_failed", err,
			"outbox_id", row.OutboxID,
		)
	}
	if !bytes.Equal(existing.Payload, row.Payload) {
		return domainerrors.ErrIdempotencyConflict
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
		Find(&rows).Error; err != nil {
		return nil, r.logError("voting_repo_list_pending_outbox_failed", err, "limit", limit)
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
		return r.logError("voting_repo_mark_outbox_published_failed", result.Error,
			"outbox_id", strings.TrimSpace(outboxID),
		)
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrConflict
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
	create := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "event_id"}},
		DoNothing: true,
	}).Create(&row)
	if create.Error != nil {
		return false, r.logError("voting_repo_reserve_event_failed", create.Error,
			"event_id", strings.TrimSpace(eventID),
		)
	}
	if create.RowsAffected > 0 {
		return false, nil
	}

	var existing eventDedupModel
	if err := r.db.WithContext(ctx).
		Select("payload_hash").
		Where("event_id = ?", row.EventID).
		First(&existing).Error; err != nil {
		return false, r.logError("voting_repo_reserve_event_load_existing_failed", err,
			"event_id", strings.TrimSpace(eventID),
		)
	}
	if existing.PayloadHash != row.PayloadHash {
		return false, domainerrors.ErrConflict
	}
	return true, nil
}

func (r *Repository) logError(event string, err error, attrs ...any) error {
	fields := make([]any, 0, len(attrs)+7)
	fields = append(fields,
		"event", event,
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"error", err.Error(),
	)
	fields = append(fields, attrs...)
	r.logger.Error("voting repository operation failed", fields...)
	return err
}

type voteModel struct {
	ID                      string    `gorm:"column:id;primaryKey"`
	SubmissionID            string    `gorm:"column:submission_id"`
	CampaignID              string    `gorm:"column:campaign_id"`
	RoundID                 *string   `gorm:"column:round_id"`
	UserID                  string    `gorm:"column:user_id"`
	VoteType                string    `gorm:"column:vote_type"`
	Weight                  float64   `gorm:"column:weight"`
	ReputationScoreSnapshot float64   `gorm:"column:reputation_score_snapshot"`
	IPAddress               string    `gorm:"column:ip_address"`
	UserAgent               string    `gorm:"column:user_agent"`
	Retracted               bool      `gorm:"column:retracted"`
	CreatedAt               time.Time `gorm:"column:created_at"`
	UpdatedAt               time.Time `gorm:"column:updated_at"`
}

func (voteModel) TableName() string {
	return "votes"
}

func voteModelFromEntity(vote entities.Vote) voteModel {
	row := voteModel{
		ID:                      strings.TrimSpace(vote.VoteID),
		SubmissionID:            strings.TrimSpace(vote.SubmissionID),
		CampaignID:              strings.TrimSpace(vote.CampaignID),
		UserID:                  strings.TrimSpace(vote.UserID),
		VoteType:                string(vote.VoteType),
		Weight:                  vote.Weight,
		ReputationScoreSnapshot: vote.ReputationScoreSnapshot,
		IPAddress:               strings.TrimSpace(vote.IPAddress),
		UserAgent:               strings.TrimSpace(vote.UserAgent),
		Retracted:               vote.Retracted,
		CreatedAt:               vote.CreatedAt.UTC(),
		UpdatedAt:               vote.UpdatedAt.UTC(),
	}
	if strings.TrimSpace(vote.RoundID) != "" {
		roundID := strings.TrimSpace(vote.RoundID)
		row.RoundID = &roundID
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	return row
}

func (m voteModel) toEntity() entities.Vote {
	roundID := ""
	if m.RoundID != nil {
		roundID = strings.TrimSpace(*m.RoundID)
	}
	return entities.Vote{
		VoteID:                  m.ID,
		SubmissionID:            m.SubmissionID,
		CampaignID:              m.CampaignID,
		RoundID:                 roundID,
		UserID:                  m.UserID,
		VoteType:                entities.VoteType(m.VoteType),
		Weight:                  m.Weight,
		ReputationScoreSnapshot: m.ReputationScoreSnapshot,
		IPAddress:               m.IPAddress,
		UserAgent:               m.UserAgent,
		Retracted:               m.Retracted,
		CreatedAt:               m.CreatedAt.UTC(),
		UpdatedAt:               m.UpdatedAt.UTC(),
	}
}

type votingRoundModel struct {
	ID         string     `gorm:"column:id;primaryKey"`
	CampaignID string     `gorm:"column:campaign_id"`
	Status     string     `gorm:"column:status"`
	StartsAt   time.Time  `gorm:"column:starts_at"`
	EndsAt     *time.Time `gorm:"column:ends_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at"`
}

func (votingRoundModel) TableName() string {
	return "voting_rounds"
}

func (m votingRoundModel) toEntity() entities.VotingRound {
	return entities.VotingRound{
		RoundID:    m.ID,
		CampaignID: m.CampaignID,
		Status:     entities.RoundStatus(m.Status),
		StartsAt:   m.StartsAt.UTC(),
		EndsAt:     normalizeOptionalTime(m.EndsAt),
		CreatedAt:  m.CreatedAt.UTC(),
		UpdatedAt:  m.UpdatedAt.UTC(),
	}
}

type quarantineModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	VoteID    string    `gorm:"column:vote_id"`
	RiskScore float64   `gorm:"column:risk_score"`
	Reason    string    `gorm:"column:reason"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (quarantineModel) TableName() string {
	return "vote_quarantine"
}

func quarantineModelFromEntity(item entities.VoteQuarantine) quarantineModel {
	row := quarantineModel{
		ID:        strings.TrimSpace(item.QuarantineID),
		VoteID:    strings.TrimSpace(item.VoteID),
		RiskScore: item.RiskScore,
		Reason:    strings.TrimSpace(item.Reason),
		Status:    string(item.Status),
		CreatedAt: item.CreatedAt.UTC(),
		UpdatedAt: item.UpdatedAt.UTC(),
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = row.CreatedAt
	}
	return row
}

func (m quarantineModel) toEntity() entities.VoteQuarantine {
	return entities.VoteQuarantine{
		QuarantineID: m.ID,
		VoteID:       m.VoteID,
		RiskScore:    m.RiskScore,
		Reason:       m.Reason,
		Status:       entities.QuarantineStatus(m.Status),
		CreatedAt:    m.CreatedAt.UTC(),
		UpdatedAt:    m.UpdatedAt.UTC(),
	}
}

type idempotencyModel struct {
	Key         string    `gorm:"column:key;primaryKey"`
	RequestHash string    `gorm:"column:request_hash"`
	VoteID      string    `gorm:"column:vote_id"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
}

func (idempotencyModel) TableName() string {
	return "voting_engine_idempotency"
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
	return "voting_outbox"
}

type eventDedupModel struct {
	EventID     string    `gorm:"column:event_id;primaryKey"`
	PayloadHash string    `gorm:"column:payload_hash"`
	ExpiresAt   time.Time `gorm:"column:expires_at"`
	ProcessedAt time.Time `gorm:"column:processed_at"`
}

func (eventDedupModel) TableName() string {
	return "voting_event_dedup"
}

type submissionProjectionModel struct {
	SubmissionID string `gorm:"column:submission_id;primaryKey"`
	CampaignID   string `gorm:"column:campaign_id"`
	CreatorID    string `gorm:"column:creator_id"`
	Status       string `gorm:"column:status"`
}

func (submissionProjectionModel) TableName() string {
	return "submissions"
}

type campaignProjectionModel struct {
	CampaignID string `gorm:"column:campaign_id;primaryKey"`
	Status     string `gorm:"column:status"`
}

func (campaignProjectionModel) TableName() string {
	return "campaigns"
}

type reputationScoreModel struct {
	UserID       string  `gorm:"column:user_id;primaryKey"`
	OverallScore float64 `gorm:"column:overall_score"`
}

func (reputationScoreModel) TableName() string {
	return "user_reputation_scores"
}

func toVoteEntities(rows []voteModel) []entities.Vote {
	items := make([]entities.Vote, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toEntity())
	}
	return items
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

func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}

var _ ports.VoteRepository = (*Repository)(nil)
var _ ports.IdempotencyStore = (*Repository)(nil)
var _ ports.OutboxWriter = (*Repository)(nil)
var _ ports.OutboxRepository = (*Repository)(nil)
var _ ports.EventDedupStore = (*Repository)(nil)
