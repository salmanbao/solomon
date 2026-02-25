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

// Module exposes M08 entrypoints needed by bootstrap (handler plus optional
// in-memory store handle for tests/dev-only wiring).
type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

// Dependencies groups infrastructure-facing ports required by the M08
// application layer. The module is storage-agnostic as long as the supplied
// adapters satisfy these contracts.
type Dependencies struct {
	Votes          ports.VoteRepository
	Idempotency    ports.IdempotencyStore
	Outbox         ports.OutboxWriter
	Clock          ports.Clock
	IDGen          ports.IDGenerator
	IdempotencyTTL time.Duration
	Logger         *slog.Logger
}

// NewModule wires the M08 application use cases and the HTTP adapter.
// Worker entrypoints are composed by callers with the same dependency set.
func NewModule(deps Dependencies) Module {
	voteUseCase := commands.VoteUseCase{
		Votes:          deps.Votes,
		Idempotency:    deps.Idempotency,
		Outbox:         deps.Outbox,
		Clock:          deps.Clock,
		IDGen:          deps.IDGen,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}
	leaderboardUseCase := queries.LeaderboardUseCase{
		Votes:  deps.Votes,
		Clock:  deps.Clock,
		Logger: deps.Logger,
	}
	return Module{
		Handler: httpadapter.Handler{
			Votes:        voteUseCase,
			Leaderboards: leaderboardUseCase,
			Logger:       deps.Logger,
		},
	}
}

// NewInMemoryModule provides a self-contained in-memory wiring used by tests
// and local bootstrap paths. A non-nil store is always returned in Module.Store
// so tests can seed state and inspect side effects deterministically.
func NewInMemoryModule(seed []entities.Vote, logger *slog.Logger) Module {
	store := memory.NewStore(seed)
	module := NewModule(Dependencies{
		Votes:          store,
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
