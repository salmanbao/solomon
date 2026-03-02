package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/clipping-tool-service/application"
	"solomon/contexts/campaign-editorial/clipping-tool-service/ports"
	httptransport "solomon/contexts/campaign-editorial/clipping-tool-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) CreateProjectHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	req httptransport.CreateProjectRequest,
) (httptransport.CreateProjectResponse, error) {
	project, err := h.Service.CreateProject(ctx, idempotencyKey, ports.CreateProjectInput{
		UserID:      strings.TrimSpace(userID),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		SourceURL:   strings.TrimSpace(req.SourceURL),
		SourceType:  strings.TrimSpace(req.SourceType),
	})
	if err != nil {
		return httptransport.CreateProjectResponse{}, err
	}

	resp := httptransport.CreateProjectResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Project = mapProject(project)
	return resp, nil
}

func (h Handler) GetProjectHandler(
	ctx context.Context,
	userID string,
	projectID string,
) (httptransport.GetProjectResponse, error) {
	project, err := h.Service.GetProject(ctx, userID, projectID)
	if err != nil {
		return httptransport.GetProjectResponse{}, err
	}
	resp := httptransport.GetProjectResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Project = mapProject(project)
	return resp, nil
}

func (h Handler) UpdateTimelineHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	projectID string,
	req httptransport.UpdateTimelineRequest,
) (httptransport.UpdateTimelineResponse, error) {
	project, err := h.Service.UpdateTimeline(ctx, idempotencyKey, ports.UpdateTimelineInput{
		UserID:    strings.TrimSpace(userID),
		ProjectID: strings.TrimSpace(projectID),
		Operation: strings.TrimSpace(req.Operation),
		Clip: ports.TimelineClip{
			ClipID:     strings.TrimSpace(req.Clip.ClipID),
			Type:       strings.TrimSpace(req.Clip.Type),
			StartMS:    req.Clip.StartMS,
			DurationMS: req.Clip.DurationMS,
			Layer:      req.Clip.Layer,
			Text:       strings.TrimSpace(req.Clip.Text),
			SourceURL:  strings.TrimSpace(req.Clip.SourceURL),
			Style:      cloneMap(req.Clip.Style),
			Effects:    cloneMap(req.Clip.Effects),
		},
	})
	if err != nil {
		return httptransport.UpdateTimelineResponse{}, err
	}
	resp := httptransport.UpdateTimelineResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Project = mapProject(project)
	return resp, nil
}

