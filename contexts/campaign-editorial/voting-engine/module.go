package votingengine

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/campaign-editorial/voting-engine/adapters/http"
	"solomon/contexts/campaign-editorial/voting-engine/adapters/memory"
	"solomon/contexts/campaign-editorial/voting-engine/application/commands"
	"solomon/contexts/campaign-editorial/voting-engine/application/queries"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

type Dependencies struct {
	Votes          ports.VoteRepository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

func NewModule(deps Dependencies) Module {
	voteUseCase := commands.VoteUseCase{
		Votes:          deps.Votes,
		Idempotency:    deps.Idempotency,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	leaderboardUseCase := queries.LeaderboardUseCase{
		Votes: deps.Votes,
	}
	return Module{
		Handler: httpadapter.Handler{
			Votes:        voteUseCase,
			Leaderboards: leaderboardUseCase,
			Logger:       deps.Logger,
		},
	}
}

func NewInMemoryModule(seed []entities.Vote, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
	module := NewModule(Dependencies{
		Votes:          store,
		Idempotency:    store,
		Clock:          store,
		IDGen:          store,
		IdempotencyTTL: 24 * time.Hour,
		Logger:         logger,
	})
	module.Store = store
	return module
}
