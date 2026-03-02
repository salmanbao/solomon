package admindashboardservice

import (
	"time"

	httpadapter "solomon/contexts/internal-ops/admin-dashboard-service/adapters/http"
	"solomon/contexts/internal-ops/admin-dashboard-service/adapters/memory"
	"solomon/contexts/internal-ops/admin-dashboard-service/application"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository     ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IdempotencyTTL time.Duration
}

func NewModule(deps Dependencies) Module {
	return Module{
		Handler: httpadapter.Handler{
			Service: application.Service{
				Repo:           deps.Repository,
				Idempotency:    deps.Idempotency,
				Clock:          deps.Clock,
				IdempotencyTTL: deps.IdempotencyTTL,
			},
		},
	}
}

func NewInMemoryModule() Module {
	store := memory.NewStore()
	module := NewModule(Dependencies{
		Repository:     store,
		Idempotency:    store,
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	})
	module.Store = store
	return module
}
