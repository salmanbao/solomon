package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
	"solomon/contexts/campaign-editorial/distribution-service/application/queries"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
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
	return h.Commands.AddOverlay(ctx, commands.AddOverlayCommand{
		ItemID:          itemID,
		OverlayType:     req.OverlayType,
		AssetPath:       req.AssetPath,
		DurationSeconds: req.DurationSeconds,
	})
}

func (h Handler) PreviewHandler(ctx context.Context, itemID string) (httptransport.PreviewResponse, error) {
	url, expiresAt, err := h.Queries.Preview(ctx, itemID)
	if err != nil {
		return httptransport.PreviewResponse{}, err
	}
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
	scheduledAt, err := parseTime(req.ScheduledFor)
	if err != nil {
		return err
	}
	return h.Commands.Schedule(ctx, commands.ScheduleCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platform:     req.Platform,
		ScheduledFor: scheduledAt,
		Timezone:     req.Timezone,
	})
}

func (h Handler) RescheduleHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.ScheduleRequest,
) error {
	return h.ScheduleHandler(ctx, userID, itemID, req)
}

func (h Handler) DownloadHandler(
	ctx context.Context,
	itemID string,
	req httptransport.DownloadRequest,
) (httptransport.DownloadResponse, error) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	return httptransport.DownloadResponse{
		JobID:       "download-" + strings.TrimSpace(itemID) + "-" + strings.TrimSpace(req.Platform),
		Status:      "ready",
		DownloadURL: "https://downloads.viralforge.local/distribution/" + strings.TrimSpace(itemID),
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}, nil
}

func (h Handler) PublishMultiHandler(
	ctx context.Context,
	userID string,
	itemID string,
	req httptransport.PublishMultiRequest,
) error {
	return h.Commands.PublishMulti(ctx, commands.PublishMultiCommand{
		ItemID:       itemID,
		InfluencerID: userID,
		Platforms:    append([]string(nil), req.Platforms...),
		Caption:      req.Caption,
	})
}

func (h Handler) RetryHandler(ctx context.Context, userID string, itemID string) error {
	return h.Commands.Retry(ctx, commands.RetryCommand{
		ItemID:       itemID,
		InfluencerID: userID,
	})
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

func parseTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value)); err == nil {
		return parsed.UTC(), nil
	}
	return time.Parse("2006-01-02T15:04:05", strings.TrimSpace(value))
}
