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

type DistributionItem struct {
	ID              string
	InfluencerID    string
	ClipID          string
	CampaignID      string
	Status          DistributionStatus
	ClaimedAt       time.Time
	ClaimExpiresAt  time.Time
	ScheduledForUTC *time.Time
	Timezone        string
	Platforms       []string
	Caption         string
	LastError       string
	RetryCount      int
	PublishedAt     *time.Time
	UpdatedAt       time.Time
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
	PlatformPostURL    string
	ErrorMessage       string
	RetryCount         int
	UpdatedAt          time.Time
}
