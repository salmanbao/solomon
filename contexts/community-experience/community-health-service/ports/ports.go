package ports

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type WebhookIngestInput struct {
	EventID         string
	EventType       string
	MessageID       string
	ServerID        string
	ChannelID       string
	UserID          string
	Content         string
	ThreadID        string
	CreatedAt       *time.Time
	EditedAt        *time.Time
	DeletedAt       *time.Time
	OldContent      string
	NewContent      string
	DeletedByUserID string
}

type MessageSentiment struct {
	MessageID      string
	ServerID       string
	ChannelID      string
	UserID         string
	SentimentScore float64
	Confidence     float64
	Category       string
	Language       string
	EmojiAdjusted  bool
	SarcasmFlag    bool
	AnalyzedAt     time.Time
}

type MessageToxicity struct {
	MessageID           string
	ServerID            string
	ChannelID           string
	UserID              string
	HateSpeechScore     float64
	HarassmentScore     float64
	ThreatsScore        float64
	SexualScore         float64
	SpamScore           float64
	MisinformationScore float64
	MaxSeverity         int
	PrimaryCategory     string
	AnalyzedAt          time.Time
}

type UserRiskScore struct {
	ServerID          string
	UserID            string
	RiskScore         float64
	RiskLevel         string
	ToxicMessageCount int
	WarningCount      int
	BanCount          int
	LastToxicAt       *time.Time
	Recommendations   []string
	LastRecalculated  time.Time
}

type CommunityHealthScore struct {
	ScoreID                string
	ServerID               string
	WeekStartDate          string
	HealthScore            int
	Category               string
	Trend                  string
	SentimentHealth        int
	ToxicityHealth         int
	EngagementHealth       int
	LatencyHealth          int
	TrendBonus             int
	TotalMessages          int
	PositivePct            float64
	ToxicityPct            float64
	EngagementGini         float64
	AvgModerationLatencyHr float64
	Alerts                 int
	CalculatedAt           time.Time
}

type RealTimeAlert struct {
	AlertID      string
	AlertType    string
	ServerID     string
	ChannelID    string
	Severity     string
	TriggeredAt  time.Time
	Status       string
	ResponseHint string
}

type WeeklyHealthReport struct {
	ReportID      string
	ServerID      string
	WeekStartDate string
	MetricsJSON   map[string]any
	GeneratedAt   time.Time
}

type ModerationFeedback struct {
	FeedbackID   string
	MessageID    string
	ModeratorID  string
	FeedbackType string
	CreatedAt    time.Time
}

type IngestionResult struct {
	MessageID        string
	EventType        string
	SentimentScore   float64
	ToxicityCategory string
	MaxSeverity      int
	RiskLevel        string
	AlertsGenerated  int
	ProcessedAt      time.Time
}

type Repository interface {
	IngestWebhook(ctx context.Context, input WebhookIngestInput, now time.Time) (IngestionResult, error)
	GetCommunityHealthScore(ctx context.Context, serverID string) (CommunityHealthScore, error)
	GetUserRiskScore(ctx context.Context, serverID string, userID string) (UserRiskScore, error)
}
