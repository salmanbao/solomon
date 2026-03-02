package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AwardPointsRequest struct {
	UserID     string `json:"user_id"`
	ActionType string `json:"action_type"`
	Points     int    `json:"points"`
	Reason     string `json:"reason,omitempty"`
}

type AwardPointsResponse struct {
	Status   string `json:"status"`
	Replayed bool   `json:"replayed,omitempty"`
	Data     struct {
		UserID       string  `json:"user_id"`
		ActionType   string  `json:"action_type"`
		BasePoints   int     `json:"base_points"`
		Multiplier   float64 `json:"multiplier"`
		FinalPoints  int     `json:"final_points"`
		TotalPoints  int     `json:"total_points"`
		CurrentLevel int     `json:"current_level"`
		GrantedAt    string  `json:"granted_at"`
	} `json:"data"`
}

type GrantBadgeRequest struct {
	UserID   string `json:"user_id"`
	BadgeKey string `json:"badge_key"`
	Reason   string `json:"reason,omitempty"`
}

type GrantBadgeResponse struct {
	Status   string `json:"status"`
	Replayed bool   `json:"replayed,omitempty"`
	Data     struct {
		BadgeID   string `json:"badge_id"`
		UserID    string `json:"user_id"`
		BadgeKey  string `json:"badge_key"`
		Reason    string `json:"reason,omitempty"`
		GrantedAt string `json:"granted_at"`
	} `json:"data"`
}

type UserSummaryResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID         string   `json:"user_id"`
		TotalPoints    int      `json:"total_points"`
		CurrentLevel   int      `json:"current_level"`
		ReputationTier string   `json:"reputation_tier"`
		Badges         []string `json:"badges"`
	} `json:"data"`
}

type LeaderboardEntryDTO struct {
	Rank         int    `json:"rank"`
	UserID       string `json:"user_id"`
	TotalPoints  int    `json:"total_points"`
	CurrentLevel int    `json:"current_level"`
}

type LeaderboardResponse struct {
	Status string                `json:"status"`
	Data   []LeaderboardEntryDTO `json:"data"`
}
