package teammanagementservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/internal-ops/team-management-service/adapters/http"
	"solomon/contexts/internal-ops/team-management-service/adapters/memory"
	"solomon/contexts/internal-ops/team-management-service/application"
	"solomon/contexts/internal-ops/team-management-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:           deps.Repository,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		Logger:         deps.Logger,
		IdempotencyTTL: deps.IdempotencyTTL,
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
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
