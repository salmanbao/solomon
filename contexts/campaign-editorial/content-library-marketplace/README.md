# Content Library Marketplace (M09)

Developer documentation for
`contexts/campaign-editorial/content-library-marketplace`.

## Responsibility And Boundary

- Implements `M09-Content-Library-Marketplace` as a Solomon monolith module.
- Canonical architecture: `monolith` (`viralForge/specs/service-architecture-map.yaml`).
- Canonical direct dependencies: none (`viralForge/specs/dependencies.yaml`, `M09.depends_on: []`).
- Module layering is enforced as:
  `domain -> application -> ports -> adapters -> transport`.
- Business logic stays in module packages only. Runtime HTTP wiring lives in
  `internal/platform/httpserver`.

## Inbound Surface And Contracts

Current HTTP endpoints exposed through monolith server:

- `GET /library/clips`
- `GET /library/clips/{clip_id}`
- `GET /library/clips/{clip_id}/preview`
- `POST /library/clips/{clip_id}/claim`
- `POST /library/clips/{clip_id}/download`
- `GET /library/claims`
- `GET /v1/marketplace/clips`
- `GET /v1/marketplace/clips/{clip_id}`
- `GET /v1/marketplace/clips/{clip_id}/preview`
- `POST /v1/marketplace/clips/{clip_id}/claim`
- `POST /v1/marketplace/clips/{clip_id}/download`
- `GET /v1/marketplace/claims`

Header policy:

- Canonical `/v1/marketplace/*` routes enforce `Authorization: Bearer ...` and
  `X-Request-Id`.
- User-scoped `/v1/marketplace/*` routes also require `X-User-Id`.
- `POST /v1/marketplace/clips/{clip_id}/claim` requires `Idempotency-Key`.
- Legacy `/library/*` aliases are retained for compatibility; claim idempotency
  key remains optional there.

Code references:

- Handler adapter: `adapters/http/handler.go`
- Transport DTOs: `transport/http/http_dto.go`
- Runtime route registration: `internal/platform/httpserver/server.go`
- Swagger annotations/docs:
  - handler annotations in `adapters/http/handler.go`
  - generated artifacts in `internal/platform/httpserver/docs`
  - UI route: `/swagger/index.html`
- Stable versioned API contract artifact:
  - `contracts/api/v1/content-library-marketplace.openapi.json`
- Stable event payload contracts:
  - `contracts/events/v1/distribution.claimed.schema.json`
  - `contracts/events/v1/distribution.published.schema.json`
  - `contracts/events/v1/distribution.failed.schema.json`

## Module Map

- `domain/entities`: `Clip`, `Claim`, value-level rules.
- `domain/services`: claim eligibility policy.
- `domain/errors`: stable module errors mapped to HTTP statuses.
- `application/queries`: list clips, get clip, list claims.
- `application/commands`: claim clip mutation with idempotency and outbox hook.
- `ports`: repository, idempotency store, clock, and ID generator contracts.
- `adapters/memory`: in-memory implementation used for current runnable slice.
- `module.go`: in-memory composition for local/runtime wiring.

## Use-Case Flow And Invariants

### `ListClips`

- Defaults to `limit=20`; caps to `50`.
- Defaults status to `active` when omitted.
- Supports optional sort by `views_7d`, `votes_7d`, `engagement_rate`.
- Uses cursor pagination.

### `GetClip`

- Fetches clip by `clip_id`.
- Returns `ErrClipNotFound` for unknown clip.

### `ClaimClip`

1. Validate required fields: `clip_id`, `user_id`, `request_id`.
2. Resolve idempotency key:
   - request header key when provided
   - fallback: `cms:{user_id}:{clip_id}:claim`
3. Load idempotency record and request hash-check to prevent key reuse drift.
4. Replay existing claim when key/request already exists.
5. Load clip and clip claims.
6. Apply domain policy:
   - clip must be `active`
   - exclusive clip: at most one occupying claim
   - non-exclusive clip: occupying claims must stay below effective claim limit
   - same user existing occupying claim is returned (no duplicate claim row)
7. Create new claim and outbox event in one repository call.
8. Persist idempotency record (TTL default: 7 days).

Domain invariants:

