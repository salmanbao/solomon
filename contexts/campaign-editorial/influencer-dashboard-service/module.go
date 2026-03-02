package influencerdashboardservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/influencer-dashboard-service/adapters/http"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/adapters/memory"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/application"
	"solomon/contexts/campaign-editorial/influencer-dashboard-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository           ports.Repository
	Idempotency          ports.IdempotencyStore
	RewardProvider       ports.RewardProvider
	GamificationProvider ports.GamificationProvider
	Clock                ports.Clock
	IdempotencyTTL       time.Duration
	Logger               *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:                 deps.Repository,
		Idempotency:          deps.Idempotency,
		RewardProvider:       deps.RewardProvider,
		GamificationProvider: deps.GamificationProvider,
		Clock:                deps.Clock,
		IdempotencyTTL:       deps.IdempotencyTTL,
		Logger:               deps.Logger,
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
		Repository:           store,
		Idempotency:          store,
		RewardProvider:       store,
		GamificationProvider: store,
		Clock:                store,
		IdempotencyTTL:       7 * 24 * time.Hour,
		Logger:               logger,
	})
	module.Store = store
	return module
}
