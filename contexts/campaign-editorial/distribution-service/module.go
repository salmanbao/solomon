package distributionservice

import (
	"log/slog"

	httpadapter "solomon/contexts/campaign-editorial/distribution-service/adapters/http"
	"solomon/contexts/campaign-editorial/distribution-service/adapters/memory"
	"solomon/contexts/campaign-editorial/distribution-service/application/commands"
	"solomon/contexts/campaign-editorial/distribution-service/application/queries"
	"solomon/contexts/campaign-editorial/distribution-service/domain/entities"
	"solomon/contexts/campaign-editorial/distribution-service/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Repository ports.Repository
	Clock      ports.Clock
	IDGen      ports.IDGenerator
	Logger     *slog.Logger
}

func NewModule(deps Dependencies) Module {
	commandUseCase := commands.UseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
		IDGen:      deps.IDGen,
		Logger:     deps.Logger,
	}
	queryUseCase := queries.UseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
	}
	return Module{
		Handler: httpadapter.Handler{
			Commands: commandUseCase,
			Queries:  queryUseCase,
			Logger:   deps.Logger,
		},
	}
}

func NewInMemoryModule(seed []entities.DistributionItem, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
	module := NewModule(Dependencies{
		Repository: store,
		Clock:      store,
		IDGen:      store,
		Logger:     logger,
	})
	module.Store = store
	return module
}
