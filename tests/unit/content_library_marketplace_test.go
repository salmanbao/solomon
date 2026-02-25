package unit

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/content-library-marketplace/transport/http"
)

func TestClaimClipExclusiveConflict(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-exclusive",
			Title:           "Exclusive Clip",
			Niche:           "fitness",
			DurationSeconds: 24,
			PreviewURL:      "https://cdn.example/clip-exclusive-preview",
			Exclusivity:     entities.ClipExclusivityExclusive,
			ClaimLimit:      1,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	_, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-a",
		"clip-exclusive",
		httptransport.ClaimClipRequest{RequestID: "req-a"},
		"idem-a",
	)
	if err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}

	_, err = module.Handler.ClaimClipHandler(
		context.Background(),
		"user-b",
		"clip-exclusive",
		httptransport.ClaimClipRequest{RequestID: "req-b"},
		"idem-b",
	)
	if !errors.Is(err, domainerrors.ErrExclusiveClaimConflict) {
		t.Fatalf("expected exclusive conflict, got %v", err)
	}
}

func TestClaimClipIdempotencyReplay(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-non-exclusive",
			Title:           "Shared Clip",
			Niche:           "education",
			DurationSeconds: 31,
			PreviewURL:      "https://cdn.example/clip-shared-preview",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			ClaimLimit:      50,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	first, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-a",
		"clip-non-exclusive",
		httptransport.ClaimClipRequest{RequestID: "req-1"},
		"idem-key",
	)
	if err != nil {
		t.Fatalf("first claim should succeed: %v", err)
	}

	second, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-a",
		"clip-non-exclusive",
		httptransport.ClaimClipRequest{RequestID: "req-1"},
		"idem-key",
	)
	if err != nil {
		t.Fatalf("second claim should replay: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed response")
	}
	if second.ClaimID != first.ClaimID {
		t.Fatalf("expected same claim id, got %s and %s", first.ClaimID, second.ClaimID)
	}
}

func TestListClipsPagination(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-1",
			Title:           "A",
			Niche:           "fitness",
			DurationSeconds: 12,
			PreviewURL:      "https://cdn.example/1",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			ClaimLimit:      50,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-3 * time.Hour),
		},
		{
			ClipID:          "clip-2",
			Title:           "B",
			Niche:           "fitness",
			DurationSeconds: 22,
			PreviewURL:      "https://cdn.example/2",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			ClaimLimit:      50,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-2 * time.Hour),
		},
		{
			ClipID:          "clip-3",
			Title:           "C",
			Niche:           "gaming",
			DurationSeconds: 42,
			PreviewURL:      "https://cdn.example/3",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			ClaimLimit:      50,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	firstPage, err := module.Handler.ListClipsHandler(context.Background(), httptransport.ListClipsRequest{
		Niche: []string{"fitness"},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("list clips first page failed: %v", err)
	}
	if len(firstPage.Items) != 1 {
		t.Fatalf("expected first page size 1, got %d", len(firstPage.Items))
	}
	if firstPage.NextCursor == "" {
		t.Fatalf("expected next cursor on first page")
	}

	secondPage, err := module.Handler.ListClipsHandler(context.Background(), httptransport.ListClipsRequest{
		Niche:  []string{"fitness"},
		Limit:  1,
		Cursor: firstPage.NextCursor,
	})
	if err != nil {
		t.Fatalf("list clips second page failed: %v", err)
	}
	if len(secondPage.Items) != 1 {
		t.Fatalf("expected second page size 1, got %d", len(secondPage.Items))
	}
	if secondPage.Items[0].ClipID == firstPage.Items[0].ClipID {
		t.Fatalf("expected different clip on second page")
	}
}

func TestListClipsDurationBucket(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-short",
			Title:           "Short",
			Niche:           "fitness",
			DurationSeconds: 12,
			PreviewURL:      "https://cdn.example/short",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
		{
			ClipID:          "clip-medium",
			Title:           "Medium",
			Niche:           "fitness",
			DurationSeconds: 24,
			PreviewURL:      "https://cdn.example/medium",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-2 * time.Hour),
		},
	}, nil)

	resp, err := module.Handler.ListClipsHandler(context.Background(), httptransport.ListClipsRequest{
		DurationBucket: "16-30",
	})
	if err != nil {
		t.Fatalf("list clips by duration bucket failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one clip in duration bucket, got %d", len(resp.Items))
	}
	if resp.Items[0].ClipID != "clip-medium" {
		t.Fatalf("expected clip-medium, got %s", resp.Items[0].ClipID)
	}
}

