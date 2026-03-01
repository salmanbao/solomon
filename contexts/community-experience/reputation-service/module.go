package reputationservice

import (
	"log/slog"

	httpadapter "solomon/contexts/community-experience/reputation-service/adapters/http"
	"solomon/contexts/community-experience/reputation-service/adapters/memory"
	"solomon/contexts/community-experience/reputation-service/application"
	"solomon/contexts/community-experience/reputation-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository ports.Repository
	Logger     *slog.Logger
}

func NewModule(deps Dependencies) Module {
	service := application.Service{
		Repo:   deps.Repository,
		Logger: deps.Logger,
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
		Repository: store,
		Logger:     logger,
	})
	module.Store = store
	return module
}
