package httpadapter

import (
	"context"
	"log/slog"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	"solomon/contexts/campaign-editorial/content-library-marketplace/application/commands"
	"solomon/contexts/campaign-editorial/content-library-marketplace/application/queries"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/content-library-marketplace/transport/http"
)

type Handler struct {
	ListClips    queries.ListClipsUseCase
	GetClip      queries.GetClipUseCase
	GetPreview   queries.GetClipPreviewUseCase
	ClaimClip    commands.ClaimClipUseCase
	DownloadClip commands.DownloadClipUseCase
	ListClaims   queries.ListClaimsUseCase
	Logger       *slog.Logger
}

// ListClipsHandler godoc
// @Summary List marketplace clips
// @Description Returns clip catalog with filters and cursor pagination.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Param niche query []string false "Niche filter"
// @Param duration_bucket query string false "Duration bucket: 0-15,16-30,31-60,60+"
// @Param popularity_sort query string false "Sort: views_7d,votes_7d,engagement_rate"
// @Param status query string false "Clip status"
// @Param cursor query string false "Cursor token"
// @Param limit query int false "Page size (max 50)"
// @Success 200 {object} httptransport.ListClipsResponse
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/clips [get]
func (h Handler) ListClipsHandler(ctx context.Context, req httptransport.ListClipsRequest) (httptransport.ListClipsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("list clips request received",
		"event", "http_list_clips_received",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "transport",
	)

	result, err := h.ListClips.Execute(ctx, queries.ListClipsQuery{
		Niches:         req.Niche,
		DurationBucket: req.DurationBucket,
		PopularitySort: req.PopularitySort,
		Status:         req.Status,
		Cursor:         req.Cursor,
		Limit:          req.Limit,
	})
	if err != nil {
		logger.Error("list clips request failed",
			"event", "http_list_clips_failed",
			"module", "campaign-editorial/content-library-marketplace",
			"layer", "transport",
			"error", err.Error(),
		)
		return httptransport.ListClipsResponse{}, err
	}

	return httptransport.ListClipsResponse{
		Items:      mapClips(result.Items),
		NextCursor: result.NextCursor,
	}, nil
}

// GetClipHandler godoc
// @Summary Get clip details
// @Description Returns one clip by id.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Param clip_id path string true "Clip id"
// @Success 200 {object} httptransport.GetClipResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/clips/{clip_id} [get]
func (h Handler) GetClipHandler(ctx context.Context, clipID string) (httptransport.GetClipResponse, error) {
	result, err := h.GetClip.Execute(ctx, queries.GetClipQuery{ClipID: clipID})
	if err != nil {
		return httptransport.GetClipResponse{}, err
	}
	return httptransport.GetClipResponse{
		Item: mapClip(result.Clip),
	}, nil
}

// GetClipPreviewHandler godoc
// @Summary Get clip preview URL
// @Description Returns a preview URL with expiry metadata.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Param clip_id path string true "Clip id"
// @Success 200 {object} httptransport.GetClipPreviewResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 410 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/clips/{clip_id}/preview [get]
func (h Handler) GetClipPreviewHandler(ctx context.Context, clipID string) (httptransport.GetClipPreviewResponse, error) {
	result, err := h.GetPreview.Execute(ctx, queries.GetClipPreviewQuery{ClipID: clipID})
	if err != nil {
		return httptransport.GetClipPreviewResponse{}, err
	}
	return httptransport.GetClipPreviewResponse{
		ClipID:     result.ClipID,
		PreviewURL: result.PreviewURL,
		ExpiresAt:  result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ClaimClipHandler godoc
// @Summary Claim a clip
// @Description Creates an active claim with idempotency support.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Param Idempotency-Key header string true "Idempotency key"
// @Param clip_id path string true "Clip id"
// @Param request body httptransport.ClaimClipRequest true "Claim payload"
// @Success 200 {object} httptransport.ClaimClipResponse
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 409 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/clips/{clip_id}/claim [post]
func (h Handler) ClaimClipHandler(
	ctx context.Context,
	userID string,
	clipID string,
	req httptransport.ClaimClipRequest,
	idempotencyKey string,
) (httptransport.ClaimClipResponse, error) {
	result, err := h.ClaimClip.Execute(ctx, commands.ClaimClipCommand{
		ClipID:         clipID,
		UserID:         userID,
		RequestID:      req.RequestID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return httptransport.ClaimClipResponse{}, err
	}
	return httptransport.ClaimClipResponse{
		ClaimID:   result.Claim.ClaimID,
		ClipID:    result.Claim.ClipID,
		Status:    string(result.Claim.Status),
		ExpiresAt: result.Claim.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		Replayed:  result.Replayed,
	}, nil
}

// DownloadClipHandler godoc
// @Summary Get signed clip download URL
// @Description Returns a signed download URL for users with an active claim.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Param Idempotency-Key header string false "Idempotency key"
// @Param clip_id path string true "Clip id"
// @Param X-Forwarded-For header string false "Client IP"
// @Success 200 {object} httptransport.DownloadClipResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 403 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 410 {object} httptransport.ErrorResponse
// @Failure 429 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/clips/{clip_id}/download [post]
func (h Handler) DownloadClipHandler(
	ctx context.Context,
	userID string,
	clipID string,
	idempotencyKey string,
	ipAddress string,
	userAgent string,
) (httptransport.DownloadClipResponse, error) {
	result, err := h.DownloadClip.Execute(ctx, commands.DownloadClipCommand{
		ClipID:         clipID,
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	})
	if err != nil {
		return httptransport.DownloadClipResponse{}, err
	}
	return httptransport.DownloadClipResponse{
		DownloadURL:        result.DownloadURL,
		ExpiresAt:          result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		RemainingDownloads: result.RemainingDownloads,
		Replayed:           result.Replayed,
	}, nil
}

// ListClaimsHandler godoc
// @Summary List influencer claims
// @Description Returns claims for the authenticated user.
// @Tags content-library-marketplace
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-Request-Id header string true "Request correlation id"
// @Success 200 {object} httptransport.ListClaimsResponse
// @Failure 401 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /library/claims [get]
func (h Handler) ListClaimsHandler(ctx context.Context, userID string) (httptransport.ListClaimsResponse, error) {
	result, err := h.ListClaims.Execute(ctx, queries.ListClaimsQuery{UserID: userID})
	if err != nil {
		return httptransport.ListClaimsResponse{}, err
	}

	items := make([]httptransport.ClaimDTO, 0, len(result.Items))
	for _, claim := range result.Items {
		items = append(items, httptransport.ClaimDTO{
			ClaimID:   claim.ClaimID,
			ClipID:    claim.ClipID,
			Status:    string(claim.Status),
			ExpiresAt: claim.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return httptransport.ListClaimsResponse{Items: items}, nil
}

func mapClips(clips []entities.Clip) []httptransport.ClipDTO {
	items := make([]httptransport.ClipDTO, 0, len(clips))
	for _, clip := range clips {
		items = append(items, mapClip(clip))
	}
	return items
}

func mapClip(clip entities.Clip) httptransport.ClipDTO {
	return httptransport.ClipDTO{
		ClipID:          clip.ClipID,
		Title:           clip.Title,
		Niche:           clip.Niche,
		DurationSeconds: clip.DurationSeconds,
		PreviewURL:      clip.PreviewURL,
		Exclusivity:     string(clip.Exclusivity),
		ClaimLimit:      clip.EffectiveClaimLimit(),
		Stats: httptransport.ClipStatsDTO{
			Views7d:        clip.Views7d,
			Votes7d:        clip.Votes7d,
			EngagementRate: clip.EngagementRate,
		},
	}
}