func TestListClipsInvalidDurationBucket(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-short",
			Title:           "Short",
			Niche:           "fitness",
			DurationSeconds: 12,
			PreviewURL:      "https://cdn.example/short",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	_, err := module.Handler.ListClipsHandler(context.Background(), httptransport.ListClipsRequest{
		DurationBucket: "invalid",
	})
	if !errors.Is(err, domainerrors.ErrInvalidListFilter) {
		t.Fatalf("expected invalid filter error, got %v", err)
	}
}

func TestDownloadClipRequiresActiveClaim(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-download",
			Title:           "Downloadable",
			Niche:           "fitness",
			DurationSeconds: 20,
			PreviewURL:      "https://cdn.example/preview",
			DownloadAssetID: "asset-download",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	_, err := module.Handler.DownloadClipHandler(
		context.Background(),
		"user-a",
		"clip-download",
		"idem-download-1",
		"127.0.0.1",
		"unit-test",
	)
	if !errors.Is(err, domainerrors.ErrClaimRequired) {
		t.Fatalf("expected claim required error, got %v", err)
	}
}

func TestDownloadClipDailyLimit(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-download-limit",
			Title:           "Downloadable",
			Niche:           "fitness",
			DurationSeconds: 20,
			PreviewURL:      "https://cdn.example/preview",
			DownloadAssetID: "asset-download-limit",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	_, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-a",
		"clip-download-limit",
		httptransport.ClaimClipRequest{RequestID: "req-download-limit"},
		"idem-claim-download-limit",
	)
	if err != nil {
		t.Fatalf("claim before download should succeed: %v", err)
	}

	for i := 0; i < 5; i++ {
		_, err := module.Handler.DownloadClipHandler(
			context.Background(),
			"user-a",
			"clip-download-limit",
			"idem-download-"+strconv.Itoa(i),
			"127.0.0.1",
			"unit-test",
		)
		if err != nil {
			t.Fatalf("download %d should succeed: %v", i+1, err)
		}
	}

	_, err = module.Handler.DownloadClipHandler(
		context.Background(),
		"user-a",
		"clip-download-limit",
		"idem-download-over-limit",
		"127.0.0.1",
		"unit-test",
	)
	if !errors.Is(err, domainerrors.ErrDownloadLimitReached) {
		t.Fatalf("expected download limit error, got %v", err)
	}
}

func TestExpireActiveClaimsSweep(t *testing.T) {
	module := contentlibrarymarketplace.NewInMemoryModule([]entities.Clip{
		{
			ClipID:          "clip-expiry",
			Title:           "Expiry",
			Niche:           "fitness",
			DurationSeconds: 20,
			PreviewURL:      "https://cdn.example/preview",
			Exclusivity:     entities.ClipExclusivityNonExclusive,
			Status:          entities.ClipStatusActive,
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}, nil)

	claimed, err := module.Handler.ClaimClipHandler(
		context.Background(),
		"user-expiry",
		"clip-expiry",
		httptransport.ClaimClipRequest{RequestID: "req-expiry"},
		"idem-expiry",
	)
	if err != nil {
		t.Fatalf("claim should succeed: %v", err)
	}

	expiredCount, err := module.Store.ExpireActiveClaims(context.Background(), time.Now().Add(25*time.Hour))
	if err != nil {
		t.Fatalf("expire sweep failed: %v", err)
	}
	if expiredCount != 1 {
		t.Fatalf("expected one expired claim, got %d", expiredCount)
	}

	claims, err := module.Handler.ListClaimsHandler(context.Background(), "user-expiry")
	if err != nil {
		t.Fatalf("list claims failed: %v", err)
	}
	if len(claims.Items) != 1 {
		t.Fatalf("expected one claim, got %d", len(claims.Items))
	}
	if claims.Items[0].ClaimID != claimed.ClaimID {
		t.Fatalf("unexpected claim id after sweep: %s", claims.Items[0].ClaimID)
	}
	if claims.Items[0].Status != string(entities.ClaimStatusExpired) {
		t.Fatalf("expected claim status expired, got %s", claims.Items[0].Status)
	}
}
