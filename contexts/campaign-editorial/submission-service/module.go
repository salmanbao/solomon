package submissionservice

import (
	"log/slog"
	"time"

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
	Repository     ports.Repository
	Campaigns      ports.CampaignReadRepository
	Idempotency    ports.IdempotencyStore
	Outbox         ports.OutboxWriter
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	createSubmission := commands.CreateSubmissionUseCase{
		Repository:     deps.Repository,
		Campaigns:      deps.Campaigns,
		Idempotency:    deps.Idempotency,
		Outbox:         deps.Outbox,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	reviewSubmission := commands.ReviewSubmissionUseCase{
		Repository:     deps.Repository,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		Outbox:         deps.Outbox,
		Idempotency:    deps.Idempotency,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	reportSubmission := commands.ReportSubmissionUseCase{
		Repository:     deps.Repository,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		Outbox:         deps.Outbox,
		Idempotency:    deps.Idempotency,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	bulkOperation := commands.BulkOperationUseCase{
		Repository:     deps.Repository,
		Review:         reviewSubmission,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
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
			BulkOperation:    bulkOperation,
			Queries:          queryUseCase,
			Logger:           deps.Logger,
		},
	}
}

func NewInMemoryModule(seed []entities.Submission, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
	module := NewModule(Dependencies{
		Repository:     store,
		Campaigns:      nil,
		Idempotency:    store,
		Outbox:         store,
		Clock:          store,
		IDGen:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