func (h Handler) RequestExportHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	projectID string,
	req httptransport.ExportRequest,
) (httptransport.ExportResponse, error) {
	job, err := h.Service.RequestExport(ctx, idempotencyKey, ports.CreateExportInput{
		UserID:    strings.TrimSpace(userID),
		ProjectID: strings.TrimSpace(projectID),
		Settings: ports.ExportSettings{
			Format:     strings.TrimSpace(req.Format),
			Resolution: strings.TrimSpace(req.Resolution),
			FPS:        req.FPS,
			Bitrate:    strings.TrimSpace(req.Bitrate),
			CampaignID: strings.TrimSpace(req.CampaignID),
		},
	})
	if err != nil {
		return httptransport.ExportResponse{}, err
	}
	resp := httptransport.ExportResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.ExportID = job.ExportID
	resp.Data.ProjectID = job.ProjectID
	resp.Data.Status = job.Status
	resp.Data.ProgressPercent = job.ProgressPercent
	resp.Data.OutputURL = job.OutputURL
	resp.Data.ProviderJobID = job.ProviderJobID
	resp.Data.CreatedAt = job.CreatedAt.UTC().Format(time.RFC3339)
	if job.CompletedAt != nil {
		resp.Data.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) GetExportStatusHandler(
	ctx context.Context,
	userID string,
	projectID string,
	exportID string,
) (httptransport.ExportStatusResponse, error) {
	job, err := h.Service.GetExportStatus(ctx, userID, projectID, exportID)
	if err != nil {
		return httptransport.ExportStatusResponse{}, err
	}
	resp := httptransport.ExportStatusResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.ExportID = job.ExportID
	resp.Data.ProjectID = job.ProjectID
	resp.Data.Status = job.Status
	resp.Data.ProgressPercent = job.ProgressPercent
	resp.Data.OutputURL = job.OutputURL
	resp.Data.ErrorMessage = job.ErrorMessage
	resp.Data.CreatedAt = job.CreatedAt.UTC().Format(time.RFC3339)
	if job.CompletedAt != nil {
		resp.Data.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) GetSuggestionsHandler(
	ctx context.Context,
	userID string,
	projectID string,
) (httptransport.SuggestionsResponse, error) {
	items, err := h.Service.GetSuggestions(ctx, userID, projectID)
	if err != nil {
		return httptransport.SuggestionsResponse{}, err
	}
	resp := httptransport.SuggestionsResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.Suggestions = make([]httptransport.SuggestionDTO, 0, len(items))
	for _, item := range items {
		resp.Data.Suggestions = append(resp.Data.Suggestions, httptransport.SuggestionDTO{
			Type:       item.Type,
			ClipID:     item.ClipID,
			StartMS:    item.StartMS,
			EndMS:      item.EndMS,
			Reason:     item.Reason,
			Confidence: item.Confidence,
		})
	}
	return resp, nil
}

func (h Handler) SubmitToCampaignHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	projectID string,
	req httptransport.SubmitRequest,
) (httptransport.SubmitResponse, error) {
	item, err := h.Service.SubmitToCampaign(ctx, idempotencyKey, ports.CreateSubmissionInput{
		UserID:     strings.TrimSpace(userID),
		ProjectID:  strings.TrimSpace(projectID),
		ExportID:   strings.TrimSpace(req.ExportID),
		CampaignID: strings.TrimSpace(req.CampaignID),
		Platform:   strings.TrimSpace(req.Platform),
		PostURL:    strings.TrimSpace(req.PostURL),
	})
	if err != nil {
		return httptransport.SubmitResponse{}, err
	}
	resp := httptransport.SubmitResponse{
		Status:    "success",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	resp.Data.SubmissionID = item.SubmissionID
	resp.Data.ProjectID = item.ProjectID
	resp.Data.ExportID = item.ExportID
	resp.Data.CampaignID = item.CampaignID
	resp.Data.Platform = item.Platform
	resp.Data.PostURL = item.PostURL
	resp.Data.Status = item.Status
	resp.Data.CreatedAt = item.CreatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func mapProject(project ports.Project) httptransport.ProjectDTO {
	output := httptransport.ProjectDTO{
		ProjectID:   project.ProjectID,
		UserID:      project.UserID,
		Title:       project.Title,
		Description: project.Description,
		State:       project.State,
		SourceURL:   project.SourceURL,
		SourceType:  project.SourceType,
		Version:     project.Version,
		CreatedAt:   project.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.UTC().Format(time.RFC3339),
		LastSavedAt: project.LastSavedAt.UTC().Format(time.RFC3339),
		Timeline: httptransport.TimelineDTO{
			DurationMS: project.Timeline.DurationMS,
			FPS:        project.Timeline.FPS,
			Width:      project.Timeline.Width,
			Height:     project.Timeline.Height,
			Clips:      make([]httptransport.TimelineClipDTO, 0, len(project.Timeline.Clips)),
		},
	}
	for _, clip := range project.Timeline.Clips {
		output.Timeline.Clips = append(output.Timeline.Clips, httptransport.TimelineClipDTO{
			ClipID:     clip.ClipID,
			Type:       clip.Type,
			StartMS:    clip.StartMS,
			DurationMS: clip.DurationMS,
			Layer:      clip.Layer,
			Text:       clip.Text,
			SourceURL:  clip.SourceURL,
			Style:      cloneMap(clip.Style),
			Effects:    cloneMap(clip.Effects),
		})
	}
	return output
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
