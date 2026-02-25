package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateCampaignRequest struct {
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Instructions      string   `json:"instructions"`
	Niche             string   `json:"niche"`
	AllowedPlatforms  []string `json:"allowed_platforms"`
	RequiredHashtags  []string `json:"required_hashtags"`
	RequiredTags      []string `json:"required_tags"`
	OptionalHashtags  []string `json:"optional_hashtags"`
	UsageGuidelines   string   `json:"usage_guidelines"`
	DosAndDonts       string   `json:"dos_and_donts"`
	CampaignType      string   `json:"campaign_type"`
	Deadline          string   `json:"deadline"`
	TargetSubmissions *int     `json:"target_submissions"`
	BannerImageURL    string   `json:"banner_image_url"`
	ExternalURL       string   `json:"external_url"`
	BudgetTotal       float64  `json:"budget_total"`
	RatePer1KViews    float64  `json:"rate_per_1k_views"`
}

type UpdateCampaignRequest struct {
	Title             *string   `json:"title"`
	Description       *string   `json:"description"`
	Instructions      *string   `json:"instructions"`
	Niche             *string   `json:"niche"`
	AllowedPlatforms  *[]string `json:"allowed_platforms"`
	RequiredHashtags  *[]string `json:"required_hashtags"`
	RequiredTags      *[]string `json:"required_tags"`
	OptionalHashtags  *[]string `json:"optional_hashtags"`
	UsageGuidelines   *string   `json:"usage_guidelines"`
	DosAndDonts       *string   `json:"dos_and_donts"`
	CampaignType      *string   `json:"campaign_type"`
	Deadline          *string   `json:"deadline"`
	TargetSubmissions *int      `json:"target_submissions"`
	BannerImageURL    *string   `json:"banner_image_url"`
	ExternalURL       *string   `json:"external_url"`
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
	FileSize    int64  `json:"file_size"`
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
	CampaignID              string   `json:"campaign_id"`
	BrandID                 string   `json:"brand_id"`
	Title                   string   `json:"title"`
	Description             string   `json:"description"`
	Instructions            string   `json:"instructions"`
	Niche                   string   `json:"niche"`
	CampaignType            string   `json:"campaign_type"`
	AllowedPlatforms        []string `json:"allowed_platforms"`
	RequiredHashtags        []string `json:"required_hashtags"`
	RequiredTags            []string `json:"required_tags"`
	OptionalHashtags        []string `json:"optional_hashtags"`
	UsageGuidelines         string   `json:"usage_guidelines"`
	DosAndDonts             string   `json:"dos_and_donts"`
	Deadline                string   `json:"deadline,omitempty"`
	TargetSubmissions       *int     `json:"target_submissions,omitempty"`
	BannerImageURL          string   `json:"banner_image_url,omitempty"`
	ExternalURL             string   `json:"external_url,omitempty"`
	BudgetTotal             float64  `json:"budget_total"`
	BudgetSpent             float64  `json:"budget_spent"`
	BudgetReserved          float64  `json:"budget_reserved"`
	BudgetRemaining         float64  `json:"budget_remaining"`
	RatePer1KViews          float64  `json:"rate_per_1k_views"`
	SubmissionCount         int      `json:"submission_count"`
	ApprovedSubmissionCount int      `json:"approved_submission_count"`
	TotalViews              int64    `json:"total_views"`
	Status                  string   `json:"status"`
	LaunchedAt              string   `json:"launched_at,omitempty"`
	CompletedAt             string   `json:"completed_at,omitempty"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
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
