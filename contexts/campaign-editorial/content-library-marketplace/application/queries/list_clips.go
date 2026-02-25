package queries

import (
	"context"
	"log/slog"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type ListClipsQuery struct {
	Niches         []string
	DurationBucket string
	PopularitySort string
	Status         string
	Cursor         string
	Limit          int
}

type ListClipsResult struct {
	Items      []entities.Clip
	NextCursor string
}

type ListClipsUseCase struct {
	Clips  ports.ClipRepository
	Logger *slog.Logger
}

func (u ListClipsUseCase) Execute(ctx context.Context, query ListClipsQuery) (ListClipsResult, error) {
	logger := application.ResolveLogger(u.Logger)
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	status := entities.ClipStatus(query.Status)
	if status == "" {
		status = entities.ClipStatusActive
	}
	if !isValidStatus(status) {
		return ListClipsResult{}, domainerrors.ErrInvalidListFilter
	}
	if !isValidDurationBucket(query.DurationBucket) {
		return ListClipsResult{}, domainerrors.ErrInvalidListFilter
	}
	if !isValidPopularity(query.PopularitySort) {
		return ListClipsResult{}, domainerrors.ErrInvalidListFilter
	}

	logger.Info("list clips started",
		"event", "list_clips_started",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"status", status,
		"limit", limit,
	)

	items, nextCursor, err := u.Clips.ListClips(ctx, ports.ClipListFilter{
		Niches:         query.Niches,
		DurationBucket: query.DurationBucket,
		Status:         status,
		Cursor:         query.Cursor,
		Limit:          limit,
		Popularity:     query.PopularitySort,
	})
	if err != nil {
		logger.Error("list clips failed",
			"event", "list_clips_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "application",
			"error", err.Error(),
		)
		return ListClipsResult{}, err
	}

	logger.Info("list clips completed",
		"event", "list_clips_completed",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"items_count", len(items),
		"has_next_cursor", nextCursor != "",
	)

	return ListClipsResult{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func isValidDurationBucket(value string) bool {
	switch value {
	case "", "0-15", "16-30", "31-60", "60+":
		return true
	default:
		return false
	}
}

func isValidPopularity(value string) bool {
	switch value {
	case "", "views_7d", "votes_7d", "engagement_rate":
		return true
	default:
		return false
	}
}

func isValidStatus(value entities.ClipStatus) bool {
	switch value {
	case entities.ClipStatusActive, entities.ClipStatusPaused, entities.ClipStatusArchived:
		return true
	default:
		return false
	}
}
