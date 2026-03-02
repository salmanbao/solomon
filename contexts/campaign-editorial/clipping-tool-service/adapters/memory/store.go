package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/clipping-tool-service/domain/errors"
	"solomon/contexts/campaign-editorial/clipping-tool-service/ports"
)

type Store struct {
	mu sync.RWMutex

	projects      map[string]ports.Project
	exports       map[string]ports.ExportJob
	projectExport map[string][]string
	submissions   map[string]ports.Submission
	idempotency   map[string]ports.IdempotencyRecord
	sequence      uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	project := ports.Project{
		ProjectID:   "proj-seed-1",
		UserID:      "creator-seed",
		Title:       "Seed Project",
		Description: "Initial in-memory project",
		State:       "in_progress",
		SourceURL:   "https://cdn.whop.dev/source/seed.mp4",
		SourceType:  "url",
		Timeline: ports.Timeline{
			DurationMS: 15000,
			FPS:        30,
			Width:      1080,
			Height:     1920,
			Clips: []ports.TimelineClip{
				{
					ClipID:     "clip-seed-1",
					Type:       "video",
					StartMS:    0,
					DurationMS: 15000,
					Layer:      0,
					SourceURL:  "https://cdn.whop.dev/source/seed.mp4",
				},
			},
		},
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastSavedAt: now,
	}
	return &Store{
		projects: map[string]ports.Project{
			project.ProjectID: project,
		},
		exports:       map[string]ports.ExportJob{},
		projectExport: map[string][]string{},
		submissions:   map[string]ports.Submission{},
		idempotency:   map[string]ports.IdempotencyRecord{},
		sequence:      1,
	}
}

func (s *Store) CreateProject(ctx context.Context, input ports.CreateProjectInput, now time.Time) (ports.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID("proj")
	project := ports.Project{
		ProjectID:   id,
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		State:       "draft",
		SourceURL:   input.SourceURL,
		SourceType:  input.SourceType,
		Timeline: ports.Timeline{
			DurationMS: 0,
			FPS:        30,
			Width:      1080,
			Height:     1920,
			Clips:      []ports.TimelineClip{},
		},
		Version:     1,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
		LastSavedAt: now.UTC(),
	}
	s.projects[id] = project
	return project, nil
}

func (s *Store) GetProject(ctx context.Context, userID string, projectID string) (ports.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return ports.Project{}, domainerrors.ErrNotFound
	}
	if project.UserID != userID {
		return ports.Project{}, domainerrors.ErrForbidden
	}
	return project, nil
}

func (s *Store) UpdateTimeline(ctx context.Context, input ports.UpdateTimelineInput, now time.Time) (ports.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[input.ProjectID]
	if !ok {
		return ports.Project{}, domainerrors.ErrNotFound
	}
	if project.UserID != input.UserID {
		return ports.Project{}, domainerrors.ErrForbidden
	}
	switch input.Operation {
	case "add_clip":
		project.Timeline.Clips = append(project.Timeline.Clips, input.Clip)
		durationEnd := input.Clip.StartMS + input.Clip.DurationMS
		if durationEnd > project.Timeline.DurationMS {
			project.Timeline.DurationMS = durationEnd
		}
		project.State = "in_progress"
	case "remove_clip":
		filtered := make([]ports.TimelineClip, 0, len(project.Timeline.Clips))
		for _, clip := range project.Timeline.Clips {
			if clip.ClipID != input.Clip.ClipID {
				filtered = append(filtered, clip)
			}
		}
		project.Timeline.Clips = filtered
	}
	project.Version++
	project.UpdatedAt = now.UTC()
	project.LastSavedAt = now.UTC()
	s.projects[input.ProjectID] = project
	return project, nil
}

