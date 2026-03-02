package platformfeeengine

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/finance-core/platform-fee-engine/adapters/http"
	"solomon/contexts/finance-core/platform-fee-engine/adapters/memory"
	"solomon/contexts/finance-core/platform-fee-engine/application"
	"solomon/contexts/finance-core/platform-fee-engine/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository                        ports.Repository
	Idempotency                       ports.IdempotencyStore
	EventDedup                        ports.EventDedupStore
	Outbox                            ports.OutboxWriter
	Clock                             ports.Clock
	IDGenerator                       ports.IDGenerator
	IdempotencyTTL                    time.Duration
	EventDedupTTL                     time.Duration
	DefaultFeeRate                    float64
	DisableFeeCalculatedEventEmission bool
	Logger                            *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:                              deps.Repository,
		Idempotency:                       deps.Idempotency,
		EventDedup:                        deps.EventDedup,
		Outbox:                            deps.Outbox,
		Clock:                             deps.Clock,
		IDGen:                             deps.IDGenerator,
		IdempotencyTTL:                    deps.IdempotencyTTL,
		EventDedupTTL:                     deps.EventDedupTTL,
		DefaultFeeRate:                    deps.DefaultFeeRate,
		DisableFeeCalculatedEventEmission: deps.DisableFeeCalculatedEventEmission,
		Logger:                            deps.Logger,
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
		Outbox:         store,
		Clock:          store,
		IDGenerator:    store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		EventDedupTTL:  7 * 24 * time.Hour,
		DefaultFeeRate: 0.15,
		Logger:         logger,
	})
	module.Store = store
	return module
}
