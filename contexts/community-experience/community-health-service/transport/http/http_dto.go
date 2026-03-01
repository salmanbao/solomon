package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type WebhookIngestRequest struct {
	EventID         string `json:"event_id,omitempty"`
	EventType       string `json:"event_type"`
	MessageID       string `json:"message_id"`
	ServerID        string `json:"server_id"`
	ChannelID       string `json:"channel_id"`
	UserID          string `json:"user_id"`
	Content         string `json:"content"`
	ThreadID        string `json:"thread_id,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
	EditedAt        string `json:"edited_at,omitempty"`
	DeletedAt       string `json:"deleted_at,omitempty"`
	OldContent      string `json:"old_content,omitempty"`
	NewContent      string `json:"new_content,omitempty"`
	DeletedByUserID string `json:"deleted_by_user_id,omitempty"`
}

type WebhookIngestResponse struct {
	Status string `json:"status"`
	Data   struct {
		MessageID        string  `json:"message_id"`
		EventType        string  `json:"event_type"`
		SentimentScore   float64 `json:"sentiment_score"`
		ToxicityCategory string  `json:"toxicity_category"`
		MaxSeverity      int     `json:"max_severity"`
		RiskLevel        string  `json:"risk_level"`
		AlertsGenerated  int     `json:"alerts_generated"`
		ProcessedAt      string  `json:"processed_at"`
	} `json:"data"`
}

type CommunityHealthScoreResponse struct {
	Status string `json:"status"`
	Data   struct {
		ServerID      string `json:"server_id"`
		HealthScore   int    `json:"health_score"`
		Category      string `json:"category"`
		Trend         string `json:"trend"`
		WeekStartDate string `json:"week_start_date"`
		Breakdown     struct {
			SentimentHealth  int `json:"sentiment_health"`
			ToxicityHealth   int `json:"toxicity_health"`
			EngagementHealth int `json:"engagement_health"`
			LatencyHealth    int `json:"latency_health"`
			TrendBonus       int `json:"trend_bonus"`
		} `json:"breakdown"`
		Metrics struct {
			TotalMessages             int     `json:"total_messages"`
			PositivePct               float64 `json:"positive_pct"`
			ToxicityPct               float64 `json:"toxicity_pct"`
			EngagementGini            float64 `json:"engagement_gini"`
			AvgModerationLatencyHours float64 `json:"avg_moderation_latency_hours"`
		} `json:"metrics"`
		Alerts       int    `json:"alerts"`
		CalculatedAt string `json:"calculated_at"`
	} `json:"data"`
}

type UserRiskScoreResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID             string   `json:"user_id"`
		ServerID           string   `json:"server_id"`
		RiskScore          float64  `json:"risk_score"`
		RiskLevel          string   `json:"risk_level"`
		ToxicMessageCount  int      `json:"toxic_message_count"`
		WarningCount       int      `json:"warning_count"`
		BanCount           int      `json:"ban_count"`
		LastToxicMessageAt string   `json:"last_toxic_message_at,omitempty"`
		Recommendations    []string `json:"recommendations"`
	} `json:"data"`
}
