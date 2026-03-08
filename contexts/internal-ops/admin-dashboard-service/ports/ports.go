package ports

import (
	"context"
	"time"
)

type AuditLog struct {
	AuditID       string
	ActorID       string
	Action        string
	TargetID      string
	Justification string
	OccurredAt    time.Time
	SourceIP      string
	CorrelationID string
}

type RoleGrantResult struct {
	AssignmentID string
	UserID       string
	RoleID       string
	AuditLogID   string
}

type ModerationDecisionResult struct {
	DecisionID   string
	SubmissionID string
	CampaignID   string
	ModeratorID  string
	Action       string
	Reason       string
	Notes        string
	QueueStatus  string
	CreatedAt    time.Time
}

type AbuseLockoutResult struct {
	ThreatID        string
	UserID          string
	Status          string
	ReleasedAt      time.Time
	OwnerAuditLogID string
}

type FinanceRefundResult struct {
	RefundID      string
	TransactionID string
	UserID        string
	Amount        float64
	Reason        string
	CreatedAt     time.Time
}

type BillingRefundResult struct {
	RefundID    string
	InvoiceID   string
	LineItemID  string
	Amount      float64
	Reason      string
	ProcessedAt time.Time
}

type RewardRecalculationResult struct {
	SubmissionID  string
	UserID        string
	CampaignID    string
	Status        string
	NetAmount     float64
	RolloverTotal float64
	CalculatedAt  time.Time
	EligibleAt    *time.Time
}

type AffiliateSuspensionResult struct {
	AffiliateID string
	Status      string
	UpdatedAt   time.Time
}

type AffiliateAttributionResult struct {
	AttributionID string
	AffiliateID   string
	OrderID       string
	Amount        float64
	AttributedAt  time.Time
}

type PayoutRetryResult struct {
	PayoutID      string
	UserID        string
	Status        string
	FailureReason string
	ProcessedAt   time.Time
}

type EditorCampaignSaveResult struct {
	CampaignID string
	Saved      bool
	SavedAt    *time.Time
}

