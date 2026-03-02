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

type FeedItemDTO struct {
	CampaignID       string  `json:"campaign_id"`
	Title            string  `json:"title"`
	Category         string  `json:"category"`
	RewardRate       float64 `json:"reward_rate"`
	BudgetRemaining  float64 `json:"budget_remaining"`
	MatchScore       float64 `json:"match_score"`
	SubmissionStatus string  `json:"submission_status,omitempty"`
	Saved            bool    `json:"saved"`
}

type FeedResponse struct {
	Status string `json:"status"`
	Data   struct {
		Items []FeedItemDTO `json:"items"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SubmissionDTO struct {
	SubmissionID   string  `json:"submission_id"`
	CampaignID     string  `json:"campaign_id"`
	CampaignTitle  string  `json:"campaign_title"`
	Status         string  `json:"status"`
	Views          int     `json:"views"`
	Earnings       float64 `json:"earnings"`
	SubmittedAt    string  `json:"submitted_at"`
	ReviewedAt     string  `json:"reviewed_at,omitempty"`
	Feedback       string  `json:"feedback,omitempty"`
	RejectionCode  string  `json:"rejection_code,omitempty"`
	ModerationNote string  `json:"moderation_note,omitempty"`
}

type SubmissionsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Items []SubmissionDTO `json:"items"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type EarningsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Available float64 `json:"available"`
		Pending   float64 `json:"pending"`
		Lifetime  float64 `json:"lifetime"`
		Currency  string  `json:"currency"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type PerformanceResponse struct {
	Status string `json:"status"`
	Data   struct {
		ApprovalRate        float64 `json:"approval_rate"`
		AvgViewsPerClip     float64 `json:"avg_views_per_clip"`
		ReputationScore     float64 `json:"reputation_score"`
		BenchmarkPercentile float64 `json:"benchmark_percentile"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SaveCampaignResponse struct {
	Status string `json:"status"`
	Data   struct {
		CampaignID string `json:"campaign_id"`
		Saved      bool   `json:"saved"`
		SavedAt    string `json:"saved_at,omitempty"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type ExportSubmissionsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}
