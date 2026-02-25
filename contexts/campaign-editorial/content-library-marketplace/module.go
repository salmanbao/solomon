package contentlibrarymarketplace

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/content-library-marketplace/adapters/http"
	"solomon/contexts/campaign-editorial/content-library-marketplace/adapters/memory"
	"solomon/contexts/campaign-editorial/content-library-marketplace/application/commands"
	"solomon/contexts/campaign-editorial/content-library-marketplace/application/queries"
	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	"solomon/contexts/campaign-editorial/content-library-marketplace/ports"
)

// Module is the composition surface for M09 within Solomon.
// Runtime wiring should consume Handler; Store is exposed for tests/inspection.
type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Clips          ports.ClipRepository
	Claims         ports.ClaimRepository
	Downloads      ports.DownloadRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	ClaimTTL       time.Duration
	IdempotencyTTL time.Duration
	PreviewTTL     time.Duration
	DownloadTTL    time.Duration
	DownloadLimit  int
	Logger         *slog.Logger
}

// NewModule wires M09 use-cases against explicit ports.
func NewModule(deps Dependencies) Module {
	listClips := queries.ListClipsUseCase{
		Clips:  deps.Clips,
		Logger: deps.Logger,
	}
	getClip := queries.GetClipUseCase{
		Clips:  deps.Clips,
		Logger: deps.Logger,
	}
	previewClip := queries.GetClipPreviewUseCase{
		Clips:      deps.Clips,
		Clock:      deps.Clock,
		PreviewTTL: deps.PreviewTTL,
		Logger:     deps.Logger,
	}
	listClaims := queries.ListClaimsUseCase{
		Claims: deps.Claims,
		Logger: deps.Logger,
	}
	claimClip := commands.ClaimClipUseCase{
		Clips:          deps.Clips,
		Claims:         deps.Claims,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		IDGenerator:    deps.IDGenerator,
		ClaimTTL:       deps.ClaimTTL,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	downloadClip := commands.DownloadClipUseCase{
		Clips:          deps.Clips,
		Claims:         deps.Claims,
		Downloads:      deps.Downloads,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		IDGenerator:    deps.IDGenerator,
		IdempotencyTTL: deps.IdempotencyTTL,
		DownloadTTL:    deps.DownloadTTL,
		DailyLimit:     deps.DownloadLimit,
		Logger:         deps.Logger,
	}

	handler := httpadapter.Handler{
		ListClips:    listClips,
		GetClip:      getClip,
		GetPreview:   previewClip,
		ClaimClip:    claimClip,
		DownloadClip: downloadClip,
		ListClaims:   listClaims,
		Logger:       deps.Logger,
	}

	return Module{Handler: handler}
}

// NewInMemoryModule wires M09 use cases against in-memory adapters.
// This is the current developer/runtime bootstrap path until platform adapters
// (Postgres/Redis/Kafka) are fully wired into bootstrap.
func NewInMemoryModule(seedClips []entities.Clip, logger *slog.Logger) Module {
	store := memory.NewStore(seedClips, logger)
	module := NewModule(Dependencies{
		Clips:          store,
		Claims:         store,
		Downloads:      store,
		Idempotency:    store,
		Clock:          store,
		IDGenerator:    store,
		ClaimTTL:       24 * time.Hour,
		IdempotencyTTL: 7 * 24 * time.Hour,
		PreviewTTL:     15 * time.Minute,
		DownloadTTL:    24 * time.Hour,
		DownloadLimit:  5,
		Logger:         logger,
	})
	module.Store = store
	return module
}
