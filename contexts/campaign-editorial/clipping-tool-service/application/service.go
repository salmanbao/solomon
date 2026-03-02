package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/clipping-tool-service/domain/errors"
	"solomon/contexts/campaign-editorial/clipping-tool-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	MediaClient    ports.MediaProcessingClient
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func (s Service) CreateProject(
	ctx context.Context,
	idempotencyKey string,
	input ports.CreateProjectInput,
) (ports.Project, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.UserID = strings.TrimSpace(input.UserID)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.SourceURL = strings.TrimSpace(input.SourceURL)
	input.SourceType = strings.TrimSpace(strings.ToLower(input.SourceType))

	if input.UserID == "" || input.Title == "" {
		return ports.Project{}, domainerrors.ErrInvalidRequest
	}
	if idempotencyKey == "" {
		return ports.Project{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if input.SourceType == "" {
		input.SourceType = "url"
	}
	if s.MediaClient != nil {
		if err := s.MediaClient.ValidateSource(ctx, input.UserID, input.SourceURL, input.SourceType); err != nil {
			return ports.Project{}, domainerrors.ErrDependencyUnavailable
		}
	}

	requestHash := hashStrings(input.UserID, input.Title, input.Description, input.SourceURL, input.SourceType)
	var output ports.Project
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			project, err := s.Repo.CreateProject(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(project)
		},
	)
	return output, err
}

func (s Service) GetProject(ctx context.Context, userID string, projectID string) (ports.Project, error) {
	userID = strings.TrimSpace(userID)
	projectID = strings.TrimSpace(projectID)
	if userID == "" || projectID == "" {
		return ports.Project{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetProject(ctx, userID, projectID)
}

func (s Service) UpdateTimeline(
	ctx context.Context,
	idempotencyKey string,
	input ports.UpdateTimelineInput,
) (ports.Project, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Operation = strings.TrimSpace(strings.ToLower(input.Operation))
	input.Clip.ClipID = strings.TrimSpace(input.Clip.ClipID)
	input.Clip.Type = strings.TrimSpace(strings.ToLower(input.Clip.Type))

	if input.UserID == "" || input.ProjectID == "" || input.Operation == "" {
		return ports.Project{}, domainerrors.ErrInvalidRequest
	}
	if idempotencyKey == "" {
		return ports.Project{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if input.Operation != "add_clip" && input.Operation != "remove_clip" {
		return ports.Project{}, domainerrors.ErrInvalidRequest
	}
	if input.Operation == "add_clip" && (input.Clip.ClipID == "" || input.Clip.DurationMS <= 0) {
		return ports.Project{}, domainerrors.ErrInvalidRequest
	}

	requestHash := hashStrings(
		input.UserID,
		input.ProjectID,
		input.Operation,
		input.Clip.ClipID,
		input.Clip.Type,
		strings.TrimSpace(input.Clip.Text),
	)
	var output ports.Project
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			project, err := s.Repo.UpdateTimeline(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(project)
		},
	)
	return output, err
}

func (s Service) RequestExport(
	ctx context.Context,
	idempotencyKey string,
	input ports.CreateExportInput,
) (ports.ExportJob, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Settings.Format = strings.TrimSpace(strings.ToLower(input.Settings.Format))
	input.Settings.Resolution = strings.TrimSpace(strings.ToLower(input.Settings.Resolution))
	input.Settings.Bitrate = strings.TrimSpace(strings.ToLower(input.Settings.Bitrate))
	input.Settings.CampaignID = strings.TrimSpace(input.Settings.CampaignID)

	if input.UserID == "" || input.ProjectID == "" {
		return ports.ExportJob{}, domainerrors.ErrInvalidRequest
	}
	if idempotencyKey == "" {
		return ports.ExportJob{}, domainerrors.ErrIdempotencyKeyRequired
	}
	if err := validateExportSettings(input.Settings); err != nil {
		return ports.ExportJob{}, err
	}
	if _, err := s.Repo.GetProject(ctx, input.UserID, input.ProjectID); err != nil {
		return ports.ExportJob{}, err
	}
	if s.MediaClient == nil {
		return ports.ExportJob{}, domainerrors.ErrDependencyUnavailable
	}

	requestHash := hashStrings(
		input.UserID,
		input.ProjectID,
		input.Settings.Format,
		input.Settings.Resolution,
		strconv.Itoa(input.Settings.FPS),
		input.Settings.Bitrate,
		input.Settings.CampaignID,
	)
	var output ports.ExportJob
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			mediaResp, err := s.MediaClient.QueueExport(ctx, input)
			if err != nil {
				return nil, domainerrors.ErrDependencyUnavailable
			}
			job, err := s.Repo.CreateExportJob(ctx, input, mediaResp, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(job)
		},
	)
	return output, err
}

func (s Service) GetExportStatus(
	ctx context.Context,
	userID string,
	projectID string,
	exportID string,
) (ports.ExportJob, error) {
	userID = strings.TrimSpace(userID)
	projectID = strings.TrimSpace(projectID)
	exportID = strings.TrimSpace(exportID)
	if userID == "" || projectID == "" || exportID == "" {
		return ports.ExportJob{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetExportJob(ctx, userID, projectID, exportID)
}

func (s Service) GetSuggestions(
	ctx context.Context,
	userID string,
	projectID string,
) ([]ports.AISuggestion, error) {
	userID = strings.TrimSpace(userID)
	projectID = strings.TrimSpace(projectID)
	if userID == "" || projectID == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	return s.Repo.ListSuggestions(ctx, userID, projectID)
}

func (s Service) SubmitToCampaign(
	ctx context.Context,
	idempotencyKey string,
	input ports.CreateSubmissionInput,
) (ports.Submission, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.ExportID = strings.TrimSpace(input.ExportID)
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	input.Platform = strings.TrimSpace(strings.ToLower(input.Platform))
	input.PostURL = strings.TrimSpace(input.PostURL)

	if input.UserID == "" || input.ProjectID == "" || input.ExportID == "" || input.CampaignID == "" {
		return ports.Submission{}, domainerrors.ErrInvalidRequest
	}
	if idempotencyKey == "" {
		return ports.Submission{}, domainerrors.ErrIdempotencyKeyRequired
	}

	requestHash := hashStrings(input.UserID, input.ProjectID, input.ExportID, input.CampaignID, input.Platform, input.PostURL)
	var output ports.Submission
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &output) },
		func() ([]byte, error) {
			item, err := s.Repo.CreateSubmission(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return output, err
}

func validateExportSettings(settings ports.ExportSettings) error {
	if settings.Format == "" {
		settings.Format = "mp4"
	}
	if settings.Format != "mp4" {
		return domainerrors.ErrInvalidRequest
	}
	switch settings.Resolution {
	case "1080x1920", "720x1280":
	default:
		return domainerrors.ErrInvalidRequest
	}
	if settings.FPS != 30 && settings.FPS != 60 {
		return domainerrors.ErrInvalidRequest
	}
	if settings.Resolution == "720x1280" && settings.FPS != 30 {
		return domainerrors.ErrInvalidRequest
	}
	return nil
}

func (s Service) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("clipping tool idempotent mutation committed",
		"event", "clipping_tool_idempotent_mutation_committed",
		"module", "campaign-editorial/clipping-tool-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
