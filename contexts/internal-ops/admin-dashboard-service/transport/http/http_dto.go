package http

type RecordAdminActionRequest struct {
	Action        string `json:"action"`
	TargetID      string `json:"target_id"`
	Justification string `json:"justification"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RecordAdminActionResponse struct {
	AuditID    string `json:"audit_id"`
	OccurredAt string `json:"occurred_at"`
}

type GrantIdentityRoleRequest struct {
	UserID        string `json:"user_id"`
	RoleID        string `json:"role_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type GrantIdentityRoleResponse struct {
	AssignmentID        string `json:"assignment_id"`
	UserID              string `json:"user_id"`
	RoleID              string `json:"role_id"`
	OwnerAuditLogID     string `json:"owner_audit_log_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
	OccurredAt          string `json:"occurred_at"`
}

type ModerateSubmissionRequest struct {
	SubmissionID  string `json:"submission_id"`
	CampaignID    string `json:"campaign_id"`
	Action        string `json:"action"`
	Reason        string `json:"reason"`
	Notes         string `json:"notes,omitempty"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type ModerateSubmissionResponse struct {
	DecisionID          string `json:"decision_id"`
	SubmissionID        string `json:"submission_id"`
	CampaignID          string `json:"campaign_id"`
	ModeratorID         string `json:"moderator_id"`
	Action              string `json:"action"`
	Reason              string `json:"reason"`
	Notes               string `json:"notes,omitempty"`
	QueueStatus         string `json:"queue_status"`
	CreatedAt           string `json:"created_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type ReleaseAbuseLockoutRequest struct {
	UserID        string `json:"user_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type ReleaseAbuseLockoutResponse struct {
	ThreatID            string `json:"threat_id"`
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	ReleasedAt          string `json:"released_at"`
	OwnerAuditLogID     string `json:"owner_audit_log_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type CreateFinanceRefundRequest struct {
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	SourceIP      string  `json:"source_ip"`
	CorrelationID string  `json:"correlation_id"`
}

type CreateFinanceRefundResponse struct {
	RefundID            string  `json:"refund_id"`
	TransactionID       string  `json:"transaction_id"`
	UserID              string  `json:"user_id"`
	Amount              float64 `json:"amount"`
	Reason              string  `json:"reason"`
	CreatedAt           string  `json:"created_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type CreateBillingRefundRequest struct {
	InvoiceID     string  `json:"invoice_id"`
	LineItemID    string  `json:"line_item_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
	SourceIP      string  `json:"source_ip"`
	CorrelationID string  `json:"correlation_id"`
}

type CreateBillingRefundResponse struct {
	RefundID            string  `json:"refund_id"`
	InvoiceID           string  `json:"invoice_id"`
	LineItemID          string  `json:"line_item_id"`
	Amount              float64 `json:"amount"`
	Reason              string  `json:"reason"`
	ProcessedAt         string  `json:"processed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type RecalculateRewardRequest struct {
	UserID                  string  `json:"user_id"`
	SubmissionID            string  `json:"submission_id"`
	CampaignID              string  `json:"campaign_id"`
	LockedViews             int64   `json:"locked_views,omitempty"`
	RatePer1K               float64 `json:"rate_per_1k,omitempty"`
	FraudScore              float64 `json:"fraud_score,omitempty"`
	VerificationCompletedAt string  `json:"verification_completed_at,omitempty"`
	Reason                  string  `json:"reason"`
	SourceIP                string  `json:"source_ip"`
	CorrelationID           string  `json:"correlation_id"`
}

type RecalculateRewardResponse struct {
	SubmissionID        string  `json:"submission_id"`
	UserID              string  `json:"user_id"`
	CampaignID          string  `json:"campaign_id"`
	Status              string  `json:"status"`
	NetAmount           float64 `json:"net_amount"`
	RolloverTotal       float64 `json:"rollover_total"`
	CalculatedAt        string  `json:"calculated_at"`
	EligibleAt          *string `json:"eligible_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type SuspendAffiliateRequest struct {
	AffiliateID   string `json:"affiliate_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type SuspendAffiliateResponse struct {
	AffiliateID         string `json:"affiliate_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type CreateAffiliateAttributionRequest struct {
	AffiliateID   string  `json:"affiliate_id"`
	ClickID       string  `json:"click_id,omitempty"`
	OrderID       string  `json:"order_id"`
	ConversionID  string  `json:"conversion_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency,omitempty"`
	Reason        string  `json:"reason"`
	SourceIP      string  `json:"source_ip"`
	CorrelationID string  `json:"correlation_id"`
}

type CreateAffiliateAttributionResponse struct {
	AttributionID       string  `json:"attribution_id"`
	AffiliateID         string  `json:"affiliate_id"`
	OrderID             string  `json:"order_id"`
	Amount              float64 `json:"amount"`
	AttributedAt        string  `json:"attributed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type RetryFailedPayoutRequest struct {
	PayoutID      string `json:"payout_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RetryFailedPayoutResponse struct {
	PayoutID            string `json:"payout_id"`
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	FailureReason       string `json:"failure_reason,omitempty"`
	ProcessedAt         string `json:"processed_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type ResolveDisputeRequest struct {
	DisputeID     string  `json:"dispute_id"`
	Action        string  `json:"action"`
	Reason        string  `json:"reason"`
	Notes         string  `json:"notes,omitempty"`
	RefundAmount  float64 `json:"refund_amount,omitempty"`
	SourceIP      string  `json:"source_ip"`
	CorrelationID string  `json:"correlation_id"`
}

type ResolveDisputeResponse struct {
	DisputeID           string  `json:"dispute_id"`
	Status              string  `json:"status"`
	ResolutionType      string  `json:"resolution_type,omitempty"`
	RefundAmount        float64 `json:"refund_amount,omitempty"`
	ProcessedAt         string  `json:"processed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type GetConsentRequest struct {
	UserID string `json:"user_id"`
}

type GetConsentResponse struct {
	UserID        string          `json:"user_id"`
	Status        string          `json:"status"`
	Preferences   map[string]bool `json:"preferences"`
	LastUpdated   string          `json:"last_updated"`
	LastUpdatedBy string          `json:"last_updated_by"`
}

type UpdateConsentRequest struct {
	UserID        string          `json:"user_id"`
	Preferences   map[string]bool `json:"preferences"`
	Reason        string          `json:"reason"`
	SourceIP      string          `json:"source_ip"`
	CorrelationID string          `json:"correlation_id"`
}

type UpdateConsentResponse struct {
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type WithdrawConsentRequest struct {
	UserID        string `json:"user_id"`
	Category      string `json:"category,omitempty"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type WithdrawConsentResponse struct {
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type StartDataExportRequest struct {
	UserID        string `json:"user_id"`
	Format        string `json:"format,omitempty"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type StartDataExportResponse struct {
	RequestID           string  `json:"request_id"`
	UserID              string  `json:"user_id"`
	RequestType         string  `json:"request_type"`
	Format              string  `json:"format,omitempty"`
	Status              string  `json:"status"`
	RequestedAt         string  `json:"requested_at"`
	CompletedAt         *string `json:"completed_at,omitempty"`
	DownloadURL         string  `json:"download_url,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type GetDataExportResponse struct {
	RequestID   string  `json:"request_id"`
	UserID      string  `json:"user_id"`
	RequestType string  `json:"request_type"`
	Format      string  `json:"format,omitempty"`
	Status      string  `json:"status"`
	Reason      string  `json:"reason,omitempty"`
	RequestedAt string  `json:"requested_at"`
	CompletedAt *string `json:"completed_at,omitempty"`
	DownloadURL string  `json:"download_url,omitempty"`
}

type RequestDeletionRequest struct {
	UserID        string `json:"user_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RequestDeletionResponse struct {
	RequestID           string  `json:"request_id"`
	UserID              string  `json:"user_id"`
	Status              string  `json:"status"`
	Reason              string  `json:"reason"`
	RequestedAt         string  `json:"requested_at"`
	CompletedAt         *string `json:"completed_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type CreateRetentionLegalHoldRequest struct {
	EntityID      string `json:"entity_id"`
	DataType      string `json:"data_type"`
	Reason        string `json:"reason"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type CreateRetentionLegalHoldResponse struct {
	HoldID              string  `json:"hold_id"`
	EntityID            string  `json:"entity_id"`
	DataType            string  `json:"data_type"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	ExpiresAt           *string `json:"expires_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type CheckLegalHoldRequest struct {
	EntityType    string `json:"entity_type"`
	EntityID      string `json:"entity_id"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type CheckLegalHoldResponse struct {
	EntityType          string `json:"entity_type"`
	EntityID            string `json:"entity_id"`
	Held                bool   `json:"held"`
	HoldID              string `json:"hold_id,omitempty"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type ReleaseLegalHoldRequest struct {
	HoldID        string `json:"hold_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type ReleaseLegalHoldResponse struct {
	HoldID              string  `json:"hold_id"`
	EntityType          string  `json:"entity_type"`
	EntityID            string  `json:"entity_id"`
	Status              string  `json:"status"`
	CreatedAt           string  `json:"created_at"`
	ReleasedAt          *string `json:"released_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type RunComplianceScanRequest struct {
	ReportType    string `json:"report_type,omitempty"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RunComplianceScanResponse struct {
	ReportID            string `json:"report_id"`
	ReportType          string `json:"report_type"`
	Status              string `json:"status"`
	FindingsCount       int    `json:"findings_count"`
	DownloadURL         string `json:"download_url"`
	CreatedAt           string `json:"created_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type GetSupportTicketRequest struct {
	TicketID string `json:"ticket_id"`
}

type SearchSupportTicketsRequest struct {
	Query      string `json:"query,omitempty"`
	Status     string `json:"status,omitempty"`
	Category   string `json:"category,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type SupportTicketResponse struct {
	TicketID         string `json:"ticket_id"`
	UserID           string `json:"user_id"`
	Subject          string `json:"subject"`
	Description      string `json:"description"`
	Category         string `json:"category"`
	Priority         string `json:"priority"`
	Status           string `json:"status"`
	SubStatus        string `json:"sub_status"`
	AssignedAgentID  string `json:"assigned_agent_id,omitempty"`
	SLAResponseDueAt string `json:"sla_response_due_at"`
	LastActivityAt   string `json:"last_activity_at"`
	UpdatedAt        string `json:"updated_at"`
}

type SearchSupportTicketsResponse struct {
	Tickets []SupportTicketResponse `json:"tickets"`
}

type AssignSupportTicketRequest struct {
	TicketID      string `json:"ticket_id"`
	AgentID       string `json:"agent_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type AssignSupportTicketResponse struct {
	SupportTicketResponse
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type UpdateSupportTicketRequest struct {
	TicketID      string `json:"ticket_id"`
	Status        string `json:"status,omitempty"`
	SubStatus     string `json:"sub_status,omitempty"`
	Priority      string `json:"priority,omitempty"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type UpdateSupportTicketResponse struct {
	SupportTicketResponse
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type SaveEditorCampaignRequest struct {
	EditorID      string `json:"editor_id"`
	CampaignID    string `json:"campaign_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type SaveEditorCampaignResponse struct {
	CampaignID          string  `json:"campaign_id"`
	Saved               bool    `json:"saved"`
	SavedAt             *string `json:"saved_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type RequestClippingExportRequest struct {
	UserID        string `json:"user_id"`
	ProjectID     string `json:"project_id"`
	Format        string `json:"format"`
	Resolution    string `json:"resolution"`
	FPS           int    `json:"fps"`
	Bitrate       string `json:"bitrate"`
	CampaignID    string `json:"campaign_id,omitempty"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RequestClippingExportResponse struct {
	ExportID            string  `json:"export_id"`
	ProjectID           string  `json:"project_id"`
	Status              string  `json:"status"`
	ProgressPercent     int     `json:"progress_percent"`
	OutputURL           string  `json:"output_url,omitempty"`
	ProviderJobID       string  `json:"provider_job_id,omitempty"`
	CreatedAt           string  `json:"created_at"`
	CompletedAt         *string `json:"completed_at,omitempty"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type DeployAutoClippingModelRequest struct {
	ModelName        string `json:"model_name"`
	VersionTag       string `json:"version_tag"`
	ModelArtifactKey string `json:"model_artifact_key"`
	CanaryPercentage int    `json:"canary_percentage"`
	Description      string `json:"description"`
	Reason           string `json:"reason"`
	SourceIP         string `json:"source_ip"`
	CorrelationID    string `json:"correlation_id"`
}

type DeployAutoClippingModelResponse struct {
	ModelVersionID      string `json:"model_version_id"`
	DeploymentStatus    string `json:"deployment_status"`
	DeployedAt          string `json:"deployed_at"`
	Message             string `json:"message"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type RotateIntegrationKeyRequest struct {
	KeyID         string `json:"key_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RotateIntegrationKeyResponse struct {
	RotationID          string `json:"rotation_id"`
	DeveloperID         string `json:"developer_id"`
	OldKeyID            string `json:"old_key_id"`
	NewKeyID            string `json:"new_key_id"`
	CreatedAt           string `json:"created_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type TestIntegrationWorkflowRequest struct {
	WorkflowID    string `json:"workflow_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type TestIntegrationWorkflowResponse struct {
	ExecutionID         string `json:"execution_id"`
	WorkflowID          string `json:"workflow_id"`
	Status              string `json:"status"`
	TestRun             bool   `json:"test_run"`
	StartedAt           string `json:"started_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type ReplayWebhookRequest struct {
	WebhookID     string `json:"webhook_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type ReplayWebhookResponse struct {
	DeliveryID          string `json:"delivery_id"`
	WebhookID           string `json:"webhook_id"`
	Status              string `json:"status"`
	HTTPStatus          int    `json:"http_status"`
	LatencyMS           int64  `json:"latency_ms"`
	Timestamp           string `json:"timestamp"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type DisableWebhookRequest struct {
	WebhookID     string `json:"webhook_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type DisableWebhookResponse struct {
	WebhookID           string `json:"webhook_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type GetWebhookDeliveriesRequest struct {
	WebhookID string `json:"webhook_id"`
	Limit     int    `json:"limit,omitempty"`
}

type WebhookDeliveryResponse struct {
	DeliveryID      string `json:"delivery_id"`
	WebhookID       string `json:"webhook_id"`
	OriginalEventID string `json:"original_event_id"`
	OriginalType    string `json:"original_event_type"`
	HTTPStatus      int    `json:"http_status"`
	LatencyMS       int64  `json:"latency_ms"`
	RetryCount      int    `json:"retry_count"`
	DeliveredAt     string `json:"delivered_at"`
	IsTest          bool   `json:"is_test"`
	Success         bool   `json:"success"`
}

type GetWebhookDeliveriesResponse struct {
	Deliveries []WebhookDeliveryResponse `json:"deliveries"`
}

type GetWebhookAnalyticsRequest struct {
	WebhookID string `json:"webhook_id"`
}

type WebhookAnalyticsMetrics struct {
	Total      int64   `json:"total"`
	Success    int64   `json:"success"`
	Failed     int64   `json:"failed"`
	AvgLatency float64 `json:"avg_latency"`
}

type GetWebhookAnalyticsResponse struct {
	TotalDeliveries      int64                              `json:"total_deliveries"`
	SuccessfulDeliveries int64                              `json:"successful_deliveries"`
	FailedDeliveries     int64                              `json:"failed_deliveries"`
	SuccessRate          float64                            `json:"success_rate"`
	AvgLatencyMS         float64                            `json:"avg_latency_ms"`
	P95LatencyMS         float64                            `json:"p95_latency_ms"`
	P99LatencyMS         float64                            `json:"p99_latency_ms"`
	ByEventType          map[string]WebhookAnalyticsMetrics `json:"by_event_type"`
}

type CreateMigrationPlanRequest struct {
	ServiceName   string                 `json:"service_name"`
	Environment   string                 `json:"environment"`
	Version       string                 `json:"version"`
	Plan          map[string]interface{} `json:"plan"`
	DryRun        bool                   `json:"dry_run"`
	RiskLevel     string                 `json:"risk_level"`
	Reason        string                 `json:"reason"`
	SourceIP      string                 `json:"source_ip"`
	CorrelationID string                 `json:"correlation_id"`
}

type CreateMigrationPlanResponse struct {
	PlanID              string                 `json:"plan_id"`
	ServiceName         string                 `json:"service_name"`
	Environment         string                 `json:"environment"`
	Version             string                 `json:"version"`
	Plan                map[string]interface{} `json:"plan"`
	Status              string                 `json:"status"`
	DryRun              bool                   `json:"dry_run"`
	RiskLevel           string                 `json:"risk_level"`
	StagingValidated    bool                   `json:"staging_validated"`
	BackupRequired      bool                   `json:"backup_required"`
	CreatedBy           string                 `json:"created_by"`
	CreatedAt           string                 `json:"created_at"`
	UpdatedAt           string                 `json:"updated_at"`
	ControlPlaneAuditID string                 `json:"control_plane_audit_id"`
}

type MigrationPlanResponse struct {
	PlanID           string                 `json:"plan_id"`
	ServiceName      string                 `json:"service_name"`
	Environment      string                 `json:"environment"`
	Version          string                 `json:"version"`
	Plan             map[string]interface{} `json:"plan"`
	Status           string                 `json:"status"`
	DryRun           bool                   `json:"dry_run"`
	RiskLevel        string                 `json:"risk_level"`
	StagingValidated bool                   `json:"staging_validated"`
	BackupRequired   bool                   `json:"backup_required"`
	CreatedBy        string                 `json:"created_by"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

type ListMigrationPlansResponse struct {
	Plans []MigrationPlanResponse `json:"plans"`
}

type StartMigrationRunRequest struct {
	PlanID        string `json:"plan_id"`
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type StartMigrationRunResponse struct {
	RunID               string `json:"run_id"`
	PlanID              string `json:"plan_id"`
	Status              string `json:"status"`
	OperatorID          string `json:"operator_id"`
	SnapshotCreated     bool   `json:"snapshot_created"`
	RollbackAvailable   bool   `json:"rollback_available"`
	ValidationStatus    string `json:"validation_status"`
	BackfillJobID       string `json:"backfill_job_id"`
	StartedAt           string `json:"started_at"`
	CompletedAt         string `json:"completed_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}
