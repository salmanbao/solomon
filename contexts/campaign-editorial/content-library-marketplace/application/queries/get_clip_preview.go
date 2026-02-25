package queries

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type GetClipPreviewQuery struct {
	ClipID string
}

type GetClipPreviewResult struct {
	ClipID     string
	PreviewURL string
	ExpiresAt  time.Time
}

type GetClipPreviewUseCase struct {
	Clips      ports.ClipRepository
	Clock      ports.Clock
	PreviewTTL time.Duration
	Logger     *slog.Logger
}

func (u GetClipPreviewUseCase) Execute(ctx context.Context, query GetClipPreviewQuery) (GetClipPreviewResult, error) {
	logger := application.ResolveLogger(u.Logger)
	clip, err := u.Clips.GetClip(ctx, query.ClipID)
	if err != nil {
		return GetClipPreviewResult{}, err
	}
	if !clip.IsClaimable() {
		return GetClipPreviewResult{}, domainerrors.ErrClipUnavailable
	}

	now := time.Now().UTC()
	if u.Clock != nil {
		now = u.Clock.Now().UTC()
	}

	logger.Info("clip preview resolved",
		"event", "content_marketplace_preview_resolved",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"clip_id", clip.ClipID,
	)

	return GetClipPreviewResult{
		ClipID:     clip.ClipID,
		PreviewURL: clip.PreviewURL,
		ExpiresAt:  now.Add(u.previewTTL()),
	}, nil
}

func (u GetClipPreviewUseCase) previewTTL() time.Duration {
	if u.PreviewTTL <= 0 {
		return 15 * time.Minute
	}
	return u.PreviewTTL
}
