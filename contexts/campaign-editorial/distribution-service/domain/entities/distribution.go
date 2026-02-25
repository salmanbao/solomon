package entities

import "time"

type DistributionStatus string

const (
	DistributionStatusClaimed    DistributionStatus = "claimed"
	DistributionStatusScheduled  DistributionStatus = "scheduled"
	DistributionStatusPublishing DistributionStatus = "publishing"
	DistributionStatusPublished  DistributionStatus = "published"
	DistributionStatusFailed     DistributionStatus = "failed"
	DistributionStatusCancelled  DistributionStatus = "cancelled"
)

type OverlayType string

const (
	OverlayTypeIntro OverlayType = "intro"
	OverlayTypeOutro OverlayType = "outro"
)

type DistributionItem struct {
	ID                 string
	InfluencerID       string
	ClipID             string
	CampaignID         string
	Status             DistributionStatus
	ClaimedAt          time.Time
	ClaimExpiresAt     time.Time
	ScheduledForUTC    *time.Time
	Timezone           string
	Platforms          []string
	Caption            string
	Hashtags           []string
	IncludeWatermark   bool
	IncludeOverlays    bool
	PublishStartedAt   *time.Time
	PublishCompletedAt *time.Time
	LastError          string
	RetryCount         int
	PublishedAt        *time.Time
	UpdatedAt          time.Time
}

type Caption struct {
	ID                 string
	DistributionItemID string
	Platform           string
	CaptionText        string
	Hashtags           []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Overlay struct {
	ID                 string
	DistributionItemID string
	OverlayType        string
	AssetPath          string
	DurationSeconds    float64
	CreatedAt          time.Time
}

type PlatformStatus struct {
	ID                 string
	DistributionItemID string
	Platform           string
	Status             string
	PlatformPostID     string
	PlatformPostURL    string
	ErrorCode          string
	ErrorMessage       string
	RetryCount         int
	MaxRetries         int
	LastRetryAt        *time.Time
	NextRetryAt        *time.Time
	PublishedAt        *time.Time
	UpdatedAt          time.Time
}

type PublishingAnalytics struct {
	ID                   string
	DistributionItemID   string
	InfluencerID         string
	CampaignID           string
	Platform             string
	Success              bool
	Status               string
	ErrorCode            string
	ErrorMessage         string
	ClaimedAt            *time.Time
	PublishStartedAt     *time.Time
	PublishCompletedAt   *time.Time
	TimeToPublishSeconds *int
	CreatedAt            time.Time
}
