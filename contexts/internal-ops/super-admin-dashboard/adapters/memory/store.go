package memory

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/internal-ops/super-admin-dashboard/domain/errors"
	"solomon/contexts/internal-ops/super-admin-dashboard/ports"
)

type Store struct {
	mu            sync.RWMutex
	users         map[string]ports.AdminUser
	balances      map[string]float64
	wallet        []ports.WalletAdjustment
	impersonation map[string]ports.ImpersonationSession
	bans          map[string]ports.UserBan
	flags         map[string]ports.FeatureFlag
	campaigns     map[string]campaignState
	submissions   map[string]string
	audits        []ports.AuditLog
	idempotency   map[string]ports.IdempotencyRecord
	sequence      uint64
}

type campaignState struct {
	Status        string
	Budget        float64
	RatePer1kView float64
}

func NewStore() *Store {
	now := time.Now().UTC()
	return &Store{
		users: map[string]ports.AdminUser{
			"user-1": {
				UserID:        "user-1",
				Email:         "user1@example.com",
				Username:      "user_one",
				Role:          "creator",
				CreatedAt:     now.Add(-90 * 24 * time.Hour),
				TotalEarnings: 1200.25,
				Status:        "active",
				KYCStatus:     "verified",
			},
			"user-2": {
				UserID:        "user-2",
				Email:         "user2@example.com",
				Username:      "user_two",
				Role:          "creator",
				CreatedAt:     now.Add(-60 * 24 * time.Hour),
				TotalEarnings: 480.00,
				Status:        "active",
				KYCStatus:     "pending",
			},
		},
		balances: map[string]float64{
			"user-1": 500.00,
			"user-2": 150.00,
		},
		wallet:        make([]ports.WalletAdjustment, 0),
		impersonation: make(map[string]ports.ImpersonationSession),
		bans:          make(map[string]ports.UserBan),
		flags: map[string]ports.FeatureFlag{
			"new_dashboard_widget": {
				FlagKey:   "new_dashboard_widget",
				Enabled:   false,
				Config:    map[string]any{"rollout": "0%"},
				UpdatedAt: now,
				UpdatedBy: "system",
			},
		},
		campaigns: map[string]campaignState{
			"campaign-1": {Status: "active", Budget: 1000, RatePer1kView: 1.50},
		},
		submissions: map[string]string{
			"submission-1": "flagged",
		},
		audits:      make([]ports.AuditLog, 0),
		idempotency: make(map[string]ports.IdempotencyRecord),
	}
}

func (s *Store) StartImpersonation(ctx context.Context, adminID string, userID string, reason string) (ports.ImpersonationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return ports.ImpersonationSession{}, domainerrors.ErrUserNotFound
	}
	if ban, ok := s.bans[userID]; ok && ban.Status == "active" {
		return ports.ImpersonationSession{}, domainerrors.ErrForbidden
	}
	for _, item := range s.impersonation {
		if item.AdminID == adminID && item.Status == "active" {
			return ports.ImpersonationSession{}, domainerrors.ErrImpersonationAlreadyActive
		}
	}
	now := s.Now()
	impersonationID := s.nextID("imp")
	session := ports.ImpersonationSession{
		ImpersonationID: impersonationID,
		UserID:          userID,
		AccessToken:     "imp-token-" + impersonationID,
		TokenExpiresAt:  now.Add(2 * time.Hour),
		StartedAt:       now,
		Status:          "active",
		Reason:          reason,
		AdminID:         adminID,
	}
	s.impersonation[impersonationID] = session
	s.appendAudit(adminID, "impersonation.start", "user", userID, nil, map[string]any{"impersonation_id": impersonationID}, reason, "")
	return session, nil
}

func (s *Store) EndImpersonation(ctx context.Context, impersonationID string) (ports.ImpersonationSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.impersonation[impersonationID]
	if !ok {
		return ports.ImpersonationSession{}, domainerrors.ErrImpersonationNotFound
	}
	if session.Status != "active" {
		return ports.ImpersonationSession{}, domainerrors.ErrImpersonationAlreadyEnded
	}
	now := s.Now()
	session.Status = "ended"
	session.EndedAt = &now
	s.impersonation[impersonationID] = session
	s.appendAudit(session.AdminID, "impersonation.end", "user", session.UserID, nil, map[string]any{"impersonation_id": impersonationID}, "impersonation ended", "")
	return session, nil
}

