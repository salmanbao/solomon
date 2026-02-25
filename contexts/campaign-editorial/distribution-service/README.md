# Distribution Service (M31)

## Responsibility and Boundary
`contexts/campaign-editorial/distribution-service` orchestrates the influencer distribution lifecycle for claimed clips:
- ingest distribution claims from M09 (`distribution.claimed`)
- maintain distribution item state machine (`claimed -> scheduled -> publishing -> published/failed`)
- register overlays and caption metadata on owned tables
- process scheduled publishing and manual publish/retry requests
- emit durable outcome events from module outbox

Boundary rules:
- owner-write only to M31 canonical tables:
  `distribution_items`, `distribution_captions`, `distribution_overlays`,
  `distribution_platform_status`, `publishing_analytics`
- no direct writes to foreign-owned tables
- canonical read dependencies are respected:
  - M06 via `owner_api`
  - M09 via `internal_sql_readonly` (clip -> campaign projection lookup only)
  - M10 via `owner_api` (integration client path reserved; not yet implemented)

## Inbound Adapters and Contracts
- Transport adapter: `adapters/http/handler.go`
- DTOs: `transport/http/http_dto.go`
- Worker adapters:
  - `application/workers/claimed_consumer.go`
  - `application/workers/scheduler_job.go`
  - `application/workers/outbox_relay.go`
- HTTP routes in `internal/platform/httpserver/server.go`:
  - `POST /api/v1/distribution/items/{id}/overlays`
  - `GET /api/v1/distribution/items/{id}/preview`
  - `POST /api/v1/distribution/items/{id}/schedule`
  - `POST /api/v1/distribution/items/{id}/reschedule`
  - `POST /api/v1/distribution/items/{id}/publish`
  - `POST /api/v1/distribution/items/{id}/download`
  - `POST /api/v1/distribution/items/{id}/publish-multi`
  - `POST /api/v1/distribution/items/{id}/retry`
- API contract artifact: `contracts/api/v1/distribution-service.openapi.json`

## Use-Case Flow and Invariants
Core command behaviors (`application/commands/commands.go`):
- `Claim`: creates a 24-hour claim window distribution item.
- `AddOverlay`: allows only `intro|outro`; duration must be `(0,3]` seconds.
- `Schedule`: validates window `[now + 5m, now + 30d]`; validates platform.
- `Reschedule`: only allowed when current status is `scheduled`.
- `PublishMulti` and `Publish`: transition through `publishing` to `published`,
  persist platform status rows and publishing analytics rows, and enqueue outbox event.
- `Retry`: only allowed from `failed`; resets retry counter and re-enters publish path.
- `ProcessDueScheduled`: worker-safe batch publish of due `scheduled` items.

Worker flow:
- `ClaimedConsumer` consumes `distribution.claimed` and projects M09 claim rows into M31 items.
- `SchedulerJob` runs periodic due-item publishing.
- `OutboxRelay` publishes pending M31 outbox rows and marks them published.

Core invariants:
- invalid state transitions are rejected
- overlay duration must stay within 3 seconds
- platform set is constrained to supported values (`tiktok`, `instagram`, `youtube`, `snapchat`)
- schedule timestamps must remain within explicit window constraints
- influencer ownership is enforced for schedule/publish/retry mutations

## Owned Data and Read Dependencies
Migration:
- `migrations/20260225_0005_m31_distribution_service.sql`
- `migrations/20260225_0011_m31_distribution_reliability.sql`

Owned tables align with canonical M31 inventory:
- `distribution_items`, `distribution_captions`, `distribution_overlays`, `distribution_platform_status`, `publishing_analytics`

Read dependency notes:
- M09 lookup is implemented via readonly projection query to `clips` (`GetCampaignIDByClip`) in
  `adapters/postgres/repository.go` to resolve claim ingestion context.

## Event and Outbox Behavior
Canonical event envelope is used for module-emitted events (`contracts/gen/events/v1/envelope.go`).

Current emitted event path:
- `distribution.published` via `distribution_outbox` with at-least-once relay semantics.

Current consumed event path:
- `distribution.claimed` (from M09 claim outbox relay topic), consumed by `ClaimedConsumer`.

## Failure Handling and Idempotency
- Domain errors mapped in `internal/platform/httpserver/server.go`
- explicit error mapping for:
  - `not_found`
  - validation (`invalid_request`, `unsupported_platform`)
  - authorization (`forbidden`)
  - transition conflicts (`invalid_state_transition`)
- outbox relay and consumer paths log structured failure events with contextual fields.

Idempotency behavior:
- claim ingestion is naturally idempotent by reusing M09 `claim_id` as M31 `distribution_items.id`
  plus uniqueness guard on `(influencer_id, clip_id, campaign_id)`.
- outbox publish ack uses row status transition (`pending -> published`).

## Testing Coverage Map
Unit tests:
- `tests/unit/distribution_service_test.go`
  - schedule window validation
  - publish multi-platform success path
  - reschedule state guard
  - timezone parsing behavior
- `tests/unit/distribution_service_workers_test.go`
  - claim event ingestion path
  - due-schedule publish and outbox emission path

## Decision Rationale
### Decision
Implement M31 as a monolith module with PostgreSQL persistence, worker-based claim/schedule processing,
and explicit outbox delivery for publish outcomes.

### Context
M31 is canonical `architecture: monolith` and sits on the MVP content pipeline critical path.
The module needed production-safe wiring (not in-memory only), explicit ownership boundaries,
and extraction-ready contracts.

### Alternatives Considered
- Keep in-memory module in bootstrap: rejected because it breaks persistence and worker-driven flows.
- Add direct cross-module writes for claim ingestion: rejected due single-writer ownership policy.
- Implement full provider API clients immediately: rejected for scope/risk; contract-compatible stubs retained.

### Tradeoffs
- Improves reliability and observability via outbox + worker model.
- Adds operational moving parts (consumer, scheduler, relay) and migration complexity.
- External platform-specific publishing semantics remain intentionally simplified in this slice.

### Consequences
- API and worker callers can integrate against stable route/event surfaces now.
- Module remains extraction-ready: contracts explicit, boundaries enforced, persistence encapsulated in adapters.
- Future slices can extend provider adapters and richer retry/backoff without transport contract churn.

### Evidence
- Code: `contexts/campaign-editorial/distribution-service/*`
- Routing: `internal/platform/httpserver/server.go`
- Migrations:
  - `migrations/20260225_0005_m31_distribution_service.sql`
  - `migrations/20260225_0011_m31_distribution_reliability.sql`
- Canonical specs:
  - `viralForge/specs/service-architecture-map.yaml`
  - `viralForge/specs/dependencies.yaml`
  - `viralForge/specs/service-data-ownership-map.yaml`
  - `viralForge/specs/M31-Distribution-Service.md`
