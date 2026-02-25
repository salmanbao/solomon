package httptransport

type ListClipsRequest struct {
	Niche          []string `json:"niche,omitempty"`
	DurationBucket string   `json:"duration_bucket,omitempty"`
	PopularitySort string   `json:"popularity_sort,omitempty"`
	Status         string   `json:"status,omitempty"`
	Cursor         string   `json:"cursor,omitempty"`
	Limit          int      `json:"limit,omitempty"`
}

type ClipStatsDTO struct {
	Views7d        int     `json:"views_7d"`
	Votes7d        int     `json:"votes_7d"`
	EngagementRate float64 `json:"engagement_rate"`
}

type ClipDTO struct {
	ClipID          string       `json:"clip_id"`
	Title           string       `json:"title"`
	Niche           string       `json:"niche"`
	DurationSeconds int          `json:"duration_seconds"`
	PreviewURL      string       `json:"preview_url"`
	Exclusivity     string       `json:"exclusivity"`
	ClaimLimit      int          `json:"claim_limit"`
	Stats           ClipStatsDTO `json:"stats"`
}

type ListClipsResponse struct {
	Items      []ClipDTO `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type GetClipResponse struct {
	Item ClipDTO `json:"item"`
}

type GetClipPreviewResponse struct {
	ClipID     string `json:"clip_id"`
	PreviewURL string `json:"preview_url"`
	ExpiresAt  string `json:"expires_at"`
}

type ClaimClipRequest struct {
	RequestID string `json:"request_id"`
}

type ClaimClipResponse struct {
	ClaimID   string `json:"claim_id"`
	ClipID    string `json:"clip_id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	Replayed  bool   `json:"replayed,omitempty"`
}

type ClaimDTO struct {
	ClaimID   string `json:"claim_id"`
	ClipID    string `json:"clip_id"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
}

type ListClaimsResponse struct {
	Items []ClaimDTO `json:"items"`
}

type DownloadClipResponse struct {
	DownloadURL        string `json:"download_url"`
	ExpiresAt          string `json:"expires_at"`
	RemainingDownloads int    `json:"remaining_downloads"`
	Replayed           bool   `json:"replayed,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
