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

type BrowseCampaignsRequest struct {
	PageSize        string
	Cursor          string
	SortBy          string
	Category        string
	BudgetMin       string
	BudgetMax       string
	DeadlineAfter   string
	DeadlineBefore  string
	Platforms       string
	State           string
	ExcludeFeatured string
}

type CampaignDTO struct {
	CampaignID       string   `json:"campaign_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	CreatorName      string   `json:"creator_name"`
	CreatorTier      string   `json:"creator_tier,omitempty"`
	BudgetTotal      float64  `json:"budget_total"`
	BudgetSpent      float64  `json:"budget_spent"`
	BudgetRemaining  float64  `json:"budget_remaining"`
	BudgetCurrency   string   `json:"budget_currency"`
	RatePer1KViews   float64  `json:"rate_per_1k_views"`
	EstimatedViews   int      `json:"estimated_views"`
	EstimatedEarning float64  `json:"estimated_earnings"`
	SubmissionCount  int      `json:"submission_count"`
	ApprovalRate     float64  `json:"approval_rate"`
	Deadline         string   `json:"deadline"`
	Category         string   `json:"category"`
	Platforms        []string `json:"platforms"`
	State            string   `json:"state"`
	IsFeatured       bool     `json:"is_featured"`
	FeaturedUntil    string   `json:"featured_until,omitempty"`
	MatchScore       float64  `json:"match_score"`
	IsEligible       bool     `json:"is_eligible"`
	Eligibility      string   `json:"eligibility_reason,omitempty"`
	UserSaved        bool     `json:"user_saved"`
	TrendingStatus   string   `json:"trending_status,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type BrowseCampaignsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Campaigns  []CampaignDTO `json:"campaigns"`
		Pagination struct {
			NextCursor     string `json:"next_cursor,omitempty"`
			PrevCursor     string `json:"prev_cursor,omitempty"`
			HasNext        bool   `json:"has_next"`
			HasPrev        bool   `json:"has_prev"`
			TotalEstimated int    `json:"total_estimated"`
			PageSize       int    `json:"page_size"`
		} `json:"pagination"`
		Summary struct {
			ResultCount  int  `json:"result_count"`
			SearchTimeMS int  `json:"search_time_ms"`
			CacheHit     bool `json:"cache_hit"`
		} `json:"summary"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SearchCampaignsRequest struct {
	Query     string
	Category  string
	BudgetMin string
	Limit     string
	Offset    string
}

type SearchResultDTO struct {
	CampaignID      string  `json:"campaign_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	CreatorName     string  `json:"creator_name"`
	MatchScore      float64 `json:"match_score"`
	Budget          float64 `json:"budget"`
	Deadline        string  `json:"deadline"`
	Category        string  `json:"category"`
	SubmissionCount int     `json:"submission_count"`
	IsFeatured      bool    `json:"is_featured"`
}

type SearchCampaignsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Results    []SearchResultDTO `json:"results"`
		Pagination struct {
			Total   int  `json:"total"`
			Limit   int  `json:"limit"`
			Offset  int  `json:"offset"`
			HasNext bool `json:"has_next"`
		} `json:"pagination"`
		SearchStats struct {
			ExecutionTimeMS int    `json:"execution_time_ms"`
			IndexVersion    string `json:"index_version"`
		} `json:"search_stats"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type CampaignDetailsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Campaign CampaignDTO `json:"campaign"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SaveBookmarkRequest struct {
	Tag  string `json:"tag"`
	Note string `json:"note"`
}

type SaveBookmarkResponse struct {
	Status string `json:"status"`
	Data   struct {
		BookmarkID string `json:"bookmark_id"`
		UserID     string `json:"user_id"`
		CampaignID string `json:"campaign_id"`
		Tag        string `json:"tag,omitempty"`
		Note       string `json:"note,omitempty"`
		CreatedAt  string `json:"created_at"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}
