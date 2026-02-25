# Solomon Monolithic Backend

`solomon/` hosts only services marked `architecture: monolith` in:
`viralForge/specs/service-architecture-map.yaml`.

Microservices remain in `mesh/` and must not depend on Solomon runtime internals.

## Source Of Truth

- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- service specs `viralForge/specs/Mxx-*.md`

## Enforced Architecture Rules

- Single canonical platform layer: `internal/platform/*` only.
- Module layering: `domain -> application -> ports -> adapters -> transport`.
- No cross-module imports inside `contexts/*` (except root contracts module).
- No direct cross-runtime DB writes.
- Single-writer ownership per canonical table.
- Outbox + idempotency required for mutating/event flows.

## Directory Semantics

```text
solomon/
|-- cmd/                    # Process entrypoints.
|-- contexts/               # Bounded-context monolith modules.
|-- contracts/              # Separate Go module for stable cross-runtime contracts.
|-- deploy/                 # Non-Go deployment assets.
|-- docs/                   # Architecture and engineering docs.
|-- integrations/           # Policy/docs only; not a shared client dumping ground.
|-- internal/               # Runtime wiring, platform implementations, shared helpers.
|-- migrations/             # DB migrations.
|-- scripts/                # Boundary and scaffolding tooling.
|-- tests/                  # Unit/integration/contract/e2e test organization.
|-- .golangci.yml           # Lint gates (depguard).
|-- go.mod                  # Solomon runtime module.
`-- go.work                 # Workspace linking runtime + contracts module.
```

## Contracts Governance

Root contracts live in `contracts/` (separate module `solomon/contracts`) and contain:

- versioned API/event/schema artifacts (`api/v{n}`, `events/v{n}`, `schemas/v{n}`)
- generated contract types (`gen/...`)

Versioning:

- additive changes within a major version
- breaking changes require a new major version directory (`v2`, `v3`, ...)

Must NOT be in root contracts:

- module-private transport DTOs
- runtime adapters/wiring/business logic

Module-private DTOs belong in:
`contexts/<context>/<service>/transport/...`

## Integrations Rule

Do not add shared microservice clients under `integrations/`.
Each module owns outbound adapters in its own `adapters/` directory and exposes outbound dependencies via `ports/`.

## Quality Gates

```bash
make test
make lint
```

PowerShell alternatives:

```powershell
./scripts/test.ps1
./scripts/lint.ps1
```

`make lint` runs:

1. `go run ./scripts/check_boundaries.go`
2. `golangci-lint run`
