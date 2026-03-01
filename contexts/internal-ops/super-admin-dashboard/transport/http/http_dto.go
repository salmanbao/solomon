package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StartImpersonationRequest struct {
	ImpersonatedUserID string `json:"impersonated_user_id"`
	Reason             string `json:"reason"`
	Notes              string `json:"notes,omitempty"`
}

type StartImpersonationResponse struct {
	ImpersonationID string `json:"impersonation_id"`
	UserID          string `json:"user_id"`
	AccessToken     string `json:"access_token"`
	TokenExpiresAt  string `json:"token_expires_at"`
	StartedAt       string `json:"started_at"`
	Status          string `json:"status"`
	Replayed        bool   `json:"replayed,omitempty"`
}

type EndImpersonationRequest struct {
	ImpersonationID string `json:"impersonation_id"`
}

type EndImpersonationResponse struct {
	EndedAt          string `json:"ended_at"`
	Status           string `json:"status"`
	ActivitySummary  struct {
		ActionsLogged   int `json:"actions_logged"`
		DurationMinutes int `json:"duration_minutes"`
	} `json:"activity_summary"`
	Replayed bool `json:"replayed,omitempty"`
}

type WalletAdjustRequest struct {
	Amount         float64 `json:"amount"`
	AdjustmentType string  `json:"adjustment_type"`
	Reason         string  `json:"reason"`
}

type WalletAdjustResponse struct {
	AdjustmentID  string  `json:"adjustment_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	BalanceBefore float64 `json:"balance_before"`
	BalanceAfter  float64 `json:"balance_after"`
	AdjustedAt    string  `json:"adjusted_at"`
	AuditLogID    string  `json:"audit_log_id"`
	Replayed      bool    `json:"replayed,omitempty"`
}

type WalletHistoryEntry struct {
	AdjustmentID string  `json:"adjustment_id"`
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"`
	Reason       string  `json:"reason"`
	AdminID      string  `json:"admin_id"`
	AdjustedAt   string  `json:"adjusted_at"`
}

type WalletHistoryResponse struct {
	Adjustments []WalletHistoryEntry `json:"adjustments"`
	Pagination  struct {
		Cursor   string `json:"cursor,omitempty"`
		HasMore  bool   `json:"has_more"`
		PageSize int    `json:"page_size"`
	} `json:"pagination"`
}

type BanUserRequest struct {
	BanType      string `json:"ban_type"`
	DurationDays int    `json:"duration_days,omitempty"`
	Reason       string `json:"reason"`
}

type BanUserResponse struct {
	BanID              string `json:"ban_id"`
	UserID             string `json:"user_id"`
	BanType            string `json:"ban_type"`
	BannedAt           string `json:"banned_at"`
	ExpiresAt          string `json:"expires_at,omitempty"`
	AllSessionsRevoked bool   `json:"all_sessions_revoked"`
	AuditLogID         string `json:"audit_log_id"`
	Replayed           bool   `json:"replayed,omitempty"`
}

type UnbanUserRequest struct {
	Reason string `json:"reason"`
}

type UnbanUserResponse struct {
	UserID    string `json:"user_id"`
	UnbannedAt string `json:"unbanned_at"`
	Status    string `json:"status"`
	Replayed  bool   `json:"replayed,omitempty"`
}

type UserSearchResponse struct {
	Users []struct {
		UserID         string  `json:"user_id"`
		Email          string  `json:"email"`
		Username       string  `json:"username"`
		Role           string  `json:"role"`
		CreatedAt      string  `json:"created_at"`
		TotalEarnings  float64 `json:"total_earnings"`
		Status         string  `json:"status"`
		KYCStatus      string  `json:"kyc_status"`
		LastLoginAt    string  `json:"last_login_at,omitempty"`
	} `json:"users"`
	Pagination struct {
		Cursor     string `json:"cursor,omitempty"`
		HasMore    bool   `json:"has_more"`
		TotalCount int    `json:"total_count"`
	} `json:"pagination"`
}

type BulkActionRequest struct {
	UserIDs      []string       `json:"user_ids"`
	Action       string         `json:"action"`
	ActionParams map[string]any `json:"action_params,omitempty"`
}

type BulkActionResponse struct {
	JobID                    string `json:"job_id"`
	Action                   string `json:"action"`
	UserCount                int    `json:"user_count"`
	Status                   string `json:"status"`
	CreatedAt                string `json:"created_at"`
	EstimatedCompletionTime  string `json:"estimated_completion_time"`
	Replayed                 bool   `json:"replayed,omitempty"`
}

