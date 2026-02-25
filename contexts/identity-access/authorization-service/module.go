package authorization

import (
	"log/slog"
	"time"

	httpadapter "solomon/contexts/identity-access/authorization-service/adapters/http"
	"solomon/contexts/identity-access/authorization-service/adapters/memory"
	"solomon/contexts/identity-access/authorization-service/application/commands"
	"solomon/contexts/identity-access/authorization-service/application/queries"
	"solomon/contexts/identity-access/authorization-service/ports"
)

// Module is the authorization-service composition root exposed to runtime wiring.
type Module struct {
	Handler httpadapter.Handler
	Store   *memory.Store
}

// Dependencies captures all runtime ports/config required by NewModule.
type Dependencies struct {
	Repository         ports.Repository
	Idempotency        ports.IdempotencyStore
	PermissionCache    ports.PermissionCache
	Clock              ports.Clock
	IDGenerator        ports.IDGenerator
	IdempotencyTTL     time.Duration
	PermissionCacheTTL time.Duration
	Logger             *slog.Logger
}

// NewModule wires M21 use-cases and transport handler using explicit ports.
func NewModule(deps Dependencies) Module {
	checkPermission := queries.CheckPermissionUseCase{
		Repository:         deps.Repository,
		PermissionCache:    deps.PermissionCache,
		Clock:              deps.Clock,
		PermissionCacheTTL: deps.PermissionCacheTTL,
		Logger:             deps.Logger,
	}
	checkBatch := queries.CheckPermissionsBatchUseCase{
		CheckPermission: checkPermission,
		Logger:          deps.Logger,
	}
	listRoles := queries.ListUserRolesUseCase{
		Repository: deps.Repository,
		Clock:      deps.Clock,
		Logger:     deps.Logger,
	}
	grantRole := commands.GrantRoleUseCase{
		Repository:      deps.Repository,
		Idempotency:     deps.Idempotency,
		PermissionCache: deps.PermissionCache,
		Clock:           deps.Clock,
		IDGenerator:     deps.IDGenerator,
		IdempotencyTTL:  deps.IdempotencyTTL,
		Logger:          deps.Logger,
	}
	revokeRole := commands.RevokeRoleUseCase{
		Repository:      deps.Repository,
		Idempotency:     deps.Idempotency,
		PermissionCache: deps.PermissionCache,
		Clock:           deps.Clock,
		IDGenerator:     deps.IDGenerator,
		IdempotencyTTL:  deps.IdempotencyTTL,
		Logger:          deps.Logger,
	}
	createDelegation := commands.CreateDelegationUseCase{
		Repository:     deps.Repository,
		Idempotency:    deps.Idempotency,
		IDGenerator:    deps.IDGenerator,
		Clock:          deps.Clock,
		IdempotencyTTL: deps.IdempotencyTTL,
		Logger:         deps.Logger,
	}

	handler := httpadapter.Handler{
		CheckPermission: checkPermission,
		CheckBatch:      checkBatch,
		ListRoles:       listRoles,
		GrantRole:       grantRole,
		RevokeRole:      revokeRole,
		DelegateRole:    createDelegation,
		Logger:          deps.Logger,
	}

	return Module{
		Handler: handler,
	}
}

// NewInMemoryModule builds a development/testing module with in-memory adapters.
func NewInMemoryModule(logger *slog.Logger) Module {
	store := memory.NewStore()
	module := NewModule(Dependencies{
		Repository:         store,
		Idempotency:        store,
		PermissionCache:    store,
		Clock:              store,
		IDGenerator:        store,
		IdempotencyTTL:     7 * 24 * time.Hour,
		PermissionCacheTTL: 5 * time.Minute,
		Logger:             logger,
	})
	module.Store = store
	return module
}
