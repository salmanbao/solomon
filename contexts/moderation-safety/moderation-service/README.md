# Moderation Service

Module scaffold for Solomon monolith.

## Structure
- domain/: entities, value objects, domain services, invariants
- application/: use cases, command/query handlers, orchestration
- ports/: repository, event, and client interfaces
- adapters/: DB, HTTP/gRPC, event bus, cache implementations
- transport/: module-private transport DTOs and event payload mappers