func (s *Store) AdjustWallet(ctx context.Context, adminID string, userID string, amount float64, adjustmentType string, reason string) (ports.WalletAdjustment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return ports.WalletAdjustment{}, domainerrors.ErrUserNotFound
	}
	before := s.balances[userID]
	after := before
	switch strings.ToLower(strings.TrimSpace(adjustmentType)) {
	case "credit":
		after += amount
	case "debit":
		if before < amount {
			return ports.WalletAdjustment{}, domainerrors.ErrConflict
		}
		after -= amount
	default:
		return ports.WalletAdjustment{}, domainerrors.ErrInvalidRequest
	}
	now := s.Now()
	adjustment := ports.WalletAdjustment{
		AdjustmentID:  s.nextID("adj"),
		UserID:        userID,
		Amount:        amount,
		Type:          strings.ToLower(adjustmentType),
		Reason:        reason,
		BalanceBefore: before,
		BalanceAfter:  after,
		AdjustedAt:    now,
		AuditLogID:    s.nextID("audit"),
		AdminID:       adminID,
	}
	s.balances[userID] = after
	s.wallet = append([]ports.WalletAdjustment{adjustment}, s.wallet...)
	s.appendAudit(adminID, "wallet.adjust", "user", userID, map[string]any{"balance": before}, map[string]any{"balance": after, "amount": amount}, reason, adjustment.AuditLogID)
	return adjustment, nil
}

