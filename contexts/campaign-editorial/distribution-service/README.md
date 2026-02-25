# Distribution Service (M31)

## Responsibility and Boundary
`contexts/campaign-editorial/distribution-service` manages distribution item orchestration:
- claim-to-item creation
- scheduling and rescheduling windows
- overlay metadata registration
- publish/retry state transitions
- preview and download response contracts

Boundary rules:
- owner-write only to M31 tables
- cross-service dependencies (M06/M09/M10) remain contract-level integrations and are not direct write targets

## Inbound Adapters and Contracts
- Transport adapter: `adapters/http/handler.go`
- DTOs: `transport/http/http_dto.go`
- Routes: `/api/v1/distribution/items/{id}/*` in `internal/platform/httpserver/server.go`
- API contract artifact: `contracts/api/v1/distribution-service.openapi.json`

## Use-Case Flow and Invariants
- `Claim`: creates a 24-hour claim window item.
- `Schedule`: validates schedule within `[now + 5m, now + 30d]`.
- `PublishMulti`: transitions item to published and records per-platform status entries.
- `Retry`: only allowed from failed state.
- `AddOverlay`: enforces duration bounds and state restrictions.

Core invariants:
- invalid state transitions are rejected
- overlay duration must stay within 3 seconds
- scheduled timestamps must remain within explicit window constraints

## Owned Data and Read Dependencies
Migration:
- `migrations/20260225_0005_m31_distribution_service.sql`

Owned tables align with canonical M31 inventory:
- `distribution_items`, `distribution_captions`, `distribution_overlays`, `distribution_platform_status`, `publishing_analytics`

## Event and Outbox Behavior
Canonical M31 dependencies and interfaces are preserved at contract level. This iteration does not yet run a dedicated outbox relay worker for M31 module-internal events.

## Failure Handling and Idempotency
- Domain errors mapped in `internal/platform/httpserver/server.go`
- state conflict paths use explicit domain error types
- schedule boundary violations return bad request semantics

## Testing Coverage Map
Unit tests:
- `tests/unit/distribution_service_test.go`
  - schedule window validation
  - publish multi-platform success path

## Decision Rationale
### Decision
Deliver M31 with minimal core operational flows and strict state guards.

### Context
M31 in canonical specs is monolith-scoped but MVP scope documents are mixed; implementation needed to unblock pipeline integration while staying boundary-compliant.

### Alternatives Considered
- Leave M31 unimplemented (README only): rejected due explicit MVP implementation request.
- Full external API posting adapters in first pass: rejected to avoid coupling and runtime instability.

### Tradeoffs
- Fast delivery of contract-compatible endpoints and state machine.
- External platform integrations and worker reliability are deferred.

### Consequences
- Internal callers can integrate against stable API shape now.
- Follow-up slices can add platform clients/outbox without breaking route contracts.

### Evidence
- Code: `contexts/campaign-editorial/distribution-service/*`
- Routing: `internal/platform/httpserver/server.go`
- Migration: `migrations/20260225_0005_m31_distribution_service.sql`

