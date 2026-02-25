package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/content-library-marketplace/application"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

type DownloadClipCommand struct {
	ClipID         string
	UserID         string
	IdempotencyKey string
	IPAddress      string
	UserAgent      string
}

type DownloadClipResult struct {
	DownloadURL        string
	ExpiresAt          time.Time
	RemainingDownloads int
	Replayed           bool
}

type DownloadClipUseCase struct {
	Clips          ports.ClipRepository
	Claims         ports.ClaimRepository
	Downloads      ports.DownloadRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	DownloadTTL    time.Duration
	DailyLimit     int
	Logger         *slog.Logger
}

func (u DownloadClipUseCase) Execute(ctx context.Context, cmd DownloadClipCommand) (DownloadClipResult, error) {
	if strings.TrimSpace(cmd.ClipID) == "" || strings.TrimSpace(cmd.UserID) == "" {
		return DownloadClipResult{}, domainerrors.ErrInvalidClaimRequest
	}

	logger := application.ResolveLogger(u.Logger)
	now := time.Now().UTC()
	if u.Clock != nil {
		now = u.Clock.Now().UTC()
	}

	clip, err := u.Clips.GetClip(ctx, cmd.ClipID)
	if err != nil {
		return DownloadClipResult{}, err
	}
	if !clip.IsClaimable() {
		return DownloadClipResult{}, domainerrors.ErrClipUnavailable
	}

	if err := u.ensureActiveClaim(ctx, cmd.UserID, cmd.ClipID, now); err != nil {
		return DownloadClipResult{}, err
	}

	key := resolveDownloadIdempotencyKey(cmd, now)
	requestHash := hashDownloadRequest(cmd.UserID, cmd.ClipID)

	record, found, err := u.Idempotency.Get(ctx, key, now)
	if err != nil {
		return DownloadClipResult{}, err
	}
	if found && record.RequestHash != requestHash {
		return DownloadClipResult{}, domainerrors.ErrIdempotencyKeyConflict
	}

	limit := u.dailyLimit()
	count, err := u.Downloads.CountUserClipDownloadsSince(ctx, cmd.UserID, cmd.ClipID, now.Add(-24*time.Hour))
	if err != nil {
		return DownloadClipResult{}, err
	}

	if found {
		return DownloadClipResult{
			DownloadURL:        buildSignedDownloadURL(clip.DownloadAssetID, cmd.UserID, cmd.ClipID, now.Add(u.downloadTTL())),
			ExpiresAt:          now.Add(u.downloadTTL()),
			RemainingDownloads: max(limit-count, 0),
			Replayed:           true,
		}, nil
	}

	if count >= limit {
		return DownloadClipResult{}, domainerrors.ErrDownloadLimitReached
	}

	downloadID, err := u.IDGenerator.NewID(ctx)
	if err != nil {
		return DownloadClipResult{}, err
	}
	if err := u.Downloads.CreateDownload(ctx, ports.ClipDownload{
		DownloadID:   downloadID,
		ClipID:       cmd.ClipID,
		UserID:       cmd.UserID,
		IPAddress:    strings.TrimSpace(cmd.IPAddress),
		UserAgent:    strings.TrimSpace(cmd.UserAgent),
		DownloadedAt: now,
	}); err != nil {
		return DownloadClipResult{}, err
	}

	if err := u.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		ClaimID:     downloadID,
		ExpiresAt:   now.Add(u.idempotencyTTL()),
	}); err != nil {
		return DownloadClipResult{}, err
	}

	logger.Info("clip download issued",
		"event", "content_marketplace_download_issued",
		"module", "campaign-editorial/content-library-marketplace",
		"layer", "application",
		"clip_id", cmd.ClipID,
		"user_id", cmd.UserID,
		"download_id", downloadID,
	)

	expiresAt := now.Add(u.downloadTTL())
	return DownloadClipResult{
		DownloadURL:        buildSignedDownloadURL(clip.DownloadAssetID, cmd.UserID, cmd.ClipID, expiresAt),
		ExpiresAt:          expiresAt,
		RemainingDownloads: max(limit-(count+1), 0),
	}, nil
}

func (u DownloadClipUseCase) ensureActiveClaim(ctx context.Context, userID string, clipID string, now time.Time) error {
	claims, err := u.Claims.ListClaimsByUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if claim.ClipID == clipID && claim.OccupiesSlot(now) {
			return nil
		}
	}
	return domainerrors.ErrClaimRequired
}

func (u DownloadClipUseCase) idempotencyTTL() time.Duration {
	if u.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return u.IdempotencyTTL
}

func (u DownloadClipUseCase) downloadTTL() time.Duration {
	if u.DownloadTTL <= 0 {
		return 24 * time.Hour
	}
	return u.DownloadTTL
}

func (u DownloadClipUseCase) dailyLimit() int {
	if u.DailyLimit <= 0 {
		return 5
	}
	return u.DailyLimit
}

func resolveDownloadIdempotencyKey(cmd DownloadClipCommand, now time.Time) string {
	if strings.TrimSpace(cmd.IdempotencyKey) != "" {
		return cmd.IdempotencyKey
	}
	return fmt.Sprintf("cms:%s:%s:download:%s", cmd.UserID, cmd.ClipID, now.Format("2006-01-02"))
}

func hashDownloadRequest(userID string, clipID string) string {
	sum := sha256.Sum256([]byte(userID + "|" + clipID))
	return hex.EncodeToString(sum[:])
}

func buildSignedDownloadURL(assetID string, userID string, clipID string, expiresAt time.Time) string {
	signature := sha256.Sum256([]byte(assetID + "|" + userID + "|" + clipID + "|" + expiresAt.UTC().Format(time.RFC3339)))
	return fmt.Sprintf(
		"https://cdn.local/downloads/%s?expires_at=%s&sig=%s",
		assetID,
		expiresAt.UTC().Format(time.RFC3339),
		hex.EncodeToString(signature[:]),
	)
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