- `Clip.IsClaimable()` allows only `active`.
- `Clip.EffectiveClaimLimit()`:
  - `exclusive -> 1`
  - `non_exclusive -> claim_limit`, fallback `50` when invalid.
- `Claim.NewClaim(...)` requires non-empty IDs, valid claim type, and
  `expires_at > claimed_at`.
- `Claim.OccupiesSlot(now)` returns true for:
  - `active` and not expired
  - `published` regardless of expiry

## Data Ownership And Dependencies

Canonical owner tables (DB-01):

- `clips` (`clip_id`)
- `clip_claims` (`claim_id`)
- `clip_downloads` (`download_id`)

Read dependencies (canonical): none.

Shared surface consumers (DB-02):

- `owner_api`: M05, M39, M55
- `internal_sql_readonly`: M31, M34

Current migration baseline:

- `migrations/20260225_0001_m09_content_library_marketplace.sql`
- Includes owned tables above plus implementation support tables:
  - `content_marketplace_idempotency`
  - `content_marketplace_outbox`

## Event And Outbox Behavior

- Command layer currently emits logical event type `distribution.claimed`.
- Port contract `CreateClaimWithOutbox` enforces "write claim + outbox" as one
  repository operation boundary.
- In-memory adapter appends outbox entries to in-process store for testability.
- Worker runtime includes:
  - outbox relay for `distribution.claimed`
  - consumer for `distribution.published` and `distribution.failed`
  - claim expiry sweep worker

## Failure Handling And Idempotency

Primary domain/application errors:

- `ErrInvalidClaimRequest` -> bad request
- `ErrClipNotFound`, `ErrClaimNotFound` -> not found
- `ErrExclusiveClaimConflict`, `ErrClaimLimitReached` -> conflict
- `ErrIdempotencyKeyConflict` -> conflict

HTTP error mapping currently lives in
`internal/platform/httpserver/server.go::writeDomainError`.

Idempotency semantics:

- Keyed by idempotency key with request hash.
- Same key + different request payload is rejected.
- Duplicate `request_id` replays existing claim.
- Default idempotency retention in command layer: 7 days.

## Testing Coverage Map

Unit tests:

- `tests/unit/content_library_marketplace_test.go`
  - exclusive claim conflict path
  - idempotent replay path
  - clip listing pagination path

Boundary and compile gates:

- `go test ./...`
- `go run ./scripts/check_boundaries.go`

## Operational Notes

- API runtime wires M09 with Postgres repository in
  `internal/app/bootstrap/bootstrap.go::BuildAPI`.
- Worker runtime wires outbox relay and distribution consumers in
  `internal/app/bootstrap/bootstrap.go::BuildWorker`.
- In-memory store remains the unit-test/local harness adapter.
- API process entrypoint: `cmd/api/main.go`.

## Decision Rationale

### Decision

- Start M09 with a complete domain/application contract and in-memory adapters
  before platform-specific persistence/event plumbing.

### Context

- Repository bootstrap is still minimal; this allows validating behavior and
  boundaries now without blocking on unfinished infra.

### Alternatives Considered

- Build Postgres/Kafka adapters first.
  - Rejected for now because it would couple module progress to incomplete
    global runtime wiring.
- Keep only scaffolding docs.
  - Rejected because it hides real invariants and makes unsafe edits likely.

### Tradeoffs

- Improves testability and implementation clarity immediately.
- Adds temporary divergence between in-memory behavior and eventual DB/runtime
  concerns (locks, transaction isolation, relay behavior).

### Consequences

- Module is extraction-ready at interfaces/ports level.
- Infrastructure replacement can occur under ports without rewriting domain and
  application logic.
- Future work should keep canonical ownership semantics unchanged while swapping
  adapters.

### Evidence

- Module code in `contexts/campaign-editorial/content-library-marketplace/*`
- Migration: `migrations/20260225_0001_m09_content_library_marketplace.sql`
- Canonical references:
  - `viralForge/specs/service-architecture-map.yaml`
  - `viralForge/specs/dependencies.yaml`
  - `viralForge/specs/DB-01-Data-Contracts.md`
  - `viralForge/specs/DB-02-Shared-Data-Surface.md`