func (s *Store) ListWalletHistory(ctx context.Context, userID string, cursor string, limit int) ([]ports.WalletAdjustment, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.users[userID]; !ok {
		return nil, "", domainerrors.ErrUserNotFound
	}
	items := make([]ports.WalletAdjustment, 0)
	for _, item := range s.wallet {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	start := decodeCursor(cursor)
	if start < 0 || start > len(items) {
		start = 0
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) {
		next = encodeCursor(end)
	}
	return append([]ports.WalletAdjustment(nil), items[start:end]...), next, nil
}

func (s *Store) BanUser(ctx context.Context, adminID string, userID string, banType string, durationDays int, reason string) (ports.UserBan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[userID]; !ok {
		return ports.UserBan{}, domainerrors.ErrUserNotFound
	}
	if existing, ok := s.bans[userID]; ok && existing.Status == "active" {
		return ports.UserBan{}, domainerrors.ErrAlreadyBanned
	}
	now := s.Now()
	ban := ports.UserBan{
		BanID:              s.nextID("ban"),
		UserID:             userID,
		BanType:            strings.ToLower(banType),
		BannedAt:           now,
		AllSessionsRevoked: true,
		AuditLogID:         s.nextID("audit"),
		Status:             "active",
		Reason:             reason,
	}
	if ban.BanType == "temporary" {
		if durationDays <= 0 {
			return ports.UserBan{}, domainerrors.ErrUnprocessable
		}
		exp := now.Add(time.Duration(durationDays) * 24 * time.Hour)
		ban.ExpiresAt = &exp
	}
	s.bans[userID] = ban
	user := s.users[userID]
	user.Status = "banned"
	s.users[userID] = user
	s.appendAudit(adminID, "user.ban", "user", userID, map[string]any{"status": "active"}, map[string]any{"status": "banned", "ban_type": ban.BanType}, reason, ban.AuditLogID)
	return ban, nil
}

func (s *Store) UnbanUser(ctx context.Context, adminID string, userID string, reason string) (ports.UserBan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ban, ok := s.bans[userID]
	if !ok || ban.Status != "active" {
		return ports.UserBan{}, domainerrors.ErrNotBanned
	}
	now := s.Now()
	ban.Status = "inactive"
	if ban.ExpiresAt == nil {
		ban.ExpiresAt = &now
	}
	s.bans[userID] = ban
	user := s.users[userID]
	user.Status = "active"
	s.users[userID] = user
	s.appendAudit(adminID, "user.unban", "user", userID, map[string]any{"status": "banned"}, map[string]any{"status": "active"}, reason, s.nextID("audit"))
	return ban, nil
}

func (s *Store) SearchUsers(ctx context.Context, query string, status string, cursor string, pageSize int) ([]ports.AdminUser, string, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	status = strings.ToLower(strings.TrimSpace(status))
	items := make([]ports.AdminUser, 0, len(s.users))
	for _, user := range s.users {
		if status != "" && strings.ToLower(user.Status) != status {
			continue
		}
		if query != "" {
			if !strings.Contains(strings.ToLower(user.UserID), query) &&
				!strings.Contains(strings.ToLower(user.Email), query) &&
				!strings.Contains(strings.ToLower(user.Username), query) {
				continue
			}
		}
		items = append(items, user)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	start := decodeCursor(cursor)
	if start < 0 || start > total {
		start = 0
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	next := ""
	if end < total {
		next = encodeCursor(end)
	}
	return append([]ports.AdminUser(nil), items[start:end]...), next, total, nil
}

func (s *Store) CreateBulkActionJob(ctx context.Context, adminID string, userIDs []string, action string) (ports.BulkActionJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(userIDs) > 1000 {
		return ports.BulkActionJob{}, domainerrors.ErrBulkActionConflict
	}
	now := s.Now()
	job := ports.BulkActionJob{
		JobID:                   s.nextID("job"),
		Action:                  action,
		UserCount:               len(userIDs),
		Status:                  "queued",
		CreatedAt:               now,
		EstimatedCompletionTime: now.Add(2 * time.Minute),
	}
	s.appendAudit(adminID, "users.bulk_action", "user", "*", nil, map[string]any{"job_id": job.JobID, "action": action, "user_count": len(userIDs)}, "bulk action queued", s.nextID("audit"))
	return job, nil
}

func (s *Store) PauseCampaign(ctx context.Context, adminID string, campaignID string, reason string) (ports.CampaignPauseResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.campaigns[campaignID]
	if !ok {
		return ports.CampaignPauseResult{}, domainerrors.ErrCampaignNotFound
	}
	if state.Status == "paused" {
		return ports.CampaignPauseResult{}, domainerrors.ErrConflict
	}
	now := s.Now()
	state.Status = "paused"
	s.campaigns[campaignID] = state
	result := ports.CampaignPauseResult{CampaignID: campaignID, Status: state.Status, PausedAt: now, AuditLogID: s.nextID("audit")}
	s.appendAudit(adminID, "campaign.pause", "campaign", campaignID, map[string]any{"status": "active"}, map[string]any{"status": "paused"}, reason, result.AuditLogID)
	return result, nil
}

func (s *Store) AdjustCampaign(ctx context.Context, adminID string, campaignID string, newBudget float64, newRate float64, reason string) (ports.CampaignAdjustResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.campaigns[campaignID]
	if !ok {
		return ports.CampaignAdjustResult{}, domainerrors.ErrCampaignNotFound
	}
	if newRate < 0.10 || newRate > 5.00 || newBudget < 0 {
		return ports.CampaignAdjustResult{}, domainerrors.ErrUnprocessable
	}
	now := s.Now()
	result := ports.CampaignAdjustResult{
		CampaignID:        campaignID,
		OldBudget:         state.Budget,
		NewBudget:         newBudget,
		OldRatePer1kViews: state.RatePer1kView,
		NewRatePer1kViews: newRate,
		AdjustedAt:        now,
		AuditLogID:        s.nextID("audit"),
	}
	state.Budget = newBudget
	state.RatePer1kView = newRate
	s.campaigns[campaignID] = state
	s.appendAudit(adminID, "campaign.adjust", "campaign", campaignID, map[string]any{"budget": result.OldBudget, "rate": result.OldRatePer1kViews}, map[string]any{"budget": newBudget, "rate": newRate}, reason, result.AuditLogID)
	return result, nil
}

func (s *Store) OverrideSubmission(ctx context.Context, adminID string, submissionID string, newStatus string, reason string) (ports.SubmissionOverride, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldStatus, ok := s.submissions[submissionID]
	if !ok {
		return ports.SubmissionOverride{}, domainerrors.ErrSubmissionNotFound
	}
	now := s.Now()
	s.submissions[submissionID] = newStatus
	result := ports.SubmissionOverride{
		SubmissionID: submissionID,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
		OverriddenAt: now,
		AuditLogID:   s.nextID("audit"),
	}
	s.appendAudit(adminID, "submission.override", "submission", submissionID, map[string]any{"status": oldStatus}, map[string]any{"status": newStatus}, reason, result.AuditLogID)
	return result, nil
}

func (s *Store) ListFeatureFlags(ctx context.Context) ([]ports.FeatureFlag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ports.FeatureFlag, 0, len(s.flags))
	for _, flag := range s.flags {
		items = append(items, flag)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].FlagKey < items[j].FlagKey })
	return items, nil
}

