package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateCampaignRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Instructions     string   `json:"instructions"`
	Niche            string   `json:"niche"`
	AllowedPlatforms []string `json:"allowed_platforms"`
	RequiredHashtags []string `json:"required_hashtags"`
	BudgetTotal      float64  `json:"budget_total"`
	RatePer1KViews   float64  `json:"rate_per_1k_views"`
}

type UpdateCampaignRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Instructions     string   `json:"instructions"`
	Niche            string   `json:"niche"`
	AllowedPlatforms []string `json:"allowed_platforms"`
	RequiredHashtags []string `json:"required_hashtags"`
}

type IncreaseBudgetRequest struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

type StatusActionRequest struct {
	Reason string `json:"reason"`
}

type GenerateUploadURLRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

type GenerateUploadURLResponse struct {
	MediaID   string `json:"media_id"`
	UploadURL string `json:"upload_url"`
	ExpiresAt string `json:"expires_at"`
	AssetPath string `json:"asset_path"`
}

type ConfirmMediaRequest struct {
	AssetPath   string `json:"asset_path"`
	ContentType string `json:"content_type"`
}

type CampaignDTO struct {
	CampaignID       string   `json:"campaign_id"`
	BrandID          string   `json:"brand_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Instructions     string   `json:"instructions"`
	Niche            string   `json:"niche"`
	AllowedPlatforms []string `json:"allowed_platforms"`
	RequiredHashtags []string `json:"required_hashtags"`
	BudgetTotal      float64  `json:"budget_total"`
	BudgetSpent      float64  `json:"budget_spent"`
	BudgetRemaining  float64  `json:"budget_remaining"`
	RatePer1KViews   float64  `json:"rate_per_1k_views"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type CreateCampaignResponse struct {
	Campaign CampaignDTO `json:"campaign"`
	Replayed bool        `json:"replayed"`
}

type ListCampaignsResponse struct {
	Items []CampaignDTO `json:"items"`
}

type GetCampaignResponse struct {
	Campaign CampaignDTO `json:"campaign"`
}

type CampaignMediaDTO struct {
	MediaID     string `json:"media_id"`
	CampaignID  string `json:"campaign_id"`
	AssetPath   string `json:"asset_path"`
	ContentType string `json:"content_type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ListMediaResponse struct {
	Items []CampaignMediaDTO `json:"items"`
}

type AnalyticsResponse struct {
	CampaignID      string  `json:"campaign_id"`
	SubmissionCount int     `json:"submission_count"`
	TotalViews      int64   `json:"total_views"`
	BudgetSpent     float64 `json:"budget_spent"`
	BudgetRemaining float64 `json:"budget_remaining"`
}

type ExportAnalyticsResponse struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
}
