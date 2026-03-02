package ports

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context, prefix string) (string, error)
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

type TimelineClip struct {
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

type Timeline struct {
	DurationMS int            `json:"duration_ms"`
	FPS        int            `json:"fps"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Clips      []TimelineClip `json:"clips"`
}

type Project struct {
	ProjectID   string
	UserID      string
	Title       string
	Description string
	State       string
	SourceURL   string
	SourceType  string
	Timeline    Timeline
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastSavedAt time.Time
}

type CreateProjectInput struct {
	UserID      string
	Title       string
	Description string
	SourceURL   string
	SourceType  string
}

type UpdateTimelineInput struct {
	UserID    string
	ProjectID string
	Operation string
	Clip      TimelineClip
}

type ExportSettings struct {
	Format     string
	Resolution string
	FPS        int
	Bitrate    string
	CampaignID string
}

type CreateExportInput struct {
	UserID    string
	ProjectID string
	Settings  ExportSettings
}

type ExportJob struct {
	ExportID        string
	ProjectID       string
	UserID          string
	Status          string
	ProgressPercent int
	OutputURL       string
	ProviderJobID   string
	ErrorMessage    string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

type AISuggestion struct {
	Type       string
	ClipID     string
	StartMS    int
	EndMS      int
	Reason     string
	Confidence float64
}

type CreateSubmissionInput struct {
	UserID     string
	ProjectID  string
	ExportID   string
	CampaignID string
	Platform   string
	PostURL    string
}

type Submission struct {
	SubmissionID string
	UserID       string
	ProjectID    string
	ExportID     string
	CampaignID   string
	Platform     string
	PostURL      string
	Status       string
	CreatedAt    time.Time
}

type MediaExportResponse struct {
	ProviderJobID string
	OutputURL     string
}

type MediaProcessingClient interface {
	ValidateSource(ctx context.Context, userID string, sourceURL string, sourceType string) error
	QueueExport(ctx context.Context, input CreateExportInput) (MediaExportResponse, error)
}

type Repository interface {
	CreateProject(ctx context.Context, input CreateProjectInput, now time.Time) (Project, error)
	GetProject(ctx context.Context, userID string, projectID string) (Project, error)
	UpdateTimeline(ctx context.Context, input UpdateTimelineInput, now time.Time) (Project, error)
	CreateExportJob(ctx context.Context, input CreateExportInput, provider MediaExportResponse, now time.Time) (ExportJob, error)
	GetExportJob(ctx context.Context, userID string, projectID string, exportID string) (ExportJob, error)
	ListSuggestions(ctx context.Context, userID string, projectID string) ([]AISuggestion, error)
	CreateSubmission(ctx context.Context, input CreateSubmissionInput, now time.Time) (Submission, error)
}
