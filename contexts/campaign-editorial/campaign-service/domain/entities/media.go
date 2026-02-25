package entities

import "time"

type MediaStatus string

const (
	MediaStatusUploading  MediaStatus = "uploading"
	MediaStatusProcessing MediaStatus = "processing"
	MediaStatusReady      MediaStatus = "ready"
	MediaStatusFailed     MediaStatus = "failed"
)

type Media struct {
	MediaID     string
	CampaignID  string
	AssetPath   string
	ContentType string
	Status      MediaStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type BudgetLog struct {
	LogID       string
	CampaignID  string
	AmountDelta float64
	Reason      string
	CreatedAt   time.Time
}

type StateHistory struct {
	HistoryID    string
	CampaignID   string
	FromState    CampaignStatus
	ToState      CampaignStatus
	ChangedBy    string
	ChangeReason string
	CreatedAt    time.Time
}
