package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateSubmissionRequest struct {
	CampaignID string `json:"campaign_id"`
	Platform   string `json:"platform"`
	PostURL    string `json:"post_url"`
}

type ApproveSubmissionRequest struct {
	Reason string `json:"reason"`
}

type RejectSubmissionRequest struct {
	Reason string `json:"reason"`
	Notes  string `json:"notes"`
}

type ReportSubmissionRequest struct {
	Reason      string `json:"reason"`
	Description string `json:"description"`
}

type BulkOperationRequest struct {
	OperationType string   `json:"operation_type"`
	SubmissionIDs []string `json:"submission_ids"`
	Reason        string   `json:"reason"`
}

type SubmissionDTO struct {
	SubmissionID     string `json:"submission_id"`
	CampaignID       string `json:"campaign_id"`
	CreatorID        string `json:"creator_id"`
	Platform         string `json:"platform"`
	PostURL          string `json:"post_url"`
	Status           string `json:"status"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	ApprovedAt       string `json:"approved_at,omitempty"`
	ApprovedByUserID string `json:"approved_by_user_id,omitempty"`
	ApprovalReason   string `json:"approval_reason,omitempty"`
	RejectedAt       string `json:"rejected_at,omitempty"`
	RejectionReason  string `json:"rejection_reason,omitempty"`
	RejectionNotes   string `json:"rejection_notes,omitempty"`
	ReportedCount    int    `json:"reported_count"`
}

type CreateSubmissionResponse struct {
	Submission SubmissionDTO `json:"submission"`
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
