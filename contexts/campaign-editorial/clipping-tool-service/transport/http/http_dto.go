package http

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Status    string    `json:"status"`
	Error     ErrorBody `json:"error"`
	Timestamp string    `json:"timestamp"`
}

type TimelineClipDTO struct {
	ClipID     string         `json:"clip_id"`
	Type       string         `json:"type"`
	StartMS    int            `json:"start_time_ms"`
	DurationMS int            `json:"duration_ms"`
	Layer      int            `json:"layer"`
	Text       string         `json:"text,omitempty"`
	SourceURL  string         `json:"source_url,omitempty"`
	Style      map[string]any `json:"style,omitempty"`
	Effects    map[string]any `json:"effects,omitempty"`
}

type CreateProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url"`
	SourceType  string `json:"source_type"`
}

type CreateProjectResponse struct {
	Status string `json:"status"`
	Data   struct {
		Project ProjectDTO `json:"project"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type ProjectDTO struct {
	ProjectID   string      `json:"project_id"`
	UserID      string      `json:"user_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	State       string      `json:"state"`
	SourceURL   string      `json:"source_url,omitempty"`
	SourceType  string      `json:"source_type,omitempty"`
	Timeline    TimelineDTO `json:"timeline"`
	Version     int         `json:"version"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	LastSavedAt string      `json:"last_saved_at"`
}

type TimelineDTO struct {
	DurationMS int               `json:"duration_ms"`
	FPS        int               `json:"fps"`
	Width      int               `json:"width"`
	Height     int               `json:"height"`
	Clips      []TimelineClipDTO `json:"clips"`
}

type GetProjectResponse struct {
	Status string `json:"status"`
	Data   struct {
		Project ProjectDTO `json:"project"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type UpdateTimelineRequest struct {
	Operation string          `json:"operation"`
	Clip      TimelineClipDTO `json:"clip"`
}

type UpdateTimelineResponse struct {
	Status string `json:"status"`
	Data   struct {
		Project ProjectDTO `json:"project"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type ExportRequest struct {
	Format     string `json:"format"`
	Resolution string `json:"resolution"`
	FPS        int    `json:"fps"`
	Bitrate    string `json:"bitrate"`
	CampaignID string `json:"campaign_id,omitempty"`
}

type ExportResponse struct {
	Status string `json:"status"`
	Data   struct {
		ExportID        string `json:"export_id"`
		ProjectID       string `json:"project_id"`
		Status          string `json:"status"`
		ProgressPercent int    `json:"progress_percent"`
		OutputURL       string `json:"output_url,omitempty"`
		ProviderJobID   string `json:"provider_job_id,omitempty"`
		CreatedAt       string `json:"created_at"`
		CompletedAt     string `json:"completed_at,omitempty"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type ExportStatusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ExportID        string `json:"export_id"`
		ProjectID       string `json:"project_id"`
		Status          string `json:"status"`
		ProgressPercent int    `json:"progress_percent"`
		OutputURL       string `json:"output_url,omitempty"`
		ErrorMessage    string `json:"error_message,omitempty"`
		CreatedAt       string `json:"created_at"`
		CompletedAt     string `json:"completed_at,omitempty"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SuggestionDTO struct {
	Type       string  `json:"type"`
	ClipID     string  `json:"clip_id"`
	StartMS    int     `json:"start_time_ms"`
	EndMS      int     `json:"end_time_ms"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
}

type SuggestionsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Suggestions []SuggestionDTO `json:"suggestions"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

type SubmitRequest struct {
	ExportID   string `json:"export_id"`
	CampaignID string `json:"campaign_id"`
	Platform   string `json:"platform"`
	PostURL    string `json:"post_url"`
}

type SubmitResponse struct {
	Status string `json:"status"`
	Data   struct {
		SubmissionID string `json:"submission_id"`
		ProjectID    string `json:"project_id"`
		ExportID     string `json:"export_id"`
		CampaignID   string `json:"campaign_id"`
		Platform     string `json:"platform"`
		PostURL      string `json:"post_url,omitempty"`
		Status       string `json:"status"`
		CreatedAt    string `json:"created_at"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}
