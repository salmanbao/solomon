package campaigndiscoveryservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/campaign-discovery-service/adapters/http"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/adapters/memory"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/application"
	"solomon/contexts/campaign-editorial/campaign-discovery-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository         ports.Repository
	Idempotency        ports.IdempotencyStore
	CampaignProjection ports.CampaignProjectionProvider
	ReputationProvider ports.ReputationProjectionProvider
	Clock              ports.Clock
	IdempotencyTTL     time.Duration
	Logger             *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:               deps.Repository,
		Idempotency:        deps.Idempotency,
		CampaignProjection: deps.CampaignProjection,
		ReputationProvider: deps.ReputationProvider,
		Clock:              deps.Clock,
		IdempotencyTTL:     deps.IdempotencyTTL,
		Logger:             deps.Logger,
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
		Repository:         store,
		Idempotency:        store,
		CampaignProjection: store,
		ReputationProvider: store,
		Clock:              store,
		IdempotencyTTL:     7 * 24 * time.Hour,
		Logger:             logger,
	})
	module.Store = store
	return module
}