type PauseCampaignRequest struct {
	Reason string `json:"reason"`
}

type PauseCampaignResponse struct {
	CampaignID string `json:"campaign_id"`
	Status     string `json:"status"`
	PausedAt   string `json:"paused_at"`
	AuditLogID string `json:"audit_log_id"`
	Replayed   bool   `json:"replayed,omitempty"`
}

type AdjustCampaignRequest struct {
	NewBudget         float64 `json:"new_budget"`
	NewRatePer1kViews float64 `json:"new_rate_per_1k_views"`
	Reason            string  `json:"reason"`
}

type AdjustCampaignResponse struct {
	CampaignID         string  `json:"campaign_id"`
	OldBudget          float64 `json:"old_budget"`
	NewBudget          float64 `json:"new_budget"`
	OldRatePer1kViews  float64 `json:"old_rate_per_1k_views"`
	NewRatePer1kViews  float64 `json:"new_rate_per_1k_views"`
	AdjustedAt         string  `json:"adjusted_at"`
	AuditLogID         string  `json:"audit_log_id"`
	Replayed           bool    `json:"replayed,omitempty"`
}

type OverrideSubmissionRequest struct {
	NewStatus      string `json:"new_status"`
	OverrideReason string `json:"override_reason"`
}

type OverrideSubmissionResponse struct {
	SubmissionID string `json:"submission_id"`
	OldStatus    string `json:"old_status"`
	NewStatus    string `json:"new_status"`
	OverriddenAt string `json:"overridden_at"`
	AuditLogID   string `json:"audit_log_id"`
	Replayed     bool   `json:"replayed,omitempty"`
}

type FeatureFlagDTO struct {
	FlagKey   string         `json:"flag_key"`
	Enabled   bool           `json:"enabled"`
	Config    map[string]any `json:"config,omitempty"`
	UpdatedAt string         `json:"updated_at"`
	UpdatedBy string         `json:"updated_by"`
}

type FeatureFlagsResponse struct {
	Flags []FeatureFlagDTO `json:"flags"`
}

type ToggleFeatureFlagRequest struct {
	Enabled bool           `json:"enabled"`
	Reason  string         `json:"reason"`
	Config  map[string]any `json:"config,omitempty"`
}

type ToggleFeatureFlagResponse struct {
	FlagKey              string   `json:"flag_key"`
	OldEnabled           bool     `json:"old_enabled"`
	NewEnabled           bool     `json:"new_enabled"`
	UpdatedAt            string   `json:"updated_at"`
	PropagatedToServices []string `json:"propagated_to_services"`
	Replayed             bool     `json:"replayed,omitempty"`
}

type AnalyticsDashboardResponse struct {
	DateRange struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"date_range"`
	Metrics struct {
		TotalRevenue  float64 `json:"total_revenue"`
		UserGrowth    int     `json:"user_growth"`
		CampaignCount int     `json:"campaign_count"`
		FraudMetrics  int     `json:"fraud_metrics"`
	} `json:"metrics"`
}

type AuditLogDTO struct {
	AuditID            string         `json:"audit_id"`
	AdminID            string         `json:"admin_id"`
	ActionType         string         `json:"action_type"`
	TargetResourceID   string         `json:"target_resource_id"`
	TargetResourceType string         `json:"target_resource_type"`
	OldValue           map[string]any `json:"old_value,omitempty"`
	NewValue           map[string]any `json:"new_value,omitempty"`
	Reason             string         `json:"reason"`
	PerformedAt        string         `json:"performed_at"`
	IPAddress          string         `json:"ip_address"`
	SignatureHash      string         `json:"signature_hash"`
	IsVerified         bool           `json:"is_verified"`
}

type AuditLogsResponse struct {
	AuditLogs  []AuditLogDTO `json:"audit_logs"`
	Pagination struct {
		Cursor  string `json:"cursor,omitempty"`
		HasMore bool   `json:"has_more"`
	} `json:"pagination"`
}

type AuditLogExportResponse struct {
	ExportJobID          string `json:"export_job_id"`
	Status               string `json:"status"`
	FileURL              string `json:"file_url"`
	CreatedAt            string `json:"created_at"`
	EstimatedCompletion  string `json:"estimated_completion"`
}