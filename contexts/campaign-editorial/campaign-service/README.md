# Campaign Service (M04)

## Responsibility and Boundary
`contexts/campaign-editorial/campaign-service` owns campaign lifecycle inside Solomon monolith:
- campaign creation/update
- lifecycle transitions (`draft`, `active`, `paused`, `completed`)
- budget increase tracking
- media confirmation metadata

Boundary rules:
- writes only to M04-owned tables (`campaigns`, `campaign_media`, `campaign_budget_log`, `campaign_state_history`)
- reads to non-owned services remain contract-level (`owner_api` for M01/M02, `internal_sql_readonly` only where declared)

## Inbound Adapters and Contracts
- Transport adapter: `adapters/http/handler.go`
- DTOs: `transport/http/http_dto.go`
- Routed endpoints are registered in `internal/platform/httpserver/server.go` under `/v1/campaigns/*`
- Stable API contract artifact: `contracts/api/v1/campaign-service.openapi.json`

## Use-Case Flow and Invariants
- `CreateCampaignUseCase`: validates required fields, budget/rate bounds, allowed platforms, and idempotency replay behavior.
- `UpdateCampaignUseCase`: only editable in `draft` or `paused`.
- `ChangeStatusUseCase`: explicit state transitions with history recording.
- `IncreaseBudgetUseCase`: additive updates only, immutable audit in budget log.
- `GenerateUploadURLUseCase` + `ConfirmMediaUseCase`: upload intent and media confirmation bookkeeping.

Core invariants:
- idempotency key required for create
- budget increase must be positive
- completed campaigns are immutable for lifecycle and budget operations

## Owned Data and Read Dependencies
Canonical ownership is aligned to:
- `viralForge/specs/service-data-ownership-map.yaml` (`M04-Campaign-Service`)
- `viralForge/specs/DB-01-Data-Contracts.md` table inventory

Migration file:
- `migrations/20260225_0003_m04_campaign_service.sql`

## Event and Outbox Behavior
This iteration defines module logic and HTTP contract surfaces. Canonical event names remain:
- `campaign.created`, `campaign.launched`, `campaign.paused`, `campaign.resumed`, `campaign.completed`, `campaign.budget_updated`

Outbox relay wiring is not yet enabled for M04 in bootstrap; it is a follow-up hardening slice.

## Failure Handling and Idempotency
- Domain errors are mapped to HTTP status in `internal/platform/httpserver/server.go`
- Create path stores idempotency request hash + response payload and replays safely
- Invalid state transitions return conflict semantics

## Testing Coverage Map
Unit tests:
- `tests/unit/campaign_service_test.go`
  - create + idempotency replay
  - invalid transition rejection

## Decision Rationale
### Decision
Implement M04 first with an in-memory adapter while preserving full layered boundaries.

### Context
M04 is the earliest monolith service in MVP dependency order and unblocks M26/M08 flows.

### Alternatives Considered
- Direct DB-first implementation: rejected for slower multi-module throughput.
- Handler-only implementation: rejected because it would violate domain/application separation.

### Tradeoffs
- Faster module rollout and deterministic tests.
- Outbox/runtime persistence for M04 events remains a follow-up.

### Consequences
- Extraction-ready boundaries are maintained.
- API contracts and migrations exist now; persistence hardening can be done without changing transport contracts.

### Evidence
- Code: `contexts/campaign-editorial/campaign-service/*`
- Routing: `internal/platform/httpserver/server.go`
- Migration: `migrations/20260225_0003_m04_campaign_service.sql`
- Canonical refs: `viralForge/specs/service-architecture-map.yaml`, `viralForge/specs/dependencies.yaml`

