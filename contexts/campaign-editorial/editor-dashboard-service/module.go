package editordashboardservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/editor-dashboard-service/adapters/http"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/adapters/memory"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/application"
	"solomon/contexts/campaign-editorial/editor-dashboard-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	EventDedup     ports.EventDedupStore
	Clock          ports.Clock
	IdempotencyTTL time.Duration
	EventDedupTTL  time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:           deps.Repository,
		Idempotency:    deps.Idempotency,
		EventDedup:     deps.EventDedup,
		Clock:          deps.Clock,
		IdempotencyTTL: deps.IdempotencyTTL,
		EventDedupTTL:  deps.EventDedupTTL,
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
		EventDedup:     store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		EventDedupTTL:  7 * 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
