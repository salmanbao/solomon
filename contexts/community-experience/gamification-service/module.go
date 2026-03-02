package gamificationservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/community-experience/gamification-service/adapters/http"
	"solomon/contexts/community-experience/gamification-service/adapters/memory"
	"solomon/contexts/community-experience/gamification-service/application"
	"solomon/contexts/community-experience/gamification-service/ports"
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
		IDGen:          deps.IDGenerator,
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

func NewInMemoryModule(seed []ports.UserProjection, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
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
