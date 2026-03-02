package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"solomon/contexts/campaign-editorial/clipping-tool-service/adapters/memory"
	domainerrors "solomon/contexts/campaign-editorial/clipping-tool-service/domain/errors"
	"solomon/contexts/campaign-editorial/clipping-tool-service/ports"
)

func TestCreateProjectRequiresIdempotencyKey(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		MediaClient:    store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 24 * time.Hour,
	}

	_, err := service.CreateProject(context.Background(), "", ports.CreateProjectInput{
		UserID: "user-1",
		Title:  "project",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyKeyRequired) {
		t.Fatalf("expected ErrIdempotencyKeyRequired, got %v", err)
	}
}

func TestCreateProjectIsIdempotent(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		MediaClient:    store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 24 * time.Hour,
	}

	key := "clip-create-1"
	input := ports.CreateProjectInput{
		UserID:      "user-1",
		Title:       "Summer Fitness Challenge",
		Description: "draft",
		SourceURL:   "https://cdn.whop.dev/source.mp4",
		SourceType:  "url",
	}
	first, err := service.CreateProject(context.Background(), key, input)
	if err != nil {
		t.Fatalf("first call returned error: %v", err)
	}
	second, err := service.CreateProject(context.Background(), key, input)
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if first.ProjectID != second.ProjectID {
		t.Fatalf("expected replay to return same project id, got %s and %s", first.ProjectID, second.ProjectID)
	}
}

func TestRequestExportRejectsInvalidPreset(t *testing.T) {
	store := memory.NewStore()
	service := Service{
		Repo:           store,
		Idempotency:    store,
		MediaClient:    store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 24 * time.Hour,
	}

	_, err := service.RequestExport(context.Background(), "exp-1", ports.CreateExportInput{
		UserID:    "creator-seed",
		ProjectID: "proj-seed-1",
		Settings: ports.ExportSettings{
			Format:     "mp4",
			Resolution: "720x1280",
			FPS:        60,
			Bitrate:    "5m",
		},
	})
	if !errors.Is(err, domainerrors.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for invalid preset, got %v", err)
	}
}
