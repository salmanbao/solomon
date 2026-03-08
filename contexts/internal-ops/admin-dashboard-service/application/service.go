package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Service struct {
	Repo                   ports.Repository
	Idempotency            ports.IdempotencyStore
	AuthorizationClient    ports.AuthorizationClient
	ModerationClient       ports.ModerationClient
	AbusePreventionClient  ports.AbusePreventionClient
	FinanceClient          ports.FinanceClient
	BillingClient          ports.BillingClient
	RewardClient           ports.RewardClient
	AffiliateClient        ports.AffiliateClient
	PayoutClient           ports.PayoutClient
	ResolutionClient       ports.ResolutionClient
	ConsentClient          ports.ConsentClient
	PortabilityClient      ports.PortabilityClient
	RetentionClient        ports.RetentionClient
	LegalClient            ports.LegalClient
	SupportClient          ports.SupportClient
	EditorWorkflowClient   ports.EditorWorkflowClient
	ClippingWorkflowClient ports.ClippingWorkflowClient
	AutoClippingClient     ports.AutoClippingClient
	DeveloperPortalClient  ports.DeveloperPortalClient
	IntegrationHubClient   ports.IntegrationHubClient
	WebhookManagerClient   ports.WebhookManagerClient
	DataMigrationClient    ports.DataMigrationClient
	Clock                  ports.Clock
	IdempotencyTTL         time.Duration
}

type RecordActionInput struct {
	ActorID       string
	Action        string
	TargetID      string
	Justification string
	SourceIP      string
	CorrelationID string
}

