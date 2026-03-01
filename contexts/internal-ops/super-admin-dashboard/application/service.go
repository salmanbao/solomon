package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/internal-ops/super-admin-dashboard/domain/errors"
	"solomon/contexts/internal-ops/super-admin-dashboard/ports"
)

type Service struct {
	Repo          ports.Repository
	Idempotency   ports.IdempotencyStore
	Clock         ports.Clock
	Logger        *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) StartImpersonation(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	reason string,
) (ports.ImpersonationSession, error) {
	var out ports.ImpersonationSession
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("start_impersonation", adminID, userID, reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.StartImpersonation(ctx, adminID, userID, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) EndImpersonation(
	ctx context.Context,
	idempotencyKey string,
	impersonationID string,
) (ports.ImpersonationSession, error) {
	var out ports.ImpersonationSession
	if strings.TrimSpace(impersonationID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("end_impersonation", impersonationID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.EndImpersonation(ctx, impersonationID)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) AdjustWallet(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	amount float64,
	adjustmentType string,
	reason string,
) (ports.WalletAdjustment, error) {
	var out ports.WalletAdjustment
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(adjustmentType) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if amount == 0 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("adjust_wallet", adminID, userID, fmt.Sprintf("%.4f", amount), adjustmentType, reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.AdjustWallet(ctx, adminID, userID, amount, adjustmentType, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) ListWalletHistory(ctx context.Context, userID string, cursor string, limit int) ([]ports.WalletAdjustment, string, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, "", domainerrors.ErrInvalidRequest
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.Repo.ListWalletHistory(ctx, userID, cursor, limit)
}

func (s Service) BanUser(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	banType string,
	durationDays int,
	reason string,
) (ports.UserBan, error) {
	var out ports.UserBan
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(banType) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("ban_user", adminID, userID, banType, fmt.Sprintf("%d", durationDays), reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.BanUser(ctx, adminID, userID, banType, durationDays, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) UnbanUser(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	reason string,
) (ports.UserBan, error) {
	var out ports.UserBan
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("unban_user", adminID, userID, reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.UnbanUser(ctx, adminID, userID, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) SearchUsers(ctx context.Context, query string, status string, cursor string, pageSize int) ([]ports.AdminUser, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.Repo.SearchUsers(ctx, query, status, cursor, pageSize)
}

func (s Service) BulkAction(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userIDs []string,
	action string,
) (ports.BulkActionJob, error) {
	var out ports.BulkActionJob
	if len(userIDs) == 0 || strings.TrimSpace(action) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(map[string]any{"user_ids": userIDs, "action": action, "admin_id": adminID})
	requestHash := hashStrings("bulk_action", string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(encoded []byte) error { return json.Unmarshal(encoded, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CreateBulkActionJob(ctx, adminID, userIDs, action)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) PauseCampaign(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	campaignID string,
	reason string,
) (ports.CampaignPauseResult, error) {
	var out ports.CampaignPauseResult
	if strings.TrimSpace(campaignID) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("pause_campaign", adminID, campaignID, reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(encoded []byte) error { return json.Unmarshal(encoded, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.PauseCampaign(ctx, adminID, campaignID, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) AdjustCampaign(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	campaignID string,
	newBudget float64,
	newRatePer1kViews float64,
	reason string,
) (ports.CampaignAdjustResult, error) {
	var out ports.CampaignAdjustResult
	if strings.TrimSpace(campaignID) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if newBudget < 0 || newRatePer1kViews <= 0 {
		return out, domainerrors.ErrUnprocessable
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings(
		"adjust_campaign",
		adminID,
		campaignID,
		fmt.Sprintf("%.2f", newBudget),
		fmt.Sprintf("%.4f", newRatePer1kViews),
		reason,
	)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(encoded []byte) error { return json.Unmarshal(encoded, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.AdjustCampaign(ctx, adminID, campaignID, newBudget, newRatePer1kViews, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) OverrideSubmission(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	submissionID string,
	newStatus string,
	reason string,
) (ports.SubmissionOverride, error) {
	var out ports.SubmissionOverride
	if strings.TrimSpace(submissionID) == "" || strings.TrimSpace(newStatus) == "" || strings.TrimSpace(reason) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("override_submission", adminID, submissionID, newStatus, reason)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(encoded []byte) error { return json.Unmarshal(encoded, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.OverrideSubmission(ctx, adminID, submissionID, newStatus, reason)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) ListFeatureFlags(ctx context.Context) ([]ports.FeatureFlag, error) {
	return s.Repo.ListFeatureFlags(ctx)
}

func (s Service) ToggleFeatureFlag(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	flagKey string,
	enabled bool,
	reason string,
	config map[string]any,
) (ports.FeatureFlag, bool, error) {
	var out ports.FeatureFlag
	var oldEnabled bool
	if strings.TrimSpace(flagKey) == "" || strings.TrimSpace(reason) == "" {
		return out, oldEnabled, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, oldEnabled, err
	}
	payload, _ := json.Marshal(config)
	requestHash := hashStrings("toggle_flag", adminID, flagKey, fmt.Sprintf("%t", enabled), reason, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(encoded []byte) error {
			var wrap struct {
				Flag       ports.FeatureFlag `json:"flag"`
				OldEnabled bool              `json:"old_enabled"`
			}
			if err := json.Unmarshal(encoded, &wrap); err != nil {
				return err
			}
			out = wrap.Flag
			oldEnabled = wrap.OldEnabled
			return nil
		},
		func() ([]byte, error) {
			flag, old, err := s.Repo.ToggleFeatureFlag(ctx, adminID, flagKey, enabled, reason, config)
			if err != nil {
				return nil, err
			}
			return json.Marshal(struct {
				Flag       ports.FeatureFlag `json:"flag"`
				OldEnabled bool              `json:"old_enabled"`
			}{Flag: flag, OldEnabled: old})
		},
	)
	return out, oldEnabled, err
}

func (s Service) GetAnalyticsDashboard(ctx context.Context, start time.Time, end time.Time) (ports.AnalyticsDashboard, error) {
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return ports.AnalyticsDashboard{}, domainerrors.ErrUnprocessable
	}
	return s.Repo.GetAnalyticsDashboard(ctx, start, end)
}

func (s Service) ListAuditLogs(ctx context.Context, adminID string, actionType string, cursor string, pageSize int) ([]ports.AuditLog, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.Repo.ListAuditLogs(ctx, adminID, actionType, cursor, pageSize)
}

func (s Service) ExportAuditLogs(ctx context.Context, format string, start time.Time, end time.Time, includeSignatures bool) (ports.AuditExport, error) {
	if strings.TrimSpace(format) == "" {
		return ports.AuditExport{}, domainerrors.ErrInvalidRequest
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return ports.AuditExport{}, domainerrors.ErrUnprocessable
	}
	return s.Repo.CreateAuditExport(ctx, format, start, end, includeSignatures)
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) requireIdempotency(key string) error {
	if strings.TrimSpace(key) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	return nil
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	logger := ResolveLogger(s.Logger)
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
	logger.Debug("super admin idempotent operation committed",
		"event", "super_admin_idempotent_operation_committed",
		"module", "internal-ops/super-admin-dashboard",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}