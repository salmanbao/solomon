# Go Monolith Implementation

## Module Shape (Required)
For each module under `solomon/contexts/<context>/<service>`:
- `domain/`: entities, value objects, domain services, invariants
- `application/`: use cases, orchestration, command/query handlers
- `ports/`: module-owned inbound/outbound interfaces
- `adapters/`: concrete drivers (HTTP, DB, events, external clients)
- `transport/`: module-private transport DTOs and mapper payloads
- `module.go` optional: registration/wiring entry

Dependency direction is one-way:
- adapters -> application -> domain
- application -> ports
- adapters implement ports
- domain does not depend on adapters or framework code

## Go Design Rules
- Pass `context.Context` as first parameter for request-scoped operations.
- Inject dependencies via constructors; avoid hidden globals.
- Wrap errors with `%w` and operation context.
- Keep functions side-effect explicit.
- Keep domain logic deterministic and testable without DB/network.

## Entrypoints and Wiring
- API process: `solomon/cmd/api/main.go`
- Worker process: `solomon/cmd/worker/main.go`
- Composition root: `solomon/internal/app/bootstrap/bootstrap.go`
- Canonical platform runtime implementations: `solomon/internal/platform/*` only.

Wire modules in bootstrap through explicit constructors and interfaces.

## Data Access Pattern
- Repository interfaces live in `ports/`.
- Repositories/adapters implement owner-table writes only.
- For allowed monolith DBR reads (`internal_sql_readonly`), isolate in read-only query adapters and keep them separate from write repositories.
- For owner API dependencies, use integration client ports/adapters.

## Transactions and Outbox
- Start transaction at application use-case boundary when multiple writes must be atomic.
- Persist business state and outbox row in same transaction.
- Worker relays outbox rows and marks publish status.
- Preserve idempotency on publish and consume paths.

## API Adapter Pattern
- Validate request DTOs in adapter layer.
- Map DTO -> command/query for application layer.
- Return explicit error classes mapped to HTTP status codes.
- Keep transport concerns (headers, auth context, pagination) out of domain logic.

## Event Adapter Pattern
- Publish canonical events only after durable state commit.
- Include schema version and stable partition keys.
- In consumers, dedupe by `event_id` before state mutation.
- Handle retries and terminal failures with DLQ policy for domain-class events.

## Migrations
- Put DDL changes in `solomon/migrations`.
- Apply additive, backward-compatible schema changes first.
- Gate destructive changes behind phased rollout and data backfill.
- Keep migration ownership aligned to canonical owner service tables.

## Cross-Runtime Integration
- For microservice-owned data, define client ports + adapters inside the owning module under `contexts/<context>/<service>/adapters/...`.
- Keep `solomon/integrations/` as policy/docs only; never as a shared client dumping ground.
- Shared low-level client utilities may live in `solomon/internal/shared`.
- Add timeout, retry, and circuit-breaker policy at adapter level.
- Do not leak upstream provider DTOs into domain layer; map into module transport types.

## Recommended Per-Service Delivery Order
1. Confirm service is monolith in architecture map.
2. Read Mxx spec + ownership + dependencies + DB contracts.
3. Scaffold module directories with `scripts/newmodule.*`.
4. Implement domain and use cases.
5. Implement repository/client/event adapters.
6. Wire module in bootstrap and entrypoints.
7. Add migrations.
8. Add tests and run quality gates.