type GrantIdentityRoleInput struct {
	ActorID       string
	UserID        string
	RoleID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type GrantIdentityRoleResult struct {
	AssignmentID        string
	UserID              string
	RoleID              string
	OwnerAuditLogID     string
	ControlPlaneAuditID string
	OccurredAt          time.Time
}

type ModerateSubmissionInput struct {
	ActorID       string
	SubmissionID  string
	CampaignID    string
	Action        string
	Reason        string
	Notes         string
	SourceIP      string
	CorrelationID string
}

type ModerateSubmissionResult struct {
	DecisionID          string
	SubmissionID        string
	CampaignID          string
	ModeratorID         string
	Action              string
	Reason              string
	Notes               string
	QueueStatus         string
	CreatedAt           time.Time
	ControlPlaneAuditID string
}

type ReleaseAbuseLockoutInput struct {
	ActorID       string
	UserID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type ReleaseAbuseLockoutResult struct {
	ThreatID            string
	UserID              string
	Status              string
	ReleasedAt          time.Time
	OwnerAuditLogID     string
	ControlPlaneAuditID string
}

type CreateFinanceRefundInput struct {
	ActorID       string
	TransactionID string
	UserID        string
	Amount        float64
	Reason        string
	SourceIP      string
	CorrelationID string
}

type CreateFinanceRefundResult struct {
	RefundID            string
	TransactionID       string
	UserID              string
	Amount              float64
	Reason              string
	CreatedAt           time.Time
	ControlPlaneAuditID string
}

type CreateBillingRefundInput struct {
	ActorID       string
	InvoiceID     string
	LineItemID    string
	Amount        float64
	Reason        string
	SourceIP      string
	CorrelationID string
}

type CreateBillingRefundResult struct {
	RefundID            string
	InvoiceID           string
	LineItemID          string
	Amount              float64
	Reason              string
	ProcessedAt         time.Time
	ControlPlaneAuditID string
}

type RecalculateRewardInput struct {
	ActorID                 string
	UserID                  string
	SubmissionID            string
	CampaignID              string
	LockedViews             int64
	RatePer1K               float64
	FraudScore              float64
	VerificationCompletedAt time.Time
	Reason                  string
	SourceIP                string
	CorrelationID           string
}

type RecalculateRewardResult struct {
	SubmissionID        string
	UserID              string
	CampaignID          string
	Status              string
	NetAmount           float64
	RolloverTotal       float64
	CalculatedAt        time.Time
	EligibleAt          *time.Time
	ControlPlaneAuditID string
}

type SuspendAffiliateInput struct {
	ActorID       string
	AffiliateID   string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type SuspendAffiliateResult struct {
	AffiliateID         string
	Status              string
	UpdatedAt           time.Time
	ControlPlaneAuditID string
}

type CreateAffiliateAttributionInput struct {
	ActorID       string
	AffiliateID   string
	ClickID       string
	OrderID       string
	ConversionID  string
	Amount        float64
	Currency      string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type CreateAffiliateAttributionResult struct {
	AttributionID       string
	AffiliateID         string
	OrderID             string
	Amount              float64
	AttributedAt        time.Time
	ControlPlaneAuditID string
}

type RetryPayoutInput struct {
	ActorID       string
	PayoutID      string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type RetryPayoutResult struct {
	PayoutID            string
	UserID              string
	Status              string
	FailureReason       string
	ProcessedAt         time.Time
	ControlPlaneAuditID string
}

type ResolveDisputeInput struct {
	ActorID       string
	DisputeID     string
	Action        string
	Reason        string
	Notes         string
	RefundAmount  float64
	SourceIP      string
	CorrelationID string
}

type ResolveDisputeResult struct {
	DisputeID           string
	Status              string
	ResolutionType      string
	RefundAmount        float64
	ProcessedAt         time.Time
	ControlPlaneAuditID string
}

type GetConsentInput struct {
	ActorID string
	UserID  string
}

type GetConsentResult struct {
	UserID        string
	Status        string
	Preferences   map[string]bool
	LastUpdated   time.Time
	LastUpdatedBy string
}

type UpdateConsentInput struct {
	ActorID       string
	UserID        string
	Preferences   map[string]bool
	Reason        string
	SourceIP      string
	CorrelationID string
}

type UpdateConsentResult struct {
	UserID              string
	Status              string
	UpdatedAt           time.Time
	ControlPlaneAuditID string
}

type WithdrawConsentInput struct {
	ActorID       string
	UserID        string
	Category      string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type WithdrawConsentResult struct {
	UserID              string
	Status              string
	UpdatedAt           time.Time
	ControlPlaneAuditID string
}

type StartDataExportInput struct {
	ActorID       string
	UserID        string
	Format        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type StartDataExportResult struct {
	RequestID           string
	UserID              string
	RequestType         string
	Format              string
	Status              string
	RequestedAt         time.Time
	CompletedAt         *time.Time
	DownloadURL         string
	ControlPlaneAuditID string
}

type GetDataExportInput struct {
	ActorID       string
	RequestID     string
	SourceIP      string
	CorrelationID string
}

type GetDataExportResult struct {
	RequestID   string
	UserID      string
	RequestType string
	Format      string
	Status      string
	Reason      string
	RequestedAt time.Time
	CompletedAt *time.Time
	DownloadURL string
}

type RequestDeletionInput struct {
	ActorID       string
	UserID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type RequestDeletionResult struct {
	RequestID           string
	UserID              string
	Status              string
	Reason              string
	RequestedAt         time.Time
	CompletedAt         *time.Time
	ControlPlaneAuditID string
}

type CreateRetentionLegalHoldInput struct {
	ActorID       string
	EntityID      string
	DataType      string
	Reason        string
	ExpiresAt     *time.Time
	SourceIP      string
	CorrelationID string
}

type CreateRetentionLegalHoldResult struct {
	HoldID              string
	EntityID            string
	DataType            string
	Status              string
	CreatedAt           time.Time
	ExpiresAt           *time.Time
	ControlPlaneAuditID string
}

type CheckLegalHoldInput struct {
	ActorID       string
	EntityType    string
	EntityID      string
	SourceIP      string
	CorrelationID string
}

type CheckLegalHoldResult struct {
	EntityType          string
	EntityID            string
	Held                bool
	HoldID              string
	ControlPlaneAuditID string
}

type ReleaseLegalHoldInput struct {
	ActorID       string
	HoldID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type ReleaseLegalHoldResult struct {
	HoldID              string
	EntityType          string
	EntityID            string
	Status              string
	CreatedAt           time.Time
	ReleasedAt          *time.Time
	ControlPlaneAuditID string
}

type RunComplianceScanInput struct {
	ActorID       string
	ReportType    string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type RunComplianceScanResult struct {
	ReportID            string
	ReportType          string
	Status              string
	FindingsCount       int
	DownloadURL         string
	CreatedAt           time.Time
	ControlPlaneAuditID string
}

type GetSupportTicketInput struct {
	ActorID  string
	TicketID string
}

type SearchSupportTicketsInput struct {
	ActorID    string
	Query      string
	Status     string
	Category   string
	AssignedTo string
	Limit      int
}

type SupportTicketResult struct {
	TicketID         string
	UserID           string
	Subject          string
	Description      string
	Category         string
	Priority         string
	Status           string
	SubStatus        string
	AssignedAgentID  string
	SLAResponseDueAt time.Time
	LastActivityAt   time.Time
	UpdatedAt        time.Time
}

type AssignSupportTicketInput struct {
	ActorID       string
	TicketID      string
	AgentID       string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type AssignSupportTicketResult struct {
	SupportTicketResult
	ControlPlaneAuditID string
}

type UpdateSupportTicketInput struct {
	ActorID       string
	TicketID      string
	Status        string
	SubStatus     string
	Priority      string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type UpdateSupportTicketResult struct {
	SupportTicketResult
	ControlPlaneAuditID string
}

type SaveEditorCampaignInput struct {
	ActorID       string
	EditorID      string
	CampaignID    string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type SaveEditorCampaignResult struct {
	CampaignID          string
	Saved               bool
	SavedAt             *time.Time
	ControlPlaneAuditID string
}

type RequestClippingExportInput struct {
	ActorID       string
	UserID        string
	ProjectID     string
	Format        string
	Resolution    string
	FPS           int
	Bitrate       string
	CampaignID    string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type RequestClippingExportResult struct {
	ExportID            string
	ProjectID           string
	Status              string
	ProgressPercent     int
	OutputURL           string
	ProviderJobID       string
	CreatedAt           time.Time
	CompletedAt         *time.Time
	ControlPlaneAuditID string
}

type DeployAutoClippingModelInput struct {
	ActorID          string
	ModelName        string
	VersionTag       string
	ModelArtifactKey string
	CanaryPercentage int
	Description      string
	Reason           string
	SourceIP         string
	CorrelationID    string
}

type DeployAutoClippingModelResult struct {
	ModelVersionID      string
	DeploymentStatus    string
	DeployedAt          time.Time
	Message             string
	ControlPlaneAuditID string
}

type RotateIntegrationKeyInput struct {
	ActorID       string
	KeyID         string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type RotateIntegrationKeyResult struct {
	RotationID          string
	DeveloperID         string
	OldKeyID            string
	NewKeyID            string
	CreatedAt           time.Time
	ControlPlaneAuditID string
}

type TestIntegrationWorkflowInput struct {
	ActorID       string
	WorkflowID    string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type TestIntegrationWorkflowResult struct {
	ExecutionID         string
	WorkflowID          string
	Status              string
	TestRun             bool
	StartedAt           time.Time
	ControlPlaneAuditID string
}

type ReplayWebhookInput struct {
	ActorID       string
	WebhookID     string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type ReplayWebhookResult struct {
	DeliveryID          string
	WebhookID           string
	Status              string
	HTTPStatus          int
	LatencyMS           int64
	Timestamp           time.Time
	ControlPlaneAuditID string
}

type DisableWebhookInput struct {
	ActorID       string
	WebhookID     string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type DisableWebhookResult struct {
	WebhookID           string
	Status              string
	UpdatedAt           time.Time
	ControlPlaneAuditID string
}

type GetWebhookDeliveriesInput struct {
	ActorID   string
	WebhookID string
	Limit     int
}

type WebhookDeliveryResult struct {
	DeliveryID      string
	WebhookID       string
	OriginalEventID string
	OriginalType    string
	HTTPStatus      int
	LatencyMS       int64
	RetryCount      int
	DeliveredAt     time.Time
	IsTest          bool
	Success         bool
}

type GetWebhookAnalyticsInput struct {
	ActorID   string
	WebhookID string
}

type WebhookAnalyticsMetrics struct {
	Total      int64
	Success    int64
	Failed     int64
	AvgLatency float64
}

type WebhookAnalyticsResult struct {
	TotalDeliveries      int64
	SuccessfulDeliveries int64
	FailedDeliveries     int64
	SuccessRate          float64
	AvgLatencyMS         float64
	P95LatencyMS         float64
	P99LatencyMS         float64
	ByEventType          map[string]WebhookAnalyticsMetrics
}

type CreateMigrationPlanInput struct {
	ActorID       string
	ServiceName   string
	Environment   string
	Version       string
	Plan          map[string]interface{}
	DryRun        bool
	RiskLevel     string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type CreateMigrationPlanResult struct {
	PlanID              string
	ServiceName         string
	Environment         string
	Version             string
	Plan                map[string]interface{}
	Status              string
	DryRun              bool
	RiskLevel           string
	StagingValidated    bool
	BackupRequired      bool
	CreatedBy           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ControlPlaneAuditID string
}

type ListMigrationPlansInput struct {
	ActorID string
}

type MigrationPlanResult struct {
	PlanID           string
	ServiceName      string
	Environment      string
	Version          string
	Plan             map[string]interface{}
	Status           string
	DryRun           bool
	RiskLevel        string
	StagingValidated bool
	BackupRequired   bool
	CreatedBy        string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type StartMigrationRunInput struct {
	ActorID       string
	PlanID        string
	Reason        string
	SourceIP      string
	CorrelationID string
}

type StartMigrationRunResult struct {
	RunID               string
	PlanID              string
	Status              string
	OperatorID          string
	SnapshotCreated     bool
	RollbackAvailable   bool
	ValidationStatus    string
	BackfillJobID       string
	StartedAt           time.Time
	CompletedAt         time.Time
	ControlPlaneAuditID string
}

func (s Service) RecordAdminAction(ctx context.Context, idempotencyKey string, input RecordActionInput) (ports.AuditLog, error) {
	if strings.TrimSpace(input.ActorID) == "" {
		return ports.AuditLog{}, domainerrors.ErrUnauthorized
	}
	if strings.TrimSpace(input.Action) == "" || strings.TrimSpace(input.Justification) == "" {
		return ports.AuditLog{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ports.AuditLog{}, domainerrors.ErrIdempotencyRequired
	}

	now := s.now()
	requestHash := hashPayload(input)
	var output ports.AuditLog
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			logRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       strings.TrimSpace(input.ActorID),
				Action:        strings.TrimSpace(input.Action),
				TargetID:      strings.TrimSpace(input.TargetID),
				Justification: strings.TrimSpace(input.Justification),
				OccurredAt:    now,
				SourceIP:      strings.TrimSpace(input.SourceIP),
				CorrelationID: strings.TrimSpace(input.CorrelationID),
			}
			if err := s.Repo.AppendAuditLog(ctx, logRow); err != nil {
				return nil, err
			}
			return json.Marshal(logRow)
		},
	); err != nil {
		return ports.AuditLog{}, err
	}
	return output, nil
}

func (s Service) GrantIdentityRole(
	ctx context.Context,
	idempotencyKey string,
	input GrantIdentityRoleInput,
) (GrantIdentityRoleResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.RoleID = strings.TrimSpace(input.RoleID)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.ActorID == "" {
		return GrantIdentityRoleResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.RoleID == "" || input.Reason == "" {
		return GrantIdentityRoleResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return GrantIdentityRoleResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.AuthorizationClient == nil {
		return GrantIdentityRoleResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output GrantIdentityRoleResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.AuthorizationClient.GrantRole(
				ctx,
				input.ActorID,
				input.UserID,
				input.RoleID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "grant_role"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.identity.role.granted",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      strings.TrimSpace(input.SourceIP),
				CorrelationID: strings.TrimSpace(input.CorrelationID),
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(GrantIdentityRoleResult{
				AssignmentID:        ownerResult.AssignmentID,
				UserID:              ownerResult.UserID,
				RoleID:              ownerResult.RoleID,
				OwnerAuditLogID:     ownerResult.AuditLogID,
				ControlPlaneAuditID: auditRow.AuditID,
				OccurredAt:          now,
			})
		},
	); err != nil {
		return GrantIdentityRoleResult{}, err
	}
	return output, nil
}

func (s Service) ModerateSubmission(
	ctx context.Context,
	idempotencyKey string,
	input ModerateSubmissionInput,
) (ModerateSubmissionResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.SubmissionID = strings.TrimSpace(input.SubmissionID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Action = strings.TrimSpace(strings.ToLower(input.Action))
	input.Reason = strings.TrimSpace(input.Reason)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.ActorID == "" {
		return ModerateSubmissionResult{}, domainerrors.ErrUnauthorized
	}
	if input.SubmissionID == "" || input.CampaignID == "" || input.Action == "" {
		return ModerateSubmissionResult{}, domainerrors.ErrInvalidInput
	}
	if input.Action != "approve" && input.Action != "reject" {
		return ModerateSubmissionResult{}, domainerrors.ErrUnsupportedAction
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ModerateSubmissionResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.ModerationClient == nil {
		return ModerateSubmissionResult{}, domainerrors.ErrDependencyUnavailable
	}
	if input.Action == "reject" && input.Reason == "" {
		return ModerateSubmissionResult{}, domainerrors.ErrInvalidInput
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ModerateSubmissionResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			var ownerResult ports.ModerationDecisionResult
			var err error
			ownerKey := childIdempotencyKey(idempotencyKey, "moderate_submission")
			switch input.Action {
			case "approve":
				ownerResult, err = s.ModerationClient.ApproveSubmission(
					ctx,
					input.ActorID,
					input.SubmissionID,
					input.CampaignID,
					input.Reason,
					input.Notes,
					ownerKey,
				)
			case "reject":
				ownerResult, err = s.ModerationClient.RejectSubmission(
					ctx,
					input.ActorID,
					input.SubmissionID,
					input.CampaignID,
					input.Reason,
					input.Notes,
					ownerKey,
				)
			default:
				err = domainerrors.ErrUnsupportedAction
			}
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.submission.moderated",
				TargetID:      input.SubmissionID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      strings.TrimSpace(input.SourceIP),
				CorrelationID: strings.TrimSpace(input.CorrelationID),
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ModerateSubmissionResult{
				DecisionID:          ownerResult.DecisionID,
				SubmissionID:        ownerResult.SubmissionID,
				CampaignID:          ownerResult.CampaignID,
				ModeratorID:         ownerResult.ModeratorID,
				Action:              ownerResult.Action,
				Reason:              ownerResult.Reason,
				Notes:               ownerResult.Notes,
				QueueStatus:         ownerResult.QueueStatus,
				CreatedAt:           ownerResult.CreatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ModerateSubmissionResult{}, err
	}
	return output, nil
}

func (s Service) ReleaseAbuseLockout(
	ctx context.Context,
	idempotencyKey string,
	input ReleaseAbuseLockoutInput,
) (ReleaseAbuseLockoutResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return ReleaseAbuseLockoutResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.Reason == "" {
		return ReleaseAbuseLockoutResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ReleaseAbuseLockoutResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.AbusePreventionClient == nil {
		return ReleaseAbuseLockoutResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ReleaseAbuseLockoutResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.AbusePreventionClient.ReleaseLockout(
				ctx,
				input.ActorID,
				input.UserID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "abuse_release_lockout"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.abuse.lockout.released",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ReleaseAbuseLockoutResult{
				ThreatID:            ownerResult.ThreatID,
				UserID:              ownerResult.UserID,
				Status:              ownerResult.Status,
				ReleasedAt:          ownerResult.ReleasedAt,
				OwnerAuditLogID:     ownerResult.OwnerAuditLogID,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ReleaseAbuseLockoutResult{}, err
	}
	return output, nil
}

func (s Service) CreateFinanceRefund(
	ctx context.Context,
	idempotencyKey string,
	input CreateFinanceRefundInput,
) (CreateFinanceRefundResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.TransactionID = strings.TrimSpace(input.TransactionID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CreateFinanceRefundResult{}, domainerrors.ErrUnauthorized
	}
	if input.TransactionID == "" || input.UserID == "" || input.Reason == "" || input.Amount <= 0 {
		return CreateFinanceRefundResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CreateFinanceRefundResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.FinanceClient == nil {
		return CreateFinanceRefundResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output CreateFinanceRefundResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.FinanceClient.CreateRefund(
				ctx,
				input.ActorID,
				input.TransactionID,
				input.UserID,
				input.Amount,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "finance_refund"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.finance.refund.created",
				TargetID:      input.TransactionID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(CreateFinanceRefundResult{
				RefundID:            ownerResult.RefundID,
				TransactionID:       ownerResult.TransactionID,
				UserID:              ownerResult.UserID,
				Amount:              ownerResult.Amount,
				Reason:              ownerResult.Reason,
				CreatedAt:           ownerResult.CreatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return CreateFinanceRefundResult{}, err
	}
	return output, nil
}

func (s Service) CreateBillingRefund(
	ctx context.Context,
	idempotencyKey string,
	input CreateBillingRefundInput,
) (CreateBillingRefundResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.InvoiceID = strings.TrimSpace(input.InvoiceID)
	input.LineItemID = strings.TrimSpace(input.LineItemID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CreateBillingRefundResult{}, domainerrors.ErrUnauthorized
	}
	if input.InvoiceID == "" || input.LineItemID == "" || input.Amount <= 0 || input.Reason == "" {
		return CreateBillingRefundResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CreateBillingRefundResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.BillingClient == nil {
		return CreateBillingRefundResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output CreateBillingRefundResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.BillingClient.CreateInvoiceRefund(
				ctx,
				input.ActorID,
				input.InvoiceID,
				input.LineItemID,
				input.Amount,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "billing_refund"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.billing.refund.created",
				TargetID:      input.InvoiceID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(CreateBillingRefundResult{
				RefundID:            ownerResult.RefundID,
				InvoiceID:           ownerResult.InvoiceID,
				LineItemID:          ownerResult.LineItemID,
				Amount:              ownerResult.Amount,
				Reason:              ownerResult.Reason,
				ProcessedAt:         ownerResult.ProcessedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return CreateBillingRefundResult{}, err
	}
	return output, nil
}

func (s Service) RecalculateReward(
	ctx context.Context,
	idempotencyKey string,
	input RecalculateRewardInput,
) (RecalculateRewardResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.SubmissionID = strings.TrimSpace(input.SubmissionID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RecalculateRewardResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.SubmissionID == "" || input.CampaignID == "" || input.Reason == "" {
		return RecalculateRewardResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RecalculateRewardResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.RewardClient == nil {
		return RecalculateRewardResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RecalculateRewardResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.RewardClient.RecalculateReward(
				ctx,
				input.ActorID,
				input.UserID,
				input.SubmissionID,
				input.CampaignID,
				input.LockedViews,
				input.RatePer1K,
				input.FraudScore,
				input.VerificationCompletedAt,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "reward_recalculate"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.reward.recalculated",
				TargetID:      input.SubmissionID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RecalculateRewardResult{
				SubmissionID:        ownerResult.SubmissionID,
				UserID:              ownerResult.UserID,
				CampaignID:          ownerResult.CampaignID,
				Status:              ownerResult.Status,
				NetAmount:           ownerResult.NetAmount,
				RolloverTotal:       ownerResult.RolloverTotal,
				CalculatedAt:        ownerResult.CalculatedAt,
				EligibleAt:          ownerResult.EligibleAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RecalculateRewardResult{}, err
	}
	return output, nil
}

func (s Service) SuspendAffiliate(
	ctx context.Context,
	idempotencyKey string,
	input SuspendAffiliateInput,
) (SuspendAffiliateResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.AffiliateID = strings.TrimSpace(input.AffiliateID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return SuspendAffiliateResult{}, domainerrors.ErrUnauthorized
	}
	if input.AffiliateID == "" || input.Reason == "" {
		return SuspendAffiliateResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return SuspendAffiliateResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.AffiliateClient == nil {
		return SuspendAffiliateResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output SuspendAffiliateResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.AffiliateClient.SuspendAffiliate(
				ctx,
				input.ActorID,
				input.AffiliateID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "affiliate_suspend"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.affiliate.suspended",
				TargetID:      input.AffiliateID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(SuspendAffiliateResult{
				AffiliateID:         ownerResult.AffiliateID,
				Status:              ownerResult.Status,
				UpdatedAt:           ownerResult.UpdatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return SuspendAffiliateResult{}, err
	}
	return output, nil
}

func (s Service) CreateAffiliateAttribution(
	ctx context.Context,
	idempotencyKey string,
	input CreateAffiliateAttributionInput,
) (CreateAffiliateAttributionResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.AffiliateID = strings.TrimSpace(input.AffiliateID)
	input.ClickID = strings.TrimSpace(input.ClickID)
	input.OrderID = strings.TrimSpace(input.OrderID)
	input.ConversionID = strings.TrimSpace(input.ConversionID)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CreateAffiliateAttributionResult{}, domainerrors.ErrUnauthorized
	}
	if input.AffiliateID == "" || input.OrderID == "" || input.ConversionID == "" || input.Amount <= 0 || input.Reason == "" {
		return CreateAffiliateAttributionResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CreateAffiliateAttributionResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.AffiliateClient == nil {
		return CreateAffiliateAttributionResult{}, domainerrors.ErrDependencyUnavailable
	}
	if input.Currency == "" {
		input.Currency = "USD"
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output CreateAffiliateAttributionResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.AffiliateClient.CreateAttribution(
				ctx,
				input.ActorID,
				input.AffiliateID,
				input.ClickID,
				input.OrderID,
				input.ConversionID,
				input.Amount,
				input.Currency,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "affiliate_attribution"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.affiliate.attribution.created",
				TargetID:      input.AffiliateID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(CreateAffiliateAttributionResult{
				AttributionID:       ownerResult.AttributionID,
				AffiliateID:         ownerResult.AffiliateID,
				OrderID:             ownerResult.OrderID,
				Amount:              ownerResult.Amount,
				AttributedAt:        ownerResult.AttributedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return CreateAffiliateAttributionResult{}, err
	}
	return output, nil
}

func (s Service) RetryPayout(
	ctx context.Context,
	idempotencyKey string,
	input RetryPayoutInput,
) (RetryPayoutResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.PayoutID = strings.TrimSpace(input.PayoutID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RetryPayoutResult{}, domainerrors.ErrUnauthorized
	}
	if input.PayoutID == "" || input.Reason == "" {
		return RetryPayoutResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RetryPayoutResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.PayoutClient == nil {
		return RetryPayoutResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RetryPayoutResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.PayoutClient.RetryFailedPayout(
				ctx,
				input.ActorID,
				input.PayoutID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "payout_retry"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.payout.retry.requested",
				TargetID:      input.PayoutID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RetryPayoutResult{
				PayoutID:            ownerResult.PayoutID,
				UserID:              ownerResult.UserID,
				Status:              ownerResult.Status,
				FailureReason:       ownerResult.FailureReason,
				ProcessedAt:         ownerResult.ProcessedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RetryPayoutResult{}, err
	}
	return output, nil
}

func (s Service) ResolveDispute(
	ctx context.Context,
	idempotencyKey string,
	input ResolveDisputeInput,
) (ResolveDisputeResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.DisputeID = strings.TrimSpace(input.DisputeID)
	input.Action = strings.TrimSpace(strings.ToLower(input.Action))
	input.Reason = strings.TrimSpace(input.Reason)
	input.Notes = strings.TrimSpace(input.Notes)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return ResolveDisputeResult{}, domainerrors.ErrUnauthorized
	}
	if input.DisputeID == "" || input.Action == "" || input.Reason == "" {
		return ResolveDisputeResult{}, domainerrors.ErrInvalidInput
	}
	if input.Action != "resolve" && input.Action != "reopen" {
		return ResolveDisputeResult{}, domainerrors.ErrUnsupportedAction
	}
	if input.Action == "resolve" && input.RefundAmount <= 0 {
		return ResolveDisputeResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ResolveDisputeResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.ResolutionClient == nil {
		return ResolveDisputeResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ResolveDisputeResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.ResolutionClient.ResolveDispute(
				ctx,
				input.ActorID,
				input.DisputeID,
				input.Action,
				input.Reason,
				input.Notes,
				input.RefundAmount,
				childIdempotencyKey(idempotencyKey, "dispute_resolution"),
			)
			if err != nil {
				return nil, err
			}
			auditAction := "admin.dispute.status.changed"
			if input.Action == "resolve" {
				auditAction = "admin.dispute.resolution.applied"
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        auditAction,
				TargetID:      input.DisputeID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ResolveDisputeResult{
				DisputeID:           ownerResult.DisputeID,
				Status:              ownerResult.Status,
				ResolutionType:      ownerResult.ResolutionType,
				RefundAmount:        ownerResult.RefundAmount,
				ProcessedAt:         ownerResult.ProcessedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ResolveDisputeResult{}, err
	}
	return output, nil
}

func (s Service) GetConsent(ctx context.Context, input GetConsentInput) (GetConsentResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	if input.ActorID == "" {
		return GetConsentResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" {
		return GetConsentResult{}, domainerrors.ErrInvalidInput
	}
	if s.ConsentClient == nil {
		return GetConsentResult{}, domainerrors.ErrDependencyUnavailable
	}

	ownerResult, err := s.ConsentClient.GetConsent(ctx, input.ActorID, input.UserID)
	if err != nil {
		return GetConsentResult{}, err
	}
	return GetConsentResult{
		UserID:        ownerResult.UserID,
		Status:        ownerResult.Status,
		Preferences:   ownerResult.Preferences,
		LastUpdated:   ownerResult.LastUpdated,
		LastUpdatedBy: ownerResult.LastUpdatedBy,
	}, nil
}

func (s Service) UpdateConsent(
	ctx context.Context,
	idempotencyKey string,
	input UpdateConsentInput,
) (UpdateConsentResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return UpdateConsentResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.Reason == "" || len(input.Preferences) == 0 {
		return UpdateConsentResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return UpdateConsentResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.ConsentClient == nil {
		return UpdateConsentResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output UpdateConsentResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.ConsentClient.UpdateConsent(
				ctx,
				input.ActorID,
				input.UserID,
				input.Preferences,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "consent_update"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.consent.updated",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(UpdateConsentResult{
				UserID:              ownerResult.UserID,
				Status:              ownerResult.Status,
				UpdatedAt:           ownerResult.UpdatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return UpdateConsentResult{}, err
	}
	return output, nil
}

func (s Service) WithdrawConsent(
	ctx context.Context,
	idempotencyKey string,
	input WithdrawConsentInput,
) (WithdrawConsentResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Category = strings.TrimSpace(input.Category)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return WithdrawConsentResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.Reason == "" {
		return WithdrawConsentResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return WithdrawConsentResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.ConsentClient == nil {
		return WithdrawConsentResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output WithdrawConsentResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.ConsentClient.WithdrawConsent(
				ctx,
				input.ActorID,
				input.UserID,
				input.Category,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "consent_withdraw"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.consent.withdrawn",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(WithdrawConsentResult{
				UserID:              ownerResult.UserID,
				Status:              ownerResult.Status,
				UpdatedAt:           ownerResult.UpdatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return WithdrawConsentResult{}, err
	}
	return output, nil
}

func (s Service) StartDataExport(
	ctx context.Context,
	idempotencyKey string,
	input StartDataExportInput,
) (StartDataExportResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Format = strings.ToLower(strings.TrimSpace(input.Format))
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return StartDataExportResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.Reason == "" {
		return StartDataExportResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return StartDataExportResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.PortabilityClient == nil {
		return StartDataExportResult{}, domainerrors.ErrDependencyUnavailable
	}
	if input.Format == "" {
		input.Format = "json"
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output StartDataExportResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.PortabilityClient.CreateExport(
				ctx,
				input.ActorID,
				input.UserID,
				input.Format,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "portability_export"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.export.started",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(StartDataExportResult{
				RequestID:           ownerResult.RequestID,
				UserID:              ownerResult.UserID,
				RequestType:         ownerResult.RequestType,
				Format:              ownerResult.Format,
				Status:              ownerResult.Status,
				RequestedAt:         ownerResult.RequestedAt,
				CompletedAt:         ownerResult.CompletedAt,
				DownloadURL:         ownerResult.DownloadURL,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return StartDataExportResult{}, err
	}
	return output, nil
}

func (s Service) GetDataExport(ctx context.Context, input GetDataExportInput) (GetDataExportResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.RequestID = strings.TrimSpace(input.RequestID)
	if input.ActorID == "" {
		return GetDataExportResult{}, domainerrors.ErrUnauthorized
	}
	if input.RequestID == "" {
		return GetDataExportResult{}, domainerrors.ErrInvalidInput
	}
	if s.PortabilityClient == nil {
		return GetDataExportResult{}, domainerrors.ErrDependencyUnavailable
	}

	ownerResult, err := s.PortabilityClient.GetExport(ctx, input.ActorID, input.RequestID)
	if err != nil {
		return GetDataExportResult{}, err
	}
	return GetDataExportResult{
		RequestID:   ownerResult.RequestID,
		UserID:      ownerResult.UserID,
		RequestType: ownerResult.RequestType,
		Format:      ownerResult.Format,
		Status:      ownerResult.Status,
		Reason:      ownerResult.Reason,
		RequestedAt: ownerResult.RequestedAt,
		CompletedAt: ownerResult.CompletedAt,
		DownloadURL: ownerResult.DownloadURL,
	}, nil
}

func (s Service) RequestDeletion(
	ctx context.Context,
	idempotencyKey string,
	input RequestDeletionInput,
) (RequestDeletionResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RequestDeletionResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.Reason == "" {
		return RequestDeletionResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RequestDeletionResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.PortabilityClient == nil {
		return RequestDeletionResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RequestDeletionResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.PortabilityClient.CreateEraseRequest(
				ctx,
				input.ActorID,
				input.UserID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "portability_erase"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.deletion.started",
				TargetID:      input.UserID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RequestDeletionResult{
				RequestID:           ownerResult.RequestID,
				UserID:              ownerResult.UserID,
				Status:              ownerResult.Status,
				Reason:              ownerResult.Reason,
				RequestedAt:         ownerResult.RequestedAt,
				CompletedAt:         ownerResult.CompletedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RequestDeletionResult{}, err
	}
	return output, nil
}

func (s Service) CreateRetentionLegalHold(
	ctx context.Context,
	idempotencyKey string,
	input CreateRetentionLegalHoldInput,
) (CreateRetentionLegalHoldResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.EntityID = strings.TrimSpace(input.EntityID)
	input.DataType = strings.TrimSpace(input.DataType)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CreateRetentionLegalHoldResult{}, domainerrors.ErrUnauthorized
	}
	if input.EntityID == "" || input.DataType == "" || input.Reason == "" {
		return CreateRetentionLegalHoldResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CreateRetentionLegalHoldResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.RetentionClient == nil {
		return CreateRetentionLegalHoldResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output CreateRetentionLegalHoldResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.RetentionClient.CreateLegalHold(
				ctx,
				input.ActorID,
				input.EntityID,
				input.DataType,
				input.Reason,
				input.ExpiresAt,
				childIdempotencyKey(idempotencyKey, "retention_legal_hold"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.retention.legal_hold.created",
				TargetID:      ownerResult.HoldID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(CreateRetentionLegalHoldResult{
				HoldID:              ownerResult.HoldID,
				EntityID:            ownerResult.EntityID,
				DataType:            ownerResult.DataType,
				Status:              ownerResult.Status,
				CreatedAt:           ownerResult.CreatedAt,
				ExpiresAt:           ownerResult.ExpiresAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return CreateRetentionLegalHoldResult{}, err
	}
	return output, nil
}

func (s Service) CheckLegalHold(ctx context.Context, input CheckLegalHoldInput) (CheckLegalHoldResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.EntityID = strings.TrimSpace(input.EntityID)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CheckLegalHoldResult{}, domainerrors.ErrUnauthorized
	}
	if input.EntityType == "" || input.EntityID == "" {
		return CheckLegalHoldResult{}, domainerrors.ErrInvalidInput
	}
	if s.LegalClient == nil {
		return CheckLegalHoldResult{}, domainerrors.ErrDependencyUnavailable
	}

	ownerResult, err := s.LegalClient.CheckHold(ctx, input.ActorID, input.EntityType, input.EntityID)
	if err != nil {
		return CheckLegalHoldResult{}, err
	}
	now := s.now()
	auditRow := ports.AuditLog{
		AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
		ActorID:       input.ActorID,
		Action:        "admin.compliance.legal_hold.checked",
		TargetID:      input.EntityID,
		Justification: "legal hold check",
		OccurredAt:    now,
		SourceIP:      input.SourceIP,
		CorrelationID: input.CorrelationID,
	}
	if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
		return CheckLegalHoldResult{}, err
	}
	return CheckLegalHoldResult{
		EntityType:          ownerResult.EntityType,
		EntityID:            ownerResult.EntityID,
		Held:                ownerResult.Held,
		HoldID:              ownerResult.HoldID,
		ControlPlaneAuditID: auditRow.AuditID,
	}, nil
}

func (s Service) ReleaseLegalHold(
	ctx context.Context,
	idempotencyKey string,
	input ReleaseLegalHoldInput,
) (ReleaseLegalHoldResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.HoldID = strings.TrimSpace(input.HoldID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return ReleaseLegalHoldResult{}, domainerrors.ErrUnauthorized
	}
	if input.HoldID == "" || input.Reason == "" {
		return ReleaseLegalHoldResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ReleaseLegalHoldResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.LegalClient == nil {
		return ReleaseLegalHoldResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ReleaseLegalHoldResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.LegalClient.ReleaseHold(
				ctx,
				input.ActorID,
				input.HoldID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "legal_hold_release"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.legal_hold.released",
				TargetID:      input.HoldID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ReleaseLegalHoldResult{
				HoldID:              ownerResult.HoldID,
				EntityType:          ownerResult.EntityType,
				EntityID:            ownerResult.EntityID,
				Status:              ownerResult.Status,
				CreatedAt:           ownerResult.CreatedAt,
				ReleasedAt:          ownerResult.ReleasedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ReleaseLegalHoldResult{}, err
	}
	return output, nil
}

func (s Service) RunComplianceScan(
	ctx context.Context,
	idempotencyKey string,
	input RunComplianceScanInput,
) (RunComplianceScanResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.ReportType = strings.TrimSpace(input.ReportType)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RunComplianceScanResult{}, domainerrors.ErrUnauthorized
	}
	if input.Reason == "" {
		return RunComplianceScanResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RunComplianceScanResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.LegalClient == nil {
		return RunComplianceScanResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RunComplianceScanResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.LegalClient.RunComplianceScan(
				ctx,
				input.ActorID,
				input.ReportType,
				childIdempotencyKey(idempotencyKey, "legal_compliance_scan"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.compliance.scan.started",
				TargetID:      ownerResult.ReportID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RunComplianceScanResult{
				ReportID:            ownerResult.ReportID,
				ReportType:          ownerResult.ReportType,
				Status:              ownerResult.Status,
				FindingsCount:       ownerResult.FindingsCount,
				DownloadURL:         ownerResult.DownloadURL,
				CreatedAt:           ownerResult.CreatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RunComplianceScanResult{}, err
	}
	return output, nil
}

func (s Service) GetSupportTicket(ctx context.Context, input GetSupportTicketInput) (SupportTicketResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.TicketID = strings.TrimSpace(input.TicketID)
	if input.ActorID == "" {
		return SupportTicketResult{}, domainerrors.ErrUnauthorized
	}
	if input.TicketID == "" {
		return SupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	if s.SupportClient == nil {
		return SupportTicketResult{}, domainerrors.ErrDependencyUnavailable
	}

	row, err := s.SupportClient.GetTicket(ctx, input.ActorID, input.TicketID)
	if err != nil {
		return SupportTicketResult{}, err
	}
	return mapSupportTicketResult(row), nil
}

func (s Service) SearchSupportTickets(ctx context.Context, input SearchSupportTicketsInput) ([]SupportTicketResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Query = strings.TrimSpace(input.Query)
	input.Status = strings.TrimSpace(input.Status)
	input.Category = strings.TrimSpace(input.Category)
	input.AssignedTo = strings.TrimSpace(input.AssignedTo)
	if input.ActorID == "" {
		return nil, domainerrors.ErrUnauthorized
	}
	if s.SupportClient == nil {
		return nil, domainerrors.ErrDependencyUnavailable
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}
	if input.Limit > 200 {
		input.Limit = 200
	}

	rows, err := s.SupportClient.SearchTickets(ctx, input.ActorID, ports.SupportTicketSearchFilter{
		Query:      input.Query,
		Status:     input.Status,
		Category:   input.Category,
		AssignedTo: input.AssignedTo,
		Limit:      input.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]SupportTicketResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapSupportTicketResult(row))
	}
	return out, nil
}

func (s Service) AssignSupportTicket(
	ctx context.Context,
	idempotencyKey string,
	input AssignSupportTicketInput,
) (AssignSupportTicketResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.TicketID = strings.TrimSpace(input.TicketID)
	input.AgentID = strings.TrimSpace(input.AgentID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return AssignSupportTicketResult{}, domainerrors.ErrUnauthorized
	}
	if input.TicketID == "" || input.AgentID == "" || input.Reason == "" {
		return AssignSupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return AssignSupportTicketResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.SupportClient == nil {
		return AssignSupportTicketResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output AssignSupportTicketResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			row, err := s.SupportClient.AssignTicket(
				ctx,
				input.ActorID,
				input.TicketID,
				input.AgentID,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "support_ticket_assign"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.support.ticket.updated",
				TargetID:      input.TicketID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(AssignSupportTicketResult{
				SupportTicketResult: mapSupportTicketResult(row),
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return AssignSupportTicketResult{}, err
	}
	return output, nil
}

func (s Service) UpdateSupportTicket(
	ctx context.Context,
	idempotencyKey string,
	input UpdateSupportTicketInput,
) (UpdateSupportTicketResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.TicketID = strings.TrimSpace(input.TicketID)
	input.Status = strings.TrimSpace(input.Status)
	input.SubStatus = strings.TrimSpace(input.SubStatus)
	input.Priority = strings.TrimSpace(input.Priority)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return UpdateSupportTicketResult{}, domainerrors.ErrUnauthorized
	}
	if input.TicketID == "" || input.Reason == "" {
		return UpdateSupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	if input.Status == "" && input.SubStatus == "" && input.Priority == "" {
		return UpdateSupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return UpdateSupportTicketResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.SupportClient == nil {
		return UpdateSupportTicketResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output UpdateSupportTicketResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			row, err := s.SupportClient.UpdateTicket(
				ctx,
				input.ActorID,
				input.TicketID,
				input.Status,
				input.SubStatus,
				input.Priority,
				input.Reason,
				childIdempotencyKey(idempotencyKey, "support_ticket_update"),
			)
			if err != nil {
				return nil, err
			}
			auditAction := "admin.support.ticket.updated"
			if strings.Contains(strings.ToLower(input.SubStatus), "escalat") {
				auditAction = "admin.support.escalation.created"
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        auditAction,
				TargetID:      input.TicketID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(UpdateSupportTicketResult{
				SupportTicketResult: mapSupportTicketResult(row),
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return UpdateSupportTicketResult{}, err
	}
	return output, nil
}

func (s Service) SaveEditorCampaign(
	ctx context.Context,
	idempotencyKey string,
	input SaveEditorCampaignInput,
) (SaveEditorCampaignResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.EditorID = strings.TrimSpace(input.EditorID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return SaveEditorCampaignResult{}, domainerrors.ErrUnauthorized
	}
	if input.EditorID == "" || input.CampaignID == "" || input.Reason == "" {
		return SaveEditorCampaignResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return SaveEditorCampaignResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.EditorWorkflowClient == nil {
		return SaveEditorCampaignResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output SaveEditorCampaignResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.EditorWorkflowClient.SaveCampaign(
				ctx,
				input.ActorID,
				input.EditorID,
				input.CampaignID,
				childIdempotencyKey(idempotencyKey, "creator_workflow_editor_save"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.creator_workflow.editor.campaign.saved",
				TargetID:      input.CampaignID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(SaveEditorCampaignResult{
				CampaignID:          ownerResult.CampaignID,
				Saved:               ownerResult.Saved,
				SavedAt:             ownerResult.SavedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return SaveEditorCampaignResult{}, err
	}
	return output, nil
}

func (s Service) RequestClippingExport(
	ctx context.Context,
	idempotencyKey string,
	input RequestClippingExportInput,
) (RequestClippingExportResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Format = strings.TrimSpace(input.Format)
	input.Resolution = strings.TrimSpace(input.Resolution)
	input.Bitrate = strings.TrimSpace(input.Bitrate)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RequestClippingExportResult{}, domainerrors.ErrUnauthorized
	}
	if input.UserID == "" || input.ProjectID == "" || input.Reason == "" {
		return RequestClippingExportResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RequestClippingExportResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.ClippingWorkflowClient == nil {
		return RequestClippingExportResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RequestClippingExportResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.ClippingWorkflowClient.RequestExport(
				ctx,
				input.ActorID,
				input.UserID,
				input.ProjectID,
				input.Format,
				input.Resolution,
				input.FPS,
				input.Bitrate,
				input.CampaignID,
				childIdempotencyKey(idempotencyKey, "creator_workflow_clipping_export"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.creator_workflow.clipping.export.requested",
				TargetID:      ownerResult.ExportID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RequestClippingExportResult{
				ExportID:            ownerResult.ExportID,
				ProjectID:           ownerResult.ProjectID,
				Status:              ownerResult.Status,
				ProgressPercent:     ownerResult.ProgressPercent,
				OutputURL:           ownerResult.OutputURL,
				ProviderJobID:       ownerResult.ProviderJobID,
				CreatedAt:           ownerResult.CreatedAt,
				CompletedAt:         ownerResult.CompletedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RequestClippingExportResult{}, err
	}
	return output, nil
}

func (s Service) DeployAutoClippingModel(
	ctx context.Context,
	idempotencyKey string,
	input DeployAutoClippingModelInput,
) (DeployAutoClippingModelResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.ModelName = strings.TrimSpace(input.ModelName)
	input.VersionTag = strings.TrimSpace(input.VersionTag)
	input.ModelArtifactKey = strings.TrimSpace(input.ModelArtifactKey)
	input.Description = strings.TrimSpace(input.Description)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return DeployAutoClippingModelResult{}, domainerrors.ErrUnauthorized
	}
	if input.ModelName == "" || input.VersionTag == "" || input.ModelArtifactKey == "" || input.Reason == "" {
		return DeployAutoClippingModelResult{}, domainerrors.ErrInvalidInput
	}
	if input.CanaryPercentage < 0 || input.CanaryPercentage > 100 {
		return DeployAutoClippingModelResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return DeployAutoClippingModelResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.AutoClippingClient == nil {
		return DeployAutoClippingModelResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output DeployAutoClippingModelResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.AutoClippingClient.DeployModel(
				ctx,
				input.ActorID,
				ports.AutoClippingModelDeployInput{
					ModelName:        input.ModelName,
					VersionTag:       input.VersionTag,
					ModelArtifactKey: input.ModelArtifactKey,
					CanaryPercentage: input.CanaryPercentage,
					Description:      input.Description,
					Reason:           input.Reason,
				},
				childIdempotencyKey(idempotencyKey, "creator_workflow_auto_clipping_deploy"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.creator_workflow.auto_clipping.model.deployed",
				TargetID:      ownerResult.ModelVersionID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(DeployAutoClippingModelResult{
				ModelVersionID:      ownerResult.ModelVersionID,
				DeploymentStatus:    ownerResult.DeploymentStatus,
				DeployedAt:          ownerResult.DeployedAt,
				Message:             ownerResult.Message,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return DeployAutoClippingModelResult{}, err
	}
	return output, nil
}

func (s Service) RotateIntegrationKey(
	ctx context.Context,
	idempotencyKey string,
	input RotateIntegrationKeyInput,
) (RotateIntegrationKeyResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.KeyID = strings.TrimSpace(input.KeyID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return RotateIntegrationKeyResult{}, domainerrors.ErrUnauthorized
	}
	if input.KeyID == "" || input.Reason == "" {
		return RotateIntegrationKeyResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RotateIntegrationKeyResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.DeveloperPortalClient == nil {
		return RotateIntegrationKeyResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output RotateIntegrationKeyResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.DeveloperPortalClient.RotateAPIKey(
				ctx,
				input.ActorID,
				input.KeyID,
				childIdempotencyKey(idempotencyKey, "integration_key_rotate"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.integration.key.rotated",
				TargetID:      input.KeyID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(RotateIntegrationKeyResult{
				RotationID:          ownerResult.RotationID,
				DeveloperID:         ownerResult.DeveloperID,
				OldKeyID:            ownerResult.OldKeyID,
				NewKeyID:            ownerResult.NewKeyID,
				CreatedAt:           ownerResult.CreatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return RotateIntegrationKeyResult{}, err
	}
	return output, nil
}

func (s Service) TestIntegrationWorkflow(
	ctx context.Context,
	idempotencyKey string,
	input TestIntegrationWorkflowInput,
) (TestIntegrationWorkflowResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.WorkflowID = strings.TrimSpace(input.WorkflowID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return TestIntegrationWorkflowResult{}, domainerrors.ErrUnauthorized
	}
	if input.WorkflowID == "" || input.Reason == "" {
		return TestIntegrationWorkflowResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return TestIntegrationWorkflowResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.IntegrationHubClient == nil {
		return TestIntegrationWorkflowResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output TestIntegrationWorkflowResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.IntegrationHubClient.TestWorkflow(
				ctx,
				input.ActorID,
				input.WorkflowID,
				childIdempotencyKey(idempotencyKey, "integration_workflow_test"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.integration.workflow.tested",
				TargetID:      input.WorkflowID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(TestIntegrationWorkflowResult{
				ExecutionID:         ownerResult.ExecutionID,
				WorkflowID:          ownerResult.WorkflowID,
				Status:              ownerResult.Status,
				TestRun:             ownerResult.TestRun,
				StartedAt:           ownerResult.StartedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return TestIntegrationWorkflowResult{}, err
	}
	return output, nil
}

func (s Service) ReplayWebhook(
	ctx context.Context,
	idempotencyKey string,
	input ReplayWebhookInput,
) (ReplayWebhookResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.WebhookID = strings.TrimSpace(input.WebhookID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return ReplayWebhookResult{}, domainerrors.ErrUnauthorized
	}
	if input.WebhookID == "" || input.Reason == "" {
		return ReplayWebhookResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return ReplayWebhookResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.WebhookManagerClient == nil {
		return ReplayWebhookResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output ReplayWebhookResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.WebhookManagerClient.ReplayWebhook(
				ctx,
				input.ActorID,
				input.WebhookID,
				childIdempotencyKey(idempotencyKey, "webhook_replay"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.webhook.replayed",
				TargetID:      input.WebhookID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(ReplayWebhookResult{
				DeliveryID:          ownerResult.DeliveryID,
				WebhookID:           ownerResult.WebhookID,
				Status:              ownerResult.Status,
				HTTPStatus:          ownerResult.HTTPStatus,
				LatencyMS:           ownerResult.LatencyMS,
				Timestamp:           ownerResult.Timestamp,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return ReplayWebhookResult{}, err
	}
	return output, nil
}

func (s Service) DisableWebhook(
	ctx context.Context,
	idempotencyKey string,
	input DisableWebhookInput,
) (DisableWebhookResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.WebhookID = strings.TrimSpace(input.WebhookID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return DisableWebhookResult{}, domainerrors.ErrUnauthorized
	}
	if input.WebhookID == "" || input.Reason == "" {
		return DisableWebhookResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return DisableWebhookResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.WebhookManagerClient == nil {
		return DisableWebhookResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output DisableWebhookResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.WebhookManagerClient.DisableWebhook(
				ctx,
				input.ActorID,
				input.WebhookID,
				childIdempotencyKey(idempotencyKey, "webhook_disable"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.webhook.disabled",
				TargetID:      input.WebhookID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(DisableWebhookResult{
				WebhookID:           ownerResult.WebhookID,
				Status:              ownerResult.Status,
				UpdatedAt:           ownerResult.UpdatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return DisableWebhookResult{}, err
	}
	return output, nil
}

func (s Service) GetWebhookDeliveries(ctx context.Context, input GetWebhookDeliveriesInput) ([]WebhookDeliveryResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.WebhookID = strings.TrimSpace(input.WebhookID)
	if input.ActorID == "" {
		return nil, domainerrors.ErrUnauthorized
	}
	if input.WebhookID == "" {
		return nil, domainerrors.ErrInvalidInput
	}
	if s.WebhookManagerClient == nil {
		return nil, domainerrors.ErrDependencyUnavailable
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}
	if input.Limit > 200 {
		input.Limit = 200
	}

	rows, err := s.WebhookManagerClient.ListDeliveries(ctx, input.ActorID, input.WebhookID, input.Limit)
	if err != nil {
		return nil, err
	}
	out := make([]WebhookDeliveryResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, WebhookDeliveryResult{
			DeliveryID:      row.DeliveryID,
			WebhookID:       row.WebhookID,
			OriginalEventID: row.OriginalEventID,
			OriginalType:    row.OriginalType,
			HTTPStatus:      row.HTTPStatus,
			LatencyMS:       row.LatencyMS,
			RetryCount:      row.RetryCount,
			DeliveredAt:     row.DeliveredAt,
			IsTest:          row.IsTest,
			Success:         row.Success,
		})
	}
	return out, nil
}

func (s Service) GetWebhookAnalytics(ctx context.Context, input GetWebhookAnalyticsInput) (WebhookAnalyticsResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.WebhookID = strings.TrimSpace(input.WebhookID)
	if input.ActorID == "" {
		return WebhookAnalyticsResult{}, domainerrors.ErrUnauthorized
	}
	if input.WebhookID == "" {
		return WebhookAnalyticsResult{}, domainerrors.ErrInvalidInput
	}
	if s.WebhookManagerClient == nil {
		return WebhookAnalyticsResult{}, domainerrors.ErrDependencyUnavailable
	}

	row, err := s.WebhookManagerClient.GetAnalytics(ctx, input.ActorID, input.WebhookID)
	if err != nil {
		return WebhookAnalyticsResult{}, err
	}
	byType := make(map[string]WebhookAnalyticsMetrics, len(row.ByEventType))
	for key, value := range row.ByEventType {
		byType[key] = WebhookAnalyticsMetrics{
			Total:      value.Total,
			Success:    value.Success,
			Failed:     value.Failed,
			AvgLatency: value.AvgLatency,
		}
	}
	return WebhookAnalyticsResult{
		TotalDeliveries:      row.TotalDeliveries,
		SuccessfulDeliveries: row.SuccessfulDeliveries,
		FailedDeliveries:     row.FailedDeliveries,
		SuccessRate:          row.SuccessRate,
		AvgLatencyMS:         row.AvgLatencyMS,
		P95LatencyMS:         row.P95LatencyMS,
		P99LatencyMS:         row.P99LatencyMS,
		ByEventType:          byType,
	}, nil
}

func (s Service) CreateMigrationPlan(
	ctx context.Context,
	idempotencyKey string,
	input CreateMigrationPlanInput,
) (CreateMigrationPlanResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.ServiceName = strings.TrimSpace(input.ServiceName)
	input.Environment = strings.TrimSpace(input.Environment)
	input.Version = strings.TrimSpace(input.Version)
	input.RiskLevel = strings.TrimSpace(input.RiskLevel)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return CreateMigrationPlanResult{}, domainerrors.ErrUnauthorized
	}
	if input.ServiceName == "" || input.Environment == "" || input.Version == "" || input.Reason == "" || len(input.Plan) == 0 {
		return CreateMigrationPlanResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return CreateMigrationPlanResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.DataMigrationClient == nil {
		return CreateMigrationPlanResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output CreateMigrationPlanResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.DataMigrationClient.CreatePlan(
				ctx,
				input.ActorID,
				input.ServiceName,
				input.Environment,
				input.Version,
				input.Plan,
				input.DryRun,
				input.RiskLevel,
				childIdempotencyKey(idempotencyKey, "backfill_plan_create"),
			)
			if err != nil {
				return nil, err
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        "admin.backfill.started",
				TargetID:      ownerResult.PlanID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(CreateMigrationPlanResult{
				PlanID:              ownerResult.PlanID,
				ServiceName:         ownerResult.ServiceName,
				Environment:         ownerResult.Environment,
				Version:             ownerResult.Version,
				Plan:                ownerResult.Plan,
				Status:              ownerResult.Status,
				DryRun:              ownerResult.DryRun,
				RiskLevel:           ownerResult.RiskLevel,
				StagingValidated:    ownerResult.StagingValidated,
				BackupRequired:      ownerResult.BackupRequired,
				CreatedBy:           ownerResult.CreatedBy,
				CreatedAt:           ownerResult.CreatedAt,
				UpdatedAt:           ownerResult.UpdatedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return CreateMigrationPlanResult{}, err
	}
	return output, nil
}

func (s Service) ListMigrationPlans(ctx context.Context, input ListMigrationPlansInput) ([]MigrationPlanResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.ActorID == "" {
		return nil, domainerrors.ErrUnauthorized
	}
	if s.DataMigrationClient == nil {
		return nil, domainerrors.ErrDependencyUnavailable
	}
	rows, err := s.DataMigrationClient.ListPlans(ctx, input.ActorID)
	if err != nil {
		return nil, err
	}
	out := make([]MigrationPlanResult, 0, len(rows))
	for _, row := range rows {
		out = append(out, MigrationPlanResult{
			PlanID:           row.PlanID,
			ServiceName:      row.ServiceName,
			Environment:      row.Environment,
			Version:          row.Version,
			Plan:             row.Plan,
			Status:           row.Status,
			DryRun:           row.DryRun,
			RiskLevel:        row.RiskLevel,
			StagingValidated: row.StagingValidated,
			BackupRequired:   row.BackupRequired,
			CreatedBy:        row.CreatedBy,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return out, nil
}

func (s Service) StartMigrationRun(
	ctx context.Context,
	idempotencyKey string,
	input StartMigrationRunInput,
) (StartMigrationRunResult, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.PlanID = strings.TrimSpace(input.PlanID)
	input.Reason = strings.TrimSpace(input.Reason)
	input.SourceIP = strings.TrimSpace(input.SourceIP)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)

	if input.ActorID == "" {
		return StartMigrationRunResult{}, domainerrors.ErrUnauthorized
	}
	if input.PlanID == "" || input.Reason == "" {
		return StartMigrationRunResult{}, domainerrors.ErrInvalidInput
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return StartMigrationRunResult{}, domainerrors.ErrIdempotencyRequired
	}
	if s.DataMigrationClient == nil {
		return StartMigrationRunResult{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashPayload(input)
	now := s.now()
	var output StartMigrationRunResult
	if err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(payload []byte) error { return json.Unmarshal(payload, &output) },
		func() ([]byte, error) {
			ownerResult, err := s.DataMigrationClient.CreateRun(
				ctx,
				input.ActorID,
				input.PlanID,
				childIdempotencyKey(idempotencyKey, "backfill_run_create"),
			)
			if err != nil {
				return nil, err
			}
			auditAction := "admin.backfill.completed"
			if strings.Contains(strings.ToLower(ownerResult.Status), "fail") {
				auditAction = "admin.backfill.failed"
			}
			auditRow := ports.AuditLog{
				AuditID:       fmt.Sprintf("audit_%d", now.UnixNano()),
				ActorID:       input.ActorID,
				Action:        auditAction,
				TargetID:      ownerResult.RunID,
				Justification: input.Reason,
				OccurredAt:    now,
				SourceIP:      input.SourceIP,
				CorrelationID: input.CorrelationID,
			}
			if err := s.Repo.AppendAuditLog(ctx, auditRow); err != nil {
				return nil, err
			}
			return json.Marshal(StartMigrationRunResult{
				RunID:               ownerResult.RunID,
				PlanID:              ownerResult.PlanID,
				Status:              ownerResult.Status,
				OperatorID:          ownerResult.OperatorID,
				SnapshotCreated:     ownerResult.SnapshotCreated,
				RollbackAvailable:   ownerResult.RollbackAvailable,
				ValidationStatus:    ownerResult.ValidationStatus,
				BackfillJobID:       ownerResult.BackfillJobID,
				StartedAt:           ownerResult.StartedAt,
				CompletedAt:         ownerResult.CompletedAt,
				ControlPlaneAuditID: auditRow.AuditID,
			})
		},
	); err != nil {
		return StartMigrationRunResult{}, err
	}
	return output, nil
}

func (s Service) ListRecentActions(ctx context.Context, limit int) ([]ports.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.Repo.ListRecentAuditLogs(ctx, limit)
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

func (s Service) runIdempotent(
	ctx context.Context,
	idempotencyKey string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	existing, err := s.Idempotency.Get(ctx, idempotencyKey, now)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(existing.ResponseBody)
	}
	if err := s.Idempotency.Reserve(ctx, idempotencyKey, requestHash, now.Add(s.idempotencyTTL())); err != nil {
		return err
	}
	body, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Complete(ctx, idempotencyKey, body, now); err != nil {
		return err
	}
	return decode(body)
}

func childIdempotencyKey(parentKey string, operation string) string {
	return fmt.Sprintf("m86:%s:%s", strings.TrimSpace(operation), strings.TrimSpace(parentKey))
}

func hashPayload(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func mapSupportTicketResult(row ports.SupportTicketResult) SupportTicketResult {
	return SupportTicketResult{
		TicketID:         row.TicketID,
		UserID:           row.UserID,
		Subject:          row.Subject,
		Description:      row.Description,
		Category:         row.Category,
		Priority:         row.Priority,
		Status:           row.Status,
		SubStatus:        row.SubStatus,
		AssignedAgentID:  row.AssignedAgentID,
		SLAResponseDueAt: row.SLAResponseDueAt,
		LastActivityAt:   row.LastActivityAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
