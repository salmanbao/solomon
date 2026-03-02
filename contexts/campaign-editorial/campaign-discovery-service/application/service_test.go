package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"solomon/contexts/campaign-editorial/campaign-discovery-service/adapters/memory"
	domainerrors "solomon/contexts/campaign-editorial/campaign-discovery-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/ports"
)

func TestBrowseCampaignsRequiresUserID(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:               store,
		Idempotency:        store,
		CampaignProjection: store,
		ReputationProvider: store,
		Clock:              store,
		IdempotencyTTL:     24 * time.Hour,
	}

	_, err := service.BrowseCampaigns(context.Background(), ports.BrowseQuery{})
	if !errors.Is(err, domainerrors.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestSaveBookmarkIsIdempotent(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:               store,
		Idempotency:        store,
		CampaignProjection: store,
		ReputationProvider: store,
		Clock:              store,
		IdempotencyTTL:     24 * time.Hour,
	}
	key := "discover-bookmark-1"
	command := ports.BookmarkCommand{
		UserID:     "user-1",
		CampaignID: "c-12345678-9abc-def0-1234-56789abcdef0",
		Tag:        "fitness",
		Note:       "watch later",
	}

	first, err := service.SaveBookmark(context.Background(), key, command)
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	second, err := service.SaveBookmark(context.Background(), key, command)
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if first.BookmarkID != second.BookmarkID {
		t.Fatalf("expected same bookmark id for idempotent replay, got %s and %s", first.BookmarkID, second.BookmarkID)
	}
}

func TestSaveBookmarkDetectsIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:               store,
		Idempotency:        store,
		CampaignProjection: store,
		ReputationProvider: store,
		Clock:              store,
		IdempotencyTTL:     24 * time.Hour,
	}
	key := "discover-bookmark-2"
	_, err := service.SaveBookmark(context.Background(), key, ports.BookmarkCommand{
		UserID:     "user-1",
		CampaignID: "c-12345678-9abc-def0-1234-56789abcdef0",
		Tag:        "fitness",
	})
	if err != nil {
		t.Fatalf("seed call returned error: %v", err)
	}

	_, err = service.SaveBookmark(context.Background(), key, ports.BookmarkCommand{
		UserID:     "user-1",
		CampaignID: "c-11111111-2222-3333-4444-555555555555",
		Tag:        "tech",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}
