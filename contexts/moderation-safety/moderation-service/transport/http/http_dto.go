package http

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Status    string    `json:"status"`
	Error     ErrorBody `json:"error"`
	Timestamp string    `json:"timestamp"`
}

type ApproveRequest struct {
	SubmissionID string `json:"submission_id"`
	CampaignID   string `json:"campaign_id"`
	Reason       string `json:"reason"`
	Notes        string `json:"notes,omitempty"`
}

type RejectRequest struct {
	SubmissionID    string `json:"submission_id"`
	CampaignID      string `json:"campaign_id"`
	RejectionReason string `json:"rejection_reason"`
	RejectionNotes  string `json:"rejection_notes,omitempty"`
}

type FlagRequest struct {
	SubmissionID string `json:"submission_id"`
	CampaignID   string `json:"campaign_id"`
	FlagReason   string `json:"flag_reason"`
	Severity     string `json:"severity,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type DecisionResponse struct {
	Status string `json:"status"`
	Data   struct {
		DecisionID   string `json:"decision_id"`
		SubmissionID string `json:"submission_id"`
		CampaignID   string `json:"campaign_id"`
		ModeratorID  string `json:"moderator_id"`
		Action       string `json:"action"`
		Reason       string `json:"reason"`
		Notes        string `json:"notes,omitempty"`
		Severity     string `json:"severity,omitempty"`
		QueueStatus  string `json:"queue_status"`
		CreatedAt    string `json:"created_at"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type QueueResponse struct {
	Status string `json:"status"`
	Data   struct {
		Items []struct {
			SubmissionID        string  `json:"submission_id"`
			CampaignID          string  `json:"campaign_id"`
			CreatorID           string  `json:"creator_id"`
			Status              string  `json:"status"`
			RiskScore           float64 `json:"risk_score"`
			ReportCount         int     `json:"report_count"`
			QueuedAt            string  `json:"queued_at"`
			AssignedModeratorID string  `json:"assigned_moderator_id,omitempty"`
		} `json:"items"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}
