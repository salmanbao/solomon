package clippingtoolservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/clipping-tool-service/adapters/http"
	"solomon/contexts/campaign-editorial/clipping-tool-service/adapters/memory"
	"solomon/contexts/campaign-editorial/clipping-tool-service/application"
	"solomon/contexts/campaign-editorial/clipping-tool-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	MediaClient    ports.MediaProcessingClient
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:           deps.Repository,
		Idempotency:    deps.Idempotency,
		MediaClient:    deps.MediaClient,
		Clock:          deps.Clock,
		IDGenerator:    deps.IDGenerator,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	return Module{
		Handler: httpadapter.Handler{
			Service: service,
			Logger:  deps.Logger,
		},
	}
}

func NewInMemoryModule(logger *slog.Logger) Module {
	store := memory.NewStore()
	module := NewModule(Dependencies{
		Repository:     store,
		Idempotency:    store,
		MediaClient:    store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
