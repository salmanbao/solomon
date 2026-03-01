package ports

import (
	"context"
	"time"
)

// Clock allows deterministic TTL and timestamp handling.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts identifier generation for module records.
type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

// IdempotencyRecord stores a response payload keyed by idempotency key + request hash.
type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

// IdempotencyStore persists idempotency response records.
type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type ImpersonationSession struct {
	ImpersonationID string
	UserID          string
	AccessToken     string
	TokenExpiresAt  time.Time
	StartedAt       time.Time
	EndedAt         *time.Time
	Status          string
	Reason          string
	AdminID         string
}

type WalletAdjustment struct {
	AdjustmentID  string
	UserID        string
	Amount        float64
	Type          string
	Reason        string
	BalanceBefore float64
	BalanceAfter  float64
	AdjustedAt    time.Time
	AuditLogID    string
	AdminID       string
}

type UserBan struct {
	BanID               string
	UserID              string
	BanType             string
	BannedAt            time.Time
	ExpiresAt           *time.Time
	AllSessionsRevoked  bool
	AuditLogID          string
	Status              string
	Reason              string
}

type AdminUser struct {
	UserID      string
	Email       string
	Username    string
	Role        string
	CreatedAt   time.Time
	TotalEarnings float64
	Status      string
	KYCStatus   string
	LastLoginAt *time.Time
}

type BulkActionJob struct {
	JobID                   string
	Action                  string
	UserCount               int
	Status                  string
	CreatedAt               time.Time
	EstimatedCompletionTime time.Time
}

type CampaignPauseResult struct {
	CampaignID string
	Status     string
	PausedAt   time.Time
	AuditLogID string
}

type CampaignAdjustResult struct {
	CampaignID         string
	OldBudget          float64
	NewBudget          float64
	OldRatePer1kViews  float64
	NewRatePer1kViews  float64
	AdjustedAt         time.Time
	AuditLogID         string
}

type SubmissionOverride struct {
	SubmissionID string
	OldStatus    string
	NewStatus    string
	OverriddenAt time.Time
	AuditLogID   string
}

type FeatureFlag struct {
	FlagKey    string
	Enabled    bool
	Config     map[string]any
	UpdatedAt  time.Time
	UpdatedBy  string
}

type AnalyticsDashboard struct {
	DateRangeStart time.Time
	DateRangeEnd   time.Time
	TotalRevenue   float64
	UserGrowth     int
	CampaignCount  int
	FraudAlerts    int
}

type AuditLog struct {
	AuditID             string
	AdminID             string
	ActionType          string
	TargetResourceID    string
	TargetResourceType  string
	OldValue            map[string]any
	NewValue            map[string]any
	Reason              string
	PerformedAt         time.Time
	IPAddress           string
	SignatureHash       string
	IsVerified          bool
}

type AuditExport struct {
	ExportJobID          string
	Status               string
	FileURL              string
	CreatedAt            time.Time
	EstimatedCompletion  time.Time
}

type Repository interface {
	StartImpersonation(ctx context.Context, adminID string, userID string, reason string) (ImpersonationSession, error)
	EndImpersonation(ctx context.Context, impersonationID string) (ImpersonationSession, error)

	AdjustWallet(ctx context.Context, adminID string, userID string, amount float64, adjustmentType string, reason string) (WalletAdjustment, error)
	ListWalletHistory(ctx context.Context, userID string, cursor string, limit int) ([]WalletAdjustment, string, error)

	BanUser(ctx context.Context, adminID string, userID string, banType string, durationDays int, reason string) (UserBan, error)
	UnbanUser(ctx context.Context, adminID string, userID string, reason string) (UserBan, error)

	SearchUsers(ctx context.Context, query string, status string, cursor string, pageSize int) ([]AdminUser, string, int, error)
	CreateBulkActionJob(ctx context.Context, adminID string, userIDs []string, action string) (BulkActionJob, error)

	PauseCampaign(ctx context.Context, adminID string, campaignID string, reason string) (CampaignPauseResult, error)
	AdjustCampaign(ctx context.Context, adminID string, campaignID string, newBudget float64, newRate float64, reason string) (CampaignAdjustResult, error)

	OverrideSubmission(ctx context.Context, adminID string, submissionID string, newStatus string, reason string) (SubmissionOverride, error)

	ListFeatureFlags(ctx context.Context) ([]FeatureFlag, error)
	ToggleFeatureFlag(ctx context.Context, adminID string, flagKey string, enabled bool, reason string, config map[string]any) (FeatureFlag, bool, error)

	GetAnalyticsDashboard(ctx context.Context, start time.Time, end time.Time) (AnalyticsDashboard, error)

	ListAuditLogs(ctx context.Context, adminID string, actionType string, cursor string, pageSize int) ([]AuditLog, string, error)
	CreateAuditExport(ctx context.Context, format string, start time.Time, end time.Time, includeSignatures bool) (AuditExport, error)
}