type ClippingExportResult struct {
	ExportID        string
	ProjectID       string
	Status          string
	ProgressPercent int
	OutputURL       string
	ProviderJobID   string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

type AutoClippingModelDeployInput struct {
	ModelName        string
	VersionTag       string
	ModelArtifactKey string
	CanaryPercentage int
	Description      string
	Reason           string
}

type AutoClippingModelDeployResult struct {
	ModelVersionID   string
	DeploymentStatus string
	DeployedAt       time.Time
	Message          string
}

type IntegrationKeyRotationResult struct {
	RotationID  string
	DeveloperID string
	OldKeyID    string
	NewKeyID    string
	CreatedAt   time.Time
}

type IntegrationWorkflowTestResult struct {
	ExecutionID string
	WorkflowID  string
	Status      string
	TestRun     bool
	StartedAt   time.Time
}

type WebhookReplayResult struct {
	DeliveryID string
	WebhookID  string
	Status     string
	HTTPStatus int
	LatencyMS  int64
	Timestamp  time.Time
}

type WebhookEndpointResult struct {
	WebhookID   string
	Status      string
	UpdatedAt   time.Time
	EndpointURL string
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

type MigrationRunResult struct {
	RunID             string
	PlanID            string
	Status            string
	OperatorID        string
	SnapshotCreated   bool
	RollbackAvailable bool
	ValidationStatus  string
	BackfillJobID     string
	StartedAt         time.Time
	CompletedAt       time.Time
}

type DisputeResolutionResult struct {
	DisputeID      string
	Status         string
	ResolutionType string
	RefundAmount   float64
	ProcessedAt    time.Time
}

type ConsentRecordResult struct {
	UserID        string
	Status        string
	Preferences   map[string]bool
	LastUpdated   time.Time
	LastUpdatedBy string
}

type ConsentChangeResult struct {
	UserID    string
	Status    string
	UpdatedAt time.Time
}

type PortabilityRequestResult struct {
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

type RetentionHoldResult struct {
	HoldID    string
	EntityID  string
	DataType  string
	Reason    string
	Status    string
	CreatedAt time.Time
	ExpiresAt *time.Time
}

type LegalHoldCheckResult struct {
	EntityType string
	EntityID   string
	Held       bool
	HoldID     string
}

type LegalHoldResult struct {
	HoldID     string
	EntityType string
	EntityID   string
	Reason     string
	Status     string
	CreatedAt  time.Time
	ReleasedAt *time.Time
}

type LegalComplianceReportResult struct {
	ReportID      string
	ReportType    string
	Status        string
	FindingsCount int
	DownloadURL   string
	CreatedAt     time.Time
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

type SupportTicketSearchFilter struct {
	Query      string
	Status     string
	Category   string
	AssignedTo string
	Limit      int
}

type Repository interface {
	AppendAuditLog(ctx context.Context, row AuditLog) error
	ListRecentAuditLogs(ctx context.Context, limit int) ([]AuditLog, error)
}

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseBody []byte, at time.Time) error
}

type Clock interface {
	Now() time.Time
}

type AuthorizationClient interface {
	GrantRole(
		ctx context.Context,
		adminID string,
		userID string,
		roleID string,
		reason string,
		idempotencyKey string,
	) (RoleGrantResult, error)
}

type ModerationClient interface {
	ApproveSubmission(
		ctx context.Context,
		moderatorID string,
		submissionID string,
		campaignID string,
		reason string,
		notes string,
		idempotencyKey string,
	) (ModerationDecisionResult, error)
	RejectSubmission(
		ctx context.Context,
		moderatorID string,
		submissionID string,
		campaignID string,
		reason string,
		notes string,
		idempotencyKey string,
	) (ModerationDecisionResult, error)
}

type AbusePreventionClient interface {
	ReleaseLockout(
		ctx context.Context,
		adminID string,
		userID string,
		reason string,
		idempotencyKey string,
	) (AbuseLockoutResult, error)
}

type FinanceClient interface {
	CreateRefund(
		ctx context.Context,
		adminID string,
		transactionID string,
		userID string,
		amount float64,
		reason string,
		idempotencyKey string,
	) (FinanceRefundResult, error)
}

type BillingClient interface {
	CreateInvoiceRefund(
		ctx context.Context,
		adminID string,
		invoiceID string,
		lineItemID string,
		amount float64,
		reason string,
		idempotencyKey string,
	) (BillingRefundResult, error)
}

type RewardClient interface {
	RecalculateReward(
		ctx context.Context,
		adminID string,
		userID string,
		submissionID string,
		campaignID string,
		lockedViews int64,
		ratePer1K float64,
		fraudScore float64,
		verificationCompletedAt time.Time,
		reason string,
		idempotencyKey string,
	) (RewardRecalculationResult, error)
}

type AffiliateClient interface {
	SuspendAffiliate(
		ctx context.Context,
		adminID string,
		affiliateID string,
		reason string,
		idempotencyKey string,
	) (AffiliateSuspensionResult, error)
	CreateAttribution(
		ctx context.Context,
		adminID string,
		affiliateID string,
		clickID string,
		orderID string,
		conversionID string,
		amount float64,
		currency string,
		reason string,
		idempotencyKey string,
	) (AffiliateAttributionResult, error)
}

type PayoutClient interface {
	RetryFailedPayout(
		ctx context.Context,
		adminID string,
		payoutID string,
		reason string,
		idempotencyKey string,
	) (PayoutRetryResult, error)
}

type ResolutionClient interface {
	ResolveDispute(
		ctx context.Context,
		adminID string,
		disputeID string,
		action string,
		reason string,
		notes string,
		refundAmount float64,
		idempotencyKey string,
	) (DisputeResolutionResult, error)
}

type ConsentClient interface {
	GetConsent(
		ctx context.Context,
		adminID string,
		userID string,
	) (ConsentRecordResult, error)
	UpdateConsent(
		ctx context.Context,
		adminID string,
		userID string,
		preferences map[string]bool,
		reason string,
		idempotencyKey string,
	) (ConsentChangeResult, error)
	WithdrawConsent(
		ctx context.Context,
		adminID string,
		userID string,
		category string,
		reason string,
		idempotencyKey string,
	) (ConsentChangeResult, error)
}

type PortabilityClient interface {
	CreateExport(
		ctx context.Context,
		adminID string,
		userID string,
		format string,
		reason string,
		idempotencyKey string,
	) (PortabilityRequestResult, error)
	GetExport(
		ctx context.Context,
		adminID string,
		requestID string,
	) (PortabilityRequestResult, error)
	CreateEraseRequest(
		ctx context.Context,
		adminID string,
		userID string,
		reason string,
		idempotencyKey string,
	) (PortabilityRequestResult, error)
}

type RetentionClient interface {
	CreateLegalHold(
		ctx context.Context,
		adminID string,
		entityID string,
		dataType string,
		reason string,
		expiresAt *time.Time,
		idempotencyKey string,
	) (RetentionHoldResult, error)
}

type LegalClient interface {
	CheckHold(
		ctx context.Context,
		adminID string,
		entityType string,
		entityID string,
	) (LegalHoldCheckResult, error)
	ReleaseHold(
		ctx context.Context,
		adminID string,
		holdID string,
		reason string,
		idempotencyKey string,
	) (LegalHoldResult, error)
	RunComplianceScan(
		ctx context.Context,
		adminID string,
		reportType string,
		idempotencyKey string,
	) (LegalComplianceReportResult, error)
}

type SupportClient interface {
	GetTicket(
		ctx context.Context,
		adminID string,
		ticketID string,
	) (SupportTicketResult, error)
	SearchTickets(
		ctx context.Context,
		adminID string,
		filter SupportTicketSearchFilter,
	) ([]SupportTicketResult, error)
	AssignTicket(
		ctx context.Context,
		adminID string,
		ticketID string,
		agentID string,
		reason string,
		idempotencyKey string,
	) (SupportTicketResult, error)
	UpdateTicket(
		ctx context.Context,
		adminID string,
		ticketID string,
		status string,
		subStatus string,
		priority string,
		reason string,
		idempotencyKey string,
	) (SupportTicketResult, error)
}

type EditorWorkflowClient interface {
	SaveCampaign(
		ctx context.Context,
		adminID string,
		editorID string,
		campaignID string,
		idempotencyKey string,
	) (EditorCampaignSaveResult, error)
}

type ClippingWorkflowClient interface {
	RequestExport(
		ctx context.Context,
		adminID string,
		userID string,
		projectID string,
		format string,
		resolution string,
		fps int,
		bitrate string,
		campaignID string,
		idempotencyKey string,
	) (ClippingExportResult, error)
}

type AutoClippingClient interface {
	DeployModel(
		ctx context.Context,
		adminID string,
		input AutoClippingModelDeployInput,
		idempotencyKey string,
	) (AutoClippingModelDeployResult, error)
}

type DeveloperPortalClient interface {
	RotateAPIKey(
		ctx context.Context,
		adminID string,
		keyID string,
		idempotencyKey string,
	) (IntegrationKeyRotationResult, error)
}

type IntegrationHubClient interface {
	TestWorkflow(
		ctx context.Context,
		adminID string,
		workflowID string,
		idempotencyKey string,
	) (IntegrationWorkflowTestResult, error)
}

type WebhookManagerClient interface {
	ReplayWebhook(
		ctx context.Context,
		adminID string,
		webhookID string,
		idempotencyKey string,
	) (WebhookReplayResult, error)
	DisableWebhook(
		ctx context.Context,
		adminID string,
		webhookID string,
		idempotencyKey string,
	) (WebhookEndpointResult, error)
	ListDeliveries(
		ctx context.Context,
		adminID string,
		webhookID string,
		limit int,
	) ([]WebhookDeliveryResult, error)
	GetAnalytics(
		ctx context.Context,
		adminID string,
		webhookID string,
	) (WebhookAnalyticsResult, error)
}

type DataMigrationClient interface {
	CreatePlan(
		ctx context.Context,
		adminID string,
		serviceName string,
		environment string,
		version string,
		plan map[string]interface{},
		dryRun bool,
		riskLevel string,
		idempotencyKey string,
	) (MigrationPlanResult, error)
	ListPlans(
		ctx context.Context,
		adminID string,
	) ([]MigrationPlanResult, error)
	CreateRun(
		ctx context.Context,
		adminID string,
		planID string,
		idempotencyKey string,
	) (MigrationRunResult, error)
}
