# Monolith Boundaries Audit

Date: 2026-02-23  
Scope: `solomon/` modular monolith structure and enforcement

## Summary Decisions

1. Platform strategy: **Option A** selected. Top-level `platform/` removed. `internal/platform/` is canonical.
2. Contracts strategy: root contracts split into separate module at `solomon/contracts/go.mod`.
3. Module-local contracts: renamed from `contracts/` semantics to `transport/` semantics.
4. Runtime packaging: `apps/` renamed to `deploy/` and kept non-Go.

## Concern 1: Dual Platform Layers

### Findings

- Duplicate platform surfaces existed:
  - `platform/README.md` (+ empty capability dirs)
  - `internal/platform/config/config.go`
  - `internal/platform/db/postgres.go`
  - `internal/platform/httpserver/server.go`
  - `internal/platform/messaging/kafka.go`
  - `internal/platform/observability/tracing.go`
- No runtime code imported top-level `platform/*`, but its presence created architectural ambiguity.

### Risk

- High drift risk: engineers can place code in two locations with conflicting semantics.

### Fix Applied

- Removed top-level `platform/`.
- Updated docs and lint policy to treat `internal/platform/*` as the only valid concrete platform layer.

## Concern 2: Contract Semantics Inconsistency

### Findings

- Root `contracts/` was described as stable/public, while module-local DTOs were in module `contracts/`.
- Example module-local DTO before fix:
  - `contexts/identity-access/authorization-service/contracts/http_dto.go`
- `mesh/` scan found no direct imports of Solomon DTO paths (`rg -n "solomon" ../mesh -g "*.go"` returned no matches).

### Risk

- Medium-high risk of future runtime coupling and accidental contract misuse.

### Fix Applied

- Module-local DTO location changed to `transport/`.
- Concrete move:
  - from `contexts/identity-access/authorization-service/contracts/http_dto.go`
  - to `contexts/identity-access/authorization-service/transport/http/http_dto.go`
- Updated module scaffolds/docs to describe `transport/` as module-private.

## Concern 3: Contracts Go Module Boundary

### Findings

- Only one Go module existed (`go.mod` at repository root, `module solomon`).
- Root `contracts/` had no separate module boundary.

### Risk

- High risk that other runtimes would depend on monolith runtime module graph to consume contracts.

### Fix Applied

- Added separate contracts module:
  - `contracts/go.mod` (`module solomon/contracts`)
  - `contracts/gen/events/v1/envelope.go` (generated-style contract type)
  - versioned schema examples under `contracts/events/v1`, `contracts/api/v1`, `contracts/schemas/v1`
- Added `go.work` to compose runtime + contracts modules locally.

## Concern 4: `apps/` vs `cmd/` Ambiguity

### Findings

- `apps/` contained only placeholders (`apps/README.md`, empty subdirs), no Go code.
- `cmd/api/main.go` and `cmd/worker/main.go` already represented runtime entrypoints.

### Risk

- Medium ambiguity on whether `apps/` was runtime code or deployment assets.

### Fix Applied

- Renamed `apps/` to `deploy/`.
- Updated docs to state `deploy/` must remain non-Go.

## Concern 5: `integrations/microservices` Dumping Ground Risk

### Findings

- `integrations/` had policy text suggesting centralized microservice clients.
- No active client implementation existed there in current code.

### Risk

- Medium-high future coupling risk via shared outbound client dumping ground.

### Fix Applied

- Removed `integrations/microservices` directory scaffolding.
- Rewrote `integrations/README.md` to enforce:
  - no shared central client layer
  - module-owned outbound adapters under `contexts/<context>/<service>/adapters`
  - shared low-level helpers only in `internal/shared`

## Concern 6: Empty Scaffolding + `.gitkeep` Spam

### Findings

- 150 `.gitkeep` placeholders existed across module scaffold directories.
- 29 of 30 service modules had no code beyond README + placeholders.

### Risk

- Medium maintenance noise; weak signal-to-noise in code review; false sense of implementation progress.

### Fix Applied

- Removed `.gitkeep` placeholders.
- Removed empty scaffold directories from repository state.
- Added module generator:
  - `scripts/newmodule.ps1`
  - `scripts/newmodule.sh`

## Enforcement Added

- `golangci-lint` config with `depguard`: `.golangci.yml`
- Strict custom boundary checker: `scripts/check_boundaries.go`
  - domain/application allowlists
  - no domain/application imports of adapters/internal/integrations/platform
  - no cross-module imports in `contexts/*`
- Developer commands:
  - `make lint` / `make test`
  - `scripts/lint.ps1` / `scripts/test.ps1`

## Migration Steps And Compatibility Notes

1. Update any references from module `contracts/` to `transport/`.
2. Consume stable contracts from `solomon/contracts` instead of runtime module paths.
3. Keep runtime imports on `internal/platform/*`; do not recreate top-level `platform/`.
4. For new modules, scaffold with `scripts/newmodule.*` instead of committing placeholder directories.
5. Run boundary gates before merge:
   - `go run ./scripts/check_boundaries.go`
   - `golangci-lint run`
   - `go test ./...`
