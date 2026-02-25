# Voting Engine (M08)

## Responsibility and Boundary
`contexts/campaign-editorial/voting-engine` provides:
- vote creation/update via idempotent key semantics
- vote retraction
- submission/campaign/global leaderboard aggregation
- round-style and analytics read endpoints

Boundary rules:
- M08 is monolith-scoped; it reads M04/M26/M48 contracts and does not write foreign-owned tables.
- This implementation uses the legacy M27 vote table surface for runtime compatibility.

## Inbound Adapters and Contracts
- Transport adapter: `adapters/http/handler.go`
- DTOs: `transport/http/http_dto.go`
- Routes: `/v1/votes*`, `/v1/leaderboards/*`, `/v1/rounds/*`, `/v1/analytics/votes`
- API contract artifact: `contracts/api/v1/voting-engine.openapi.json`

## Use-Case Flow and Invariants
- `CreateVote`: validates request, enforces idempotency replay/conflict behavior, upserts by `(user_id, submission_id)` identity.
- `RetractVote`: marks vote retracted and prevents repeat retraction.
- `LeaderboardUseCase`: computes score aggregates from active votes only.

Core invariants:
- one active identity vote path per `(user, submission)`
- invalid vote types rejected
- retracted votes contribute `0` score

## Owned Data and Read Dependencies
Schema migration:
- `migrations/20260225_0006_m27_voting_tables_for_m08.sql`

This preserves M27 legacy table identifiers (`votes`, `voting_rounds`, `vote_quarantine`) while M08 remains the runtime successor in code.

## Event and Outbox Behavior
Canonical M08 events remain:
- `vote.created`, `vote.updated`, `vote.retracted`, `voting_round.closed`

Current slice implements API behavior and idempotency persistence; event outbox relay remains a follow-up.

## Failure Handling and Idempotency
- Domain errors mapped in `internal/platform/httpserver/server.go`
- idempotency key conflicts produce conflict semantics
- vote not found and submission not found map to not-found responses

## Testing Coverage Map
Unit tests:
- `tests/unit/voting_engine_test.go`
  - create + replay
  - retract and weighted score reset

## Decision Rationale
### Decision
Use an in-memory-first M08 module with explicit ports and deterministic leaderboard aggregation.

### Context
M08 is in MVP monolith scope and depends on M26/M04 availability; quick integration required while preserving extraction-ready design.

### Alternatives Considered
- Direct SQL-first voting implementation: rejected for slower rollout and tighter coupling.
- No idempotency support: rejected because canonical spec requires mutating idempotent semantics.

### Tradeoffs
- High implementation velocity and testability.
- No worker/outbox reliability path yet for vote events.

### Consequences
- API and domain flows are stable and test-backed.
- Future persistence or event infrastructure can be added behind ports without transport changes.

### Evidence
- Code: `contexts/campaign-editorial/voting-engine/*`
- Routing: `internal/platform/httpserver/server.go`
- Migration: `migrations/20260225_0006_m27_voting_tables_for_m08.sql`

