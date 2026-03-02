package moderationservice

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/moderation-safety/moderation-service/adapters/http"
	"solomon/contexts/moderation-safety/moderation-service/adapters/memory"
	"solomon/contexts/moderation-safety/moderation-service/application"
	"solomon/contexts/moderation-safety/moderation-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository       ports.Repository
	Idempotency      ports.IdempotencyStore
	SubmissionClient ports.SubmissionDecisionClient
	Clock            ports.Clock
	IdempotencyTTL   time.Duration
	Logger           *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:             deps.Repository,
		Idempotency:      deps.Idempotency,
		SubmissionClient: deps.SubmissionClient,
		Clock:            deps.Clock,
		IdempotencyTTL:   deps.IdempotencyTTL,
		Logger:           deps.Logger,
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
		Repository:       store,
		Idempotency:      store,
		SubmissionClient: store,
		Clock:            store,
		IdempotencyTTL:   7 * 24 * time.Hour,
		Logger:           logger,
	})
	module.Store = store
	return module
}
