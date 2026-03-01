package superadmindashboard

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/internal-ops/super-admin-dashboard/adapters/http"
	"solomon/contexts/internal-ops/super-admin-dashboard/adapters/memory"
	"solomon/contexts/internal-ops/super-admin-dashboard/application"
	"solomon/contexts/internal-ops/super-admin-dashboard/ports"
)

// Module is the M20 composition surface exposed to Solomon runtime wiring.
type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

// Dependencies captures runtime ports/config required by NewModule.
type Dependencies struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGenerator    ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

// NewModule wires M20 use cases with explicit ports.
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

// NewInMemoryModule wires M20 against in-memory adapters for foundation/runtime bootstrap.
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
