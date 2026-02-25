package queries

import (
	"context"
	"log/slog"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type GetClipQuery struct {
	ClipID string
}

type GetClipResult struct {
	Clip entities.Clip
}

type GetClipUseCase struct {
	Clips  ports.ClipRepository
	Logger *slog.Logger
}

func (u GetClipUseCase) Execute(ctx context.Context, query GetClipQuery) (GetClipResult, error) {
	logger := application.ResolveLogger(u.Logger)
	logger.Info("get clip started",
		"event", "get_clip_started",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"clip_id", query.ClipID,
	)

	clip, err := u.Clips.GetClip(ctx, query.ClipID)
	if err != nil {
		logger.Error("get clip failed",
			"event", "get_clip_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"clip_id", query.ClipID,
			"error", err.Error(),
		)
		return GetClipResult{}, err
	}

	logger.Info("get clip completed",
		"event", "get_clip_completed",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"clip_id", query.ClipID,
	)

	return GetClipResult{Clip: clip}, nil
}