func (s *Store) CreateExportJob(
	ctx context.Context,
	input ports.CreateExportInput,
	provider ports.MediaExportResponse,
	now time.Time,
) (ports.ExportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[input.ProjectID]
	if !ok {
		return ports.ExportJob{}, domainerrors.ErrNotFound
	}
	if project.UserID != input.UserID {
		return ports.ExportJob{}, domainerrors.ErrForbidden
	}
	for _, exportID := range s.projectExport[input.ProjectID] {
		item := s.exports[exportID]
		if item.Status == "queued" || item.Status == "processing" {
			return ports.ExportJob{}, domainerrors.ErrProjectExporting
		}
	}
	id := s.nextID("exp")
	completed := now.UTC()
	job := ports.ExportJob{
		ExportID:        id,
		ProjectID:       input.ProjectID,
		UserID:          input.UserID,
		Status:          "completed",
		ProgressPercent: 100,
		OutputURL:       provider.OutputURL,
		ProviderJobID:   provider.ProviderJobID,
		CreatedAt:       now.UTC(),
		CompletedAt:     &completed,
	}
	s.exports[id] = job
	s.projectExport[input.ProjectID] = append(s.projectExport[input.ProjectID], id)
	return job, nil
}

func (s *Store) GetExportJob(ctx context.Context, userID string, projectID string, exportID string) (ports.ExportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.exports[exportID]
	if !ok {
		return ports.ExportJob{}, domainerrors.ErrNotFound
	}
	if job.ProjectID != projectID || job.UserID != userID {
		return ports.ExportJob{}, domainerrors.ErrForbidden
	}
	return job, nil
}

func (s *Store) ListSuggestions(ctx context.Context, userID string, projectID string) ([]ports.AISuggestion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	if !ok {
		return nil, domainerrors.ErrNotFound
	}
	if project.UserID != userID {
		return nil, domainerrors.ErrForbidden
	}
	if len(project.Timeline.Clips) == 0 {
		return []ports.AISuggestion{}, nil
	}
	first := project.Timeline.Clips[0]
	return []ports.AISuggestion{
		{
			Type:       "auto_zoom",
			ClipID:     first.ClipID,
			StartMS:    first.StartMS,
			EndMS:      first.StartMS + first.DurationMS,
			Reason:     "High motion detected",
			Confidence: 0.92,
		},
	}, nil
}

func (s *Store) CreateSubmission(ctx context.Context, input ports.CreateSubmissionInput, now time.Time) (ports.Submission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[input.ProjectID]
	if !ok {
		return ports.Submission{}, domainerrors.ErrNotFound
	}
	if project.UserID != input.UserID {
		return ports.Submission{}, domainerrors.ErrForbidden
	}
	export, ok := s.exports[input.ExportID]
	if !ok || export.ProjectID != input.ProjectID {
		return ports.Submission{}, domainerrors.ErrNotFound
	}
	if export.Status != "completed" {
		return ports.Submission{}, domainerrors.ErrConflict
	}
	id := s.nextID("sub")
	record := ports.Submission{
		SubmissionID: id,
		UserID:       input.UserID,
		ProjectID:    input.ProjectID,
		ExportID:     input.ExportID,
		CampaignID:   input.CampaignID,
		Platform:     input.Platform,
		PostURL:      input.PostURL,
		Status:       "submitted",
		CreatedAt:    now.UTC(),
	}
	s.submissions[id] = record
	return record, nil
}

func (s *Store) ValidateSource(ctx context.Context, userID string, sourceURL string, sourceType string) error {
	sourceType = strings.TrimSpace(strings.ToLower(sourceType))
	if sourceType == "" {
		sourceType = "url"
	}
	switch sourceType {
	case "url", "upload", "cloud":
	default:
		return domainerrors.ErrInvalidRequest
	}
	if strings.TrimSpace(sourceURL) == "" {
		return domainerrors.ErrInvalidRequest
	}
	return nil
}

func (s *Store) QueueExport(ctx context.Context, input ports.CreateExportInput) (ports.MediaExportResponse, error) {
	return ports.MediaExportResponse{
		ProviderJobID: s.nextID("job"),
		OutputURL:     "https://cdn.whop.dev/exports/" + s.nextID("asset") + ".mp4",
	}, nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(ctx context.Context, prefix string) (string, error) {
	return s.nextID(prefix), nil
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%d", prefix, n)
}

var _ ports.Repository = (*Store)(nil)
var _ ports.MediaProcessingClient = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