func (s *Store) ToggleFeatureFlag(ctx context.Context, adminID string, flagKey string, enabled bool, reason string, config map[string]any) (ports.FeatureFlag, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	flag, ok := s.flags[flagKey]
	if !ok {
		return ports.FeatureFlag{}, false, domainerrors.ErrFlagNotFound
	}
	oldEnabled := flag.Enabled
	flag.Enabled = enabled
	flag.Config = cloneMap(config)
	flag.UpdatedBy = adminID
	flag.UpdatedAt = s.Now()
	s.flags[flagKey] = flag
	s.appendAudit(adminID, "feature_flag.toggle", "feature_flag", flagKey, map[string]any{"enabled": oldEnabled}, map[string]any{"enabled": enabled}, reason, s.nextID("audit"))
	return flag, oldEnabled, nil
}

func (s *Store) GetAnalyticsDashboard(ctx context.Context, start time.Time, end time.Time) (ports.AnalyticsDashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if start.IsZero() {
		start = s.Now().Add(-7 * 24 * time.Hour)
	}
	if end.IsZero() {
		end = s.Now()
	}
	return ports.AnalyticsDashboard{
		DateRangeStart: start,
		DateRangeEnd:   end,
		TotalRevenue:   25000.75,
		UserGrowth:     len(s.users),
		CampaignCount:  len(s.campaigns),
		FraudAlerts:    2,
	}, nil
}

func (s *Store) ListAuditLogs(ctx context.Context, adminID string, actionType string, cursor string, pageSize int) ([]ports.AuditLog, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	adminID = strings.TrimSpace(adminID)
	actionType = strings.TrimSpace(actionType)
	items := make([]ports.AuditLog, 0, len(s.audits))
	for _, item := range s.audits {
		if adminID != "" && item.AdminID != adminID {
			continue
		}
		if actionType != "" && item.ActionType != actionType {
			continue
		}
		items = append(items, item)
	}
	start := decodeCursor(cursor)
	if start < 0 || start > len(items) {
		start = 0
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) {
		next = encodeCursor(end)
	}
	return append([]ports.AuditLog(nil), items[start:end]...), next, nil
}

func (s *Store) CreateAuditExport(ctx context.Context, format string, start time.Time, end time.Time, includeSignatures bool) (ports.AuditExport, error) {
	now := s.Now()
	return ports.AuditExport{
		ExportJobID:         s.nextID("export"),
		Status:              "queued",
		FileURL:             "https://exports.local/audit-logs.csv",
		CreatedAt:           now,
		EstimatedCompletion: now.Add(1 * time.Minute),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) NewID(ctx context.Context) (string, error) {
	return s.nextID("m20"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	id := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s-%d", prefix, id)
}

func (s *Store) appendAudit(
	adminID string,
	actionType string,
	targetResourceType string,
	targetResourceID string,
	oldValue map[string]any,
	newValue map[string]any,
	reason string,
	auditID string,
) {
	if strings.TrimSpace(auditID) == "" {
		auditID = s.nextID("audit")
	}
	now := s.Now()
	entry := ports.AuditLog{
		AuditID:            auditID,
		AdminID:            adminID,
		ActionType:         actionType,
		TargetResourceID:   targetResourceID,
		TargetResourceType: targetResourceType,
		OldValue:           cloneMap(oldValue),
		NewValue:           cloneMap(newValue),
		Reason:             reason,
		PerformedAt:        now,
		IPAddress:          "127.0.0.1",
		SignatureHash:      signatureHash(entrySeed(actionType, targetResourceType, targetResourceID, now)),
		IsVerified:         true,
	}
	s.audits = append([]ports.AuditLog{entry}, s.audits...)
}

func entrySeed(actionType string, targetType string, targetID string, now time.Time) string {
	return actionType + "|" + targetType + "|" + targetID + "|" + now.UTC().Format(time.RFC3339)
}

func signatureHash(seed string) string {
	sum := sha256String(seed)
	return "sig_" + sum[:16]
}

func sha256String(v string) string {
	sum := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", sum)
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func decodeCursor(cursor string) int {
	if strings.TrimSpace(cursor) == "" {
		return 0
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	idx, err := strconv.Atoi(string(raw))
	if err != nil || idx < 0 {
		return 0
	}
	return idx
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
