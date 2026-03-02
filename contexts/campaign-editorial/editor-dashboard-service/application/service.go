package application

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/editor-dashboard-service/domain/errors"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	EventDedup     ports.EventDedupStore
	Clock          ports.Clock
	IdempotencyTTL time.Duration
	EventDedupTTL  time.Duration
	Logger         *slog.Logger
}

func (s Service) GetFeed(ctx context.Context, userID string, query ports.FeedQuery) ([]ports.FeedItem, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		return nil, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetFeed(ctx, userID, query)
}

func (s Service) ListSubmissions(ctx context.Context, userID string, query ports.SubmissionQuery) ([]ports.SubmissionRecord, error) {
	userID = strings.TrimSpace(userID)
	query.Status = strings.TrimSpace(strings.ToLower(query.Status))
	if userID == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	if query.Status != "" {
		switch query.Status {
		case "approved", "rejected", "pending", "flagged":
		default:
			return nil, domainerrors.ErrInvalidRequest
		}
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 50 {
		query.Limit = 50
	}
	if query.Offset < 0 {
		return nil, domainerrors.ErrInvalidRequest
	}
	return s.Repo.ListSubmissions(ctx, userID, query)
}

func (s Service) GetEarnings(ctx context.Context, userID string) (ports.EarningsSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ports.EarningsSummary{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetEarnings(ctx, userID)
}

func (s Service) GetPerformance(ctx context.Context, userID string) (ports.PerformanceSummary, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ports.PerformanceSummary{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetPerformance(ctx, userID)
}

func (s Service) SaveCampaign(ctx context.Context, idempotencyKey string, userID string, campaignID string) (ports.SaveCampaignResult, error) {
	return s.saveOrUnsave(ctx, idempotencyKey, userID, campaignID, true)
}

func (s Service) RemoveSavedCampaign(ctx context.Context, idempotencyKey string, userID string, campaignID string) (ports.SaveCampaignResult, error) {
	return s.saveOrUnsave(ctx, idempotencyKey, userID, campaignID, false)
}

func (s Service) ExportSubmissionsCSV(ctx context.Context, userID string, query ports.SubmissionQuery) (string, error) {
	items, err := s.ListSubmissions(ctx, userID, query)
	if err != nil {
		return "", err
	}
	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	_ = writer.Write([]string{"submission_id", "campaign_id", "campaign_title", "status", "views", "earnings", "submitted_at"})
	for _, item := range items {
		_ = writer.Write([]string{
			item.SubmissionID,
			item.CampaignID,
			item.CampaignTitle,
			item.Status,
			strconv.Itoa(item.Views),
			strconv.FormatFloat(item.Earnings, 'f', 2, 64),
			item.SubmittedAt.UTC().Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func (s Service) ApplySubmissionLifecycleEvent(ctx context.Context, event ports.SubmissionLifecycleEvent) error {
	event.EventID = strings.TrimSpace(event.EventID)
	event.SubmissionID = strings.TrimSpace(event.SubmissionID)
	event.UserID = strings.TrimSpace(event.UserID)
	event.Status = strings.TrimSpace(strings.ToLower(event.Status))
	if event.EventID == "" || event.SubmissionID == "" || event.UserID == "" {
		return domainerrors.ErrInvalidRequest
	}
	processed, err := s.EventDedup.HasProcessedEvent(ctx, event.EventID, s.now())
	if err != nil {
		return err
	}
	if processed {
		return nil
	}
	if err := s.Repo.ApplySubmissionLifecycleEvent(ctx, event); err != nil {
		return err
	}
	return s.EventDedup.MarkProcessedEvent(ctx, event.EventID, s.now().Add(s.eventDedupTTL()))
}

func (s Service) saveOrUnsave(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	campaignID string,
	save bool,
) (ports.SaveCampaignResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	userID = strings.TrimSpace(userID)
	campaignID = strings.TrimSpace(campaignID)
	if idempotencyKey == "" {
		return ports.SaveCampaignResult{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if userID == "" || campaignID == "" {
		return ports.SaveCampaignResult{}, domainerrors.ErrInvalidRequest
	}

	action := "unsave"
	if save {
		action = "save"
	}
	requestHash := hashStrings(userID, campaignID, action)
	var output ports.SaveCampaignResult
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			var result ports.SaveCampaignResult
			var err error
			if save {
				result, err = s.Repo.SaveCampaign(ctx, ports.SaveCampaignCommand{UserID: userID, CampaignID: campaignID}, s.now())
			} else {
				result, err = s.Repo.RemoveSavedCampaign(ctx, ports.SaveCampaignCommand{UserID: userID, CampaignID: campaignID}, s.now())
			}
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return output, err
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) eventDedupTTL() time.Duration {
	if s.EventDedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.EventDedupTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("editor dashboard idempotent mutation committed",
		"event", "editor_dashboard_idempotent_mutation_committed",
		"module", "campaign-editorial/editor-dashboard-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
