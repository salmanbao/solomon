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

type SummaryResponse struct {
	Status string `json:"status"`
	Data   struct {
		QuickStats struct {
			TotalViews    int     `json:"total_views"`
			TotalEarnings float64 `json:"total_earnings"`
			AverageCPV    float64 `json:"average_cpv"`
			SuccessRate   float64 `json:"success_rate"`
		} `json:"quick_stats"`
		TopClips []struct {
			ID             string  `json:"id"`
			Title          string  `json:"title"`
			ThumbnailURL   string  `json:"thumbnail_url"`
			Views          int     `json:"views"`
			Earnings       float64 `json:"earnings"`
			EngagementRate float64 `json:"engagement_rate"`
			PublishedAt    string  `json:"published_at"`
		} `json:"top_clips"`
		UpcomingPayouts []struct {
			ID     string  `json:"id"`
			Date   string  `json:"date"`
			Amount float64 `json:"amount"`
			Status string  `json:"status"`
			Method string  `json:"method"`
		} `json:"upcoming_payouts"`
		Reward struct {
			Available float64 `json:"available"`
			Pending   float64 `json:"pending"`
			Currency  string  `json:"currency"`
		} `json:"reward"`
		Gamification struct {
			Level  int      `json:"level"`
			Points int      `json:"points"`
			Badges []string `json:"badges"`
		} `json:"gamification"`
		DependencyStatus map[string]string `json:"dependency_status"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type ContentResponse struct {
	Status string `json:"status"`
	Data   struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			ID             string  `json:"id"`
			Title          string  `json:"title"`
			ThumbnailURL   string  `json:"thumbnail_url"`
			Status         string  `json:"status"`
			Views          int     `json:"views"`
			Earnings       float64 `json:"earnings"`
			EngagementRate float64 `json:"engagement_rate"`
			ClaimedAt      string  `json:"claimed_at"`
			PublishedAt    string  `json:"published_at,omitempty"`
		} `json:"items"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type CreateGoalRequest struct {
	GoalType    string  `json:"goal_type"`
	GoalName    string  `json:"goal_name"`
	TargetValue float64 `json:"target_value"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
}

type CreateGoalResponse struct {
	Status string `json:"status"`
	Data   struct {
		ID              string  `json:"id"`
		GoalType        string  `json:"goal_type"`
		GoalName        string  `json:"goal_name"`
		TargetValue     float64 `json:"target_value"`
		CurrentValue    float64 `json:"current_value"`
		ProgressPercent float64 `json:"progress_percent"`
		Status          string  `json:"status"`
		StartDate       string  `json:"start_date"`
		EndDate         string  `json:"end_date"`
		CreatedAt       string  `json:"created_at"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}
