# Admin Dashboard Service

Configuration declaration: no runtime config; inherits platform defaults.

M86 Admin Dashboard Service module for Solomon monolith.

## Current Capabilities
- Idempotent admin action audit logging (`RecordAdminAction`)
- Hybrid control-plane identity role grant orchestration to owner authz module (`GrantIdentityRole`)
- Hybrid control-plane moderation decision orchestration to owner moderation module (`ModerateSubmission`)
- Hybrid control-plane abuse lockout release orchestration to owner abuse-prevention module (`ReleaseAbuseLockout`)
- In-memory module wiring for tests/bootstrap
- HTTP transport DTO + handler mapping layer

## Production Reliability Boundary
- `ReleaseAbuseLockout` is routed to M37 owner execution and depends on durable owner-side lockout/audit persistence.
- API bootstrap wiring uses persistent M37 adapters for this owner execution path.
- This preserves audit/idempotency reliability guarantees without introducing cross-module direct writes.

## Structure
- `domain/`: domain-level errors and invariants
- `application/`: admin action use cases
- `ports/`: repository/idempotency/clock interfaces
- `adapters/`: in-memory persistence and HTTP handler
- `transport/`: module-private HTTP DTOs
