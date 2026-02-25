package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AddOverlayRequest struct {
	OverlayType     string  `json:"overlay_type"`
	AssetPath       string  `json:"asset_path"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type ScheduleRequest struct {
	Platform     string `json:"platform"`
	ScheduledFor string `json:"scheduled_for"`
	Timezone     string `json:"timezone"`
}

type DownloadRequest struct {
	Platform        string `json:"platform"`
	IncludeOverlays bool   `json:"include_overlays"`
}

type PublishMultiRequest struct {
	Platforms []string `json:"platforms"`
	Caption   string   `json:"caption"`
}

type DistributionItemDTO struct {
	ID              string   `json:"id"`
	InfluencerID    string   `json:"influencer_id"`
	ClipID          string   `json:"clip_id"`
	CampaignID      string   `json:"campaign_id"`
	Status          string   `json:"status"`
	ClaimExpiresAt  string   `json:"claim_expires_at"`
	ScheduledForUTC string   `json:"scheduled_for_utc,omitempty"`
	Timezone        string   `json:"timezone,omitempty"`
	Platforms       []string `json:"platforms"`
	Caption         string   `json:"caption,omitempty"`
	RetryCount      int      `json:"retry_count"`
	LastError       string   `json:"last_error,omitempty"`
}

type PreviewResponse struct {
	ID         string `json:"id"`
	PreviewURL string `json:"preview_url"`
	ExpiresAt  string `json:"expires_at"`
}

type DownloadResponse struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
}
