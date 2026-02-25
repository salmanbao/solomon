package submissionservice

import (
	"log/slog"

	httpadapter "solomon/contexts/campaign-editorial/submission-service/adapters/http"
	"solomon/contexts/campaign-editorial/submission-service/adapters/memory"
	"solomon/contexts/campaign-editorial/submission-service/application/commands"
	"solomon/contexts/campaign-editorial/submission-service/application/queries"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	"solomon/contexts/campaign-editorial/submission-service/ports"
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
	createSubmission := commands.CreateSubmissionUseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
		IDGen:      deps.IDGen,
		Logger:     deps.Logger,
	}
	reviewSubmission := commands.ReviewSubmissionUseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
		Logger:     deps.Logger,
	}
	reportSubmission := commands.ReportSubmissionUseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
		IDGen:      deps.IDGen,
		Logger:     deps.Logger,
	}
	queryUseCase := queries.QueryUseCase{
		Repository: deps.Repository,
		Logger:     deps.Logger,
	}

	return Module{
		Handler: httpadapter.Handler{
			CreateSubmission: createSubmission,
			ReviewSubmission: reviewSubmission,
			ReportSubmission: reportSubmission,
			Queries:          queryUseCase,
			Logger:           deps.Logger,
		},
	}
}

func NewInMemoryModule(seed []entities.Submission, logger *slog.Logger) Module {
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
