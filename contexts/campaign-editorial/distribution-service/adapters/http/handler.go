package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/distribution-service/application"
	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
	"solomon/contexts/campaign-editorial/distribution-service/application/queries"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/distribution-service/transport/http"
)

type Handler struct {
	Commands commands.UseCase
	Queries  queries.UseCase
	Logger   *slog.Logger
}

func (h Handler) AddOverlayHandler(
	ctx context.Context,
	itemID string,
	req httptransport.AddOverlayRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	normalizedItemID := strings.TrimSpace(itemID)
	if err := h.Commands.AddOverlay(ctx, commands.AddOverlayCommand{
		ItemID:          itemID,
		OverlayType:     req.OverlayType,
		AssetPath:       req.AssetPath,
		DurationSeconds: req.DurationSeconds,
	}); err != nil {
		logger.Warn("distribution http add overlay failed",
			"event", "distribution_http_add_overlay_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", normalizedItemID,
			"overlay_type", strings.TrimSpace(req.OverlayType),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http add overlay completed",
		"event", "distribution_http_add_overlay_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", normalizedItemID,
		"overlay_type", strings.TrimSpace(req.OverlayType),
	)
	return nil
}

func (h Handler) PreviewHandler(ctx context.Context, itemID string) (httptransport.PreviewResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	url, expiresAt, err := h.Queries.Preview(ctx, itemID)
	if err != nil {
		logger.Warn("distribution http preview failed",
			"event", "distribution_http_preview_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"error", err.Error(),
		)
		return httptransport.PreviewResponse{}, err
	}
	logger.Info("distribution http preview completed",
		"event", "distribution_http_preview_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
	)
	return httptransport.PreviewResponse{
		ID:         itemID,
		PreviewURL: url,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
	}, nil
}

func (h Handler) ScheduleHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.ScheduleRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	scheduledAt, err := parseScheduledTime(req.ScheduledFor, req.Timezone)
	if err != nil {
		logger.Warn("distribution http schedule parse failed",
			"event", "distribution_http_schedule_parse_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"timezone", strings.TrimSpace(req.Timezone),
			"error", err.Error(),
		)
		return err
	}
	if err := h.Commands.Schedule(ctx, commands.ScheduleCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platform:     req.Platform,
		ScheduledFor: scheduledAt,
		Timezone:     req.Timezone,
	}); err != nil {
		logger.Warn("distribution http schedule failed",
			"event", "distribution_http_schedule_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"platform", strings.TrimSpace(req.Platform),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http schedule completed",
		"event", "distribution_http_schedule_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"influencer_id", strings.TrimSpace(userID),
		"platform", strings.TrimSpace(req.Platform),
	)
	return nil
}

func (h Handler) RescheduleHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.ScheduleRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	scheduledAt, err := parseScheduledTime(req.ScheduledFor, req.Timezone)
	if err != nil {
		logger.Warn("distribution http reschedule parse failed",
			"event", "distribution_http_reschedule_parse_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"timezone", strings.TrimSpace(req.Timezone),
			"error", err.Error(),
		)
		return err
	}
	if err := h.Commands.Reschedule(ctx, commands.RescheduleCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platform:     req.Platform,
		ScheduledFor: scheduledAt,
		Timezone:     req.Timezone,
	}); err != nil {
		logger.Warn("distribution http reschedule failed",
			"event", "distribution_http_reschedule_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"platform", strings.TrimSpace(req.Platform),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http reschedule completed",
		"event", "distribution_http_reschedule_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"influencer_id", strings.TrimSpace(userID),
		"platform", strings.TrimSpace(req.Platform),
	)
	return nil
}

func (h Handler) DownloadHandler(
	ctx context.Context,
	itemID string,
	req httptransport.DownloadRequest,
) (httptransport.DownloadResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
	response := httptransport.DownloadResponse{
		JobID:       "download-" + strings.TrimSpace(itemID) + "-" + strings.TrimSpace(req.Platform),
		Status:      "queued",
		DownloadURL: "https://downloads.viralforge.local/distribution/" + strings.TrimSpace(itemID),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}
	logger.Info("distribution http download queued",
		"event", "distribution_http_download_queued",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"platform", strings.TrimSpace(req.Platform),
		"include_overlays", req.IncludeOverlays,
		"download_job_id", response.JobID,
	)
	return response, nil
}

func (h Handler) PublishMultiHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.PublishMultiRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.Commands.PublishMulti(ctx, commands.PublishMultiCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platforms:    append([]string(nil), req.Platforms...),
		Caption:      req.Caption,
	}); err != nil {
		logger.Warn("distribution http publish multi failed",
			"event", "distribution_http_publish_multi_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"platform_count", len(req.Platforms),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http publish multi completed",
		"event", "distribution_http_publish_multi_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"influencer_id", strings.TrimSpace(userID),
		"platform_count", len(req.Platforms),
	)
	return nil
}

func (h Handler) RetryHandler(ctx context.Context, userID string, itemID string) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.Commands.Retry(ctx, commands.RetryCommand{
		ItemID:       itemID,
		InfluencerID: userID,
	}); err != nil {
		logger.Warn("distribution http retry failed",
			"event", "distribution_http_retry_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http retry accepted",
		"event", "distribution_http_retry_accepted",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"influencer_id", strings.TrimSpace(userID),
	)
	return nil
}

func (h Handler) PublishHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.PublishRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.Commands.PublishMulti(ctx, commands.PublishMultiCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platforms:    append([]string(nil), req.Platforms...),
		Caption:      req.Caption,
	}); err != nil {
		logger.Warn("distribution http publish failed",
			"event", "distribution_http_publish_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "adapter",
			"item_id", strings.TrimSpace(itemID),
			"influencer_id", strings.TrimSpace(userID),
			"platform_count", len(req.Platforms),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("distribution http publish completed",
		"event", "distribution_http_publish_completed",
		"module", "campaign-editorial/distribution-service",
		"layer", "adapter",
		"item_id", strings.TrimSpace(itemID),
		"influencer_id", strings.TrimSpace(userID),
		"platform_count", len(req.Platforms),
	)
	return nil
}

func mapDistributionItem(item entities.DistributionItem) httptransport.DistributionItemDTO {
	dto := httptransport.DistributionItemDTO{
		ID:             item.ID,
		InfluencerID:   item.InfluencerID,
		ClipID:         item.ClipID,
		CampaignID:     item.CampaignID,
		Status:         string(item.Status),
		ClaimExpiresAt: item.ClaimExpiresAt.Format(time.RFC3339),
		Timezone:       item.Timezone,
		Platforms:      append([]string(nil), item.Platforms...),
		Caption:        item.Caption,
		RetryCount:     item.RetryCount,
		LastError:      item.LastError,
	}
	if item.ScheduledForUTC != nil {
		dto.ScheduledForUTC = item.ScheduledForUTC.Format(time.RFC3339)
	}
	return dto
}

func parseScheduledTime(value string, timezone string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}

	locationName := strings.TrimSpace(timezone)
	if locationName == "" {
		locationName = "UTC"
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return time.Time{}, domainerrors.ErrInvalidTimezone
	}
	parsed, err := time.ParseInLocation("2006-01-02T15:04:05", raw, location)
	if err != nil {
		return time.Time{}, domainerrors.ErrInvalidDistributionInput
	}
	return parsed.UTC(), nil
}
