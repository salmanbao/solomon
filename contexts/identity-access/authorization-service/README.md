# Authorization Service (M21)

M21 implements monolith RBAC authorization in `contexts/identity-access/authorization-service`.

## Responsibility and Boundary
- Context: `identity-access`
- Service: `authorization-service`
- Runtime: monolith module in Solomon
- Responsibility: permission checks, role grant/revoke, and temporary delegation orchestration
- Boundary rule: domain/application layers do not import adapters or other contexts

## Inbound Contracts and Adapters
- HTTP transport adapter:
  - `adapters/http/handler.go`
  - DTOs in `transport/http/http_dto.go`
- Registered routes:
  - `POST /api/authz/v1/check`
  - `POST /api/authz/v1/check-batch`
  - `GET /api/authz/v1/users/{user_id}/roles`
  - `POST /api/authz/v1/users/{user_id}/roles/grant`
  - `POST /api/authz/v1/users/{user_id}/roles/revoke`
  - `POST /api/authz/v1/delegations`
- Stable versioned API contract artifact:
  - `contracts/api/v1/authorization-service.openapi.json`
- Stable event payload contract:
  - `contracts/events/v1/authz.policy_changed.schema.json`

## Use-Case Flow and Invariants
- Permission check (`application/queries/check_permission.go`):
  - validates `user_id` and `permission`
  - reads cache first, falls back to repository
  - deny-by-default on lookup failures
- Grant/revoke/delegation commands:
  - validate ids and command invariants
  - require idempotency key
  - hash request payload and enforce replay semantics
  - persist state mutation + audit + outbox through repository transaction
  - invalidate per-user permission cache after role changes

## Data Ownership and Read Dependencies
- Canonical dependency terminology:
  - `DBR:M01-Authentication-Service` via `owner_api`
  - `DBR:M87-Team-Management-Service` via `internal_sql_readonly`
- Current implementation uses module-local persistence ports for:
  - role assignments
  - delegations
  - idempotency records
  - outbox rows
  - event dedupe rows
- Important: canonical DB ownership docs currently mark M21 as owning no tables, while functional M21 spec expects persistent authz state. This module keeps persistence behind ports to allow contract alignment without reworking application/domain layers.

## Event and Outbox Behavior
- Mutating flows enqueue `authz.policy_changed` payloads through transactional outbox writes.
- Worker primitives:
  - `application/workers/outbox_relay.go`: publishes pending outbox messages
  - `application/workers/policy_changed_consumer.go`: dedupes by `event_id` and invalidates affected user cache

## Failure Handling and Idempotency
- Idempotency keys are mandatory for mutating endpoints.
- Duplicate key + identical request hash returns replayed response.
- Duplicate key + different request hash returns conflict.
- Cache invalidation failure after successful mutation is logged as warning; command still returns success.

## Testing Coverage Map
- Unit tests: `tests/unit/authorization_service_test.go`
  - grant + permission check success
  - grant idempotency replay
  - idempotency conflict
  - revoke behavior
  - delegation expiry validation

## Decision Rationale
### Decision
- Implement M21 as a layered, extraction-ready module with explicit ports for repository/cache/idempotency/outbox.

### Context
- M21 spec requires idempotent mutations, outbox reliability, and dedupe behavior.
- Canonical ownership tables and functional spec are not fully aligned yet.

### Alternatives Considered
- Inline all persistence assumptions in HTTP handlers: rejected due to boundary violations and lower extraction readiness.
- Keep only a stub module until ownership alignment: rejected because it blocks API-level integration and reliability semantics.

### Tradeoffs
- Improves: maintainability, testability, and migration flexibility.
- Cost: additional adapter/port abstractions before full runtime integration.

### Consequences
- Allows independent evolution of storage implementation and contracts.
- Enables staged rollout of worker wiring and observability without changing application behavior.
