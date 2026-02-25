package entities

import (
	"strings"
	"time"
)

type SubmissionStatus string

const (
	SubmissionStatusPending         SubmissionStatus = "pending"
	SubmissionStatusApproved        SubmissionStatus = "approved"
	SubmissionStatusRejected        SubmissionStatus = "rejected"
	SubmissionStatusFlagged         SubmissionStatus = "flagged"
	SubmissionStatusVerification    SubmissionStatus = "verification_period"
	SubmissionStatusViewLocked      SubmissionStatus = "view_locked"
	SubmissionStatusRewardEligible  SubmissionStatus = "reward_eligible"
	SubmissionStatusPaid            SubmissionStatus = "paid"
	SubmissionStatusDisputed        SubmissionStatus = "disputed"
	SubmissionStatusCancelled       SubmissionStatus = "cancelled"
	defaultMinimumLockedViewsAmount int              = 100
)

type Submission struct {
	SubmissionID          string
	CampaignID            string
	CreatorID             string
	Platform              string
	PostURL               string
	PostID                string
	CreatorPlatformHandle string
	Status                SubmissionStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ApprovedAt            *time.Time
	ApprovedByUserID      string
	ApprovalReason        string
	RejectedAt            *time.Time
	RejectionReason       string
	RejectionNotes        string
	ReportedCount         int
	VerificationStart     *time.Time
	VerificationWindowEnd *time.Time
	ViewsCount            int
	LockedViews           *int
	LockedAt              *time.Time
	LastViewSync          *time.Time
	CpvRate               float64
	GrossAmount           float64
	PlatformFee           float64
	NetAmount             float64
	Metadata              map[string]any
}

func (s Submission) ValidateCreate() bool {
	return strings.TrimSpace(s.CampaignID) != "" &&
		strings.TrimSpace(s.CreatorID) != "" &&
		IsSupportedPlatform(s.Platform) &&
		strings.TrimSpace(s.PostURL) != ""
}

type SubmissionReport struct {
	ReportID     string
	SubmissionID string
	ReportedByID string
	Reason       string
	Description  string
	ReportedAt   time.Time
}

type SubmissionAudit struct {
	AuditID      string
	SubmissionID string
	Action       string
	OldStatus    SubmissionStatus
	NewStatus    SubmissionStatus
	ActorID      string
	ActorRole    string
	ReasonCode   string
	ReasonNotes  string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

type SubmissionFlag struct {
	FlagID       string
	SubmissionID string
	FlagType     string
	Severity     string
	Details      map[string]any
	CreatedAt    time.Time
	IsResolved   bool
	ResolvedAt   *time.Time
}

type BulkSubmissionOperation struct {
	OperationID       string
	CampaignID        string
	OperationType     string
	SubmissionIDs     []string
	PerformedByUserID string
	SucceededCount    int
	FailedCount       int
	ReasonCode        string
	ReasonNotes       string
	CreatedAt         time.Time
}

type ViewSnapshot struct {
	SnapshotID          string
	SubmissionID        string
	ViewsCount          int
	EngagementEstimate  int
	PlatformMetricsJSON map[string]any
	SyncedAt            time.Time
	IsAnomaly           bool
	AnomalyReason       string
}

func IsSupportedPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "tiktok", "instagram", "youtube", "x", "twitter":
		return true
	default:
		return false
	}
}

func NormalizePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "tiktok":
		return "tiktok"
	case "instagram":
		return "instagram"
	case "youtube":
		return "youtube"
	case "twitter", "x":
		return "x"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func MinimumLockedViewsThreshold() int {
	return defaultMinimumLockedViewsAmount
}
