package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateSubmissionRequest struct {
	IdempotencyKey string  `json:"idempotency_key"`
	CampaignID     string  `json:"campaign_id"`
	Platform       string  `json:"platform"`
	PostURL        string  `json:"post_url"`
	CpvRate        float64 `json:"cpv_rate"`
}

type ApproveSubmissionRequest struct {
	IdempotencyKey string `json:"-"`
	Reason         string `json:"reason"`
}

type RejectSubmissionRequest struct {
	IdempotencyKey string `json:"-"`
	Reason         string `json:"reason"`
	Notes          string `json:"notes"`
}

type ReportSubmissionRequest struct {
	IdempotencyKey string `json:"-"`
	Reason         string `json:"reason"`
	Description    string `json:"description"`
}

type BulkOperationRequest struct {
	IdempotencyKey string   `json:"-"`
	OperationType  string   `json:"operation_type"`
	SubmissionIDs  []string `json:"submission_ids"`
	ReasonCode     string   `json:"reason_code"`
	Reason         string   `json:"reason,omitempty"`
}

type BulkOperationResponse struct {
	Processed      int `json:"processed"`
	SucceededCount int `json:"succeeded_count"`
	FailedCount    int `json:"failed_count"`
}

type SubmissionDTO struct {
	SubmissionID      string `json:"submission_id"`
	CampaignID        string `json:"campaign_id"`
	CreatorID         string `json:"creator_id"`
	Platform          string `json:"platform"`
	PostURL           string `json:"post_url"`
	PostID            string `json:"post_id"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	ApprovedAt        string `json:"approved_at,omitempty"`
	ApprovedByUserID  string `json:"approved_by_user_id,omitempty"`
	ApprovalReason    string `json:"approval_reason,omitempty"`
	RejectedAt        string `json:"rejected_at,omitempty"`
	RejectionReason   string `json:"rejection_reason,omitempty"`
	RejectionNotes    string `json:"rejection_notes,omitempty"`
	ReportedCount     int    `json:"reported_count"`
	VerificationStart string `json:"verification_start,omitempty"`
	VerificationEnd   string `json:"verification_end,omitempty"`
	ViewsCount        int    `json:"views_count"`
	LockedViews       int    `json:"locked_views,omitempty"`
}

type CreateSubmissionResponse struct {
	Submission SubmissionDTO `json:"submission,omitempty"`
	Replayed   bool          `json:"replayed"`

	SubmissionID      string  `json:"submission_id,omitempty"`
	CampaignID        string  `json:"campaign_id,omitempty"`
	CreatorID         string  `json:"creator_id,omitempty"`
	Platform          string  `json:"platform,omitempty"`
	PostURL           string  `json:"post_url,omitempty"`
	Status            string  `json:"status,omitempty"`
	CreatedAt         string  `json:"created_at,omitempty"`
	CpvRate           float64 `json:"cpv_rate,omitempty"`
	EstimatedEarnings string  `json:"estimated_earnings,omitempty"`
}

type GetSubmissionResponse struct {
	Submission SubmissionDTO `json:"submission"`
}

type ListSubmissionsResponse struct {
	Items []SubmissionDTO `json:"items"`
}

type AnalyticsResponse struct {
	SubmissionID string `json:"submission_id"`
	ViewCount    int64  `json:"view_count"`
	Reported     int    `json:"reported_count"`
}

type DashboardResponse struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
	Flagged  int `json:"flagged"`
}
