# Admin Dashboard Service

M86 Admin Dashboard Service module for Solomon monolith.

## Current Capabilities
- Idempotent admin action audit logging (`RecordAdminAction`)
- In-memory module wiring for tests/bootstrap
- HTTP transport DTO + handler mapping layer

## Structure
- `domain/`: domain-level errors and invariants
- `application/`: admin action use cases
- `ports/`: repository/idempotency/clock interfaces
- `adapters/`: in-memory persistence and HTTP handler
- `transport/`: module-private HTTP DTOs
