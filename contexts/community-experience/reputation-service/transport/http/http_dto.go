package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LeaderboardRequest struct {
	Tier   string
	Limit  string
	Offset string
}

type ScoreComponentDTO struct {
	Value        any     `json:"value"`
	Weight       float64 `json:"weight"`
	Contribution float64 `json:"contribution"`
}

type UserReputationResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID          string `json:"user_id"`
		ReputationScore int    `json:"reputation_score"`
		Tier            string `json:"tier"`
		TierProgress    struct {
			CurrentPoints    int `json:"current_points"`
			NextTierPoints   int `json:"next_tier_points"`
			PointsToNextTier int `json:"points_to_next_tier"`
		} `json:"tier_progress"`
		PreviousScore int `json:"previous_score"`
		ScoreTrend    struct {
			WeekOverWeek   int    `json:"week_over_week"`
			MonthOverMonth int    `json:"month_over_month"`
			Direction      string `json:"direction"`
		} `json:"score_trend"`
		ScoreBreakdown struct {
			ApprovalRate        ScoreComponentDTO `json:"approval_rate"`
			ViewVelocity        ScoreComponentDTO `json:"view_velocity"`
			EarningsConsistency ScoreComponentDTO `json:"earnings_consistency"`
			SupportSatisfaction ScoreComponentDTO `json:"support_satisfaction"`
			ModerationRecord    ScoreComponentDTO `json:"moderation_record"`
			CommunitySentiment  ScoreComponentDTO `json:"community_sentiment"`
		} `json:"score_breakdown"`
		Badges []struct {
			BadgeID   string `json:"badge_id"`
			BadgeName string `json:"badge_name"`
			EarnedAt  string `json:"earned_at"`
			Category  string `json:"category"`
			Rarity    string `json:"rarity"`
			IconURL   string `json:"icon_url"`
		} `json:"badges"`
		CalculatedAt        string `json:"calculated_at"`
		NextRecalculationAt string `json:"next_recalculation_at"`
	} `json:"data"`
}

type LeaderboardResponse struct {
	Status string `json:"status"`
	Data   struct {
		Leaderboard []struct {
			Rank     int      `json:"rank"`
			UserID   string   `json:"user_id"`
			Username string   `json:"username"`
			Tier     string   `json:"tier"`
			Score    int      `json:"score"`
			Badges   []string `json:"badges"`
			Trend    string   `json:"trend"`
		} `json:"leaderboard"`
		TotalCreators int `json:"total_creators"`
		YourRank      int `json:"your_rank"`
	} `json:"data"`
}
