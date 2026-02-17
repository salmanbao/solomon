# Platform Fee Engine

Module scaffold for Solomon monolith.

## Structure
- domain/: entities, value objects, domain services, invariants
- application/: use cases, command/query handlers, orchestration
- ports/: repository, event, and client interfaces
- adapters/: DB, HTTP/gRPC, event bus, cache implementations
- contracts/: request/response/event contracts local to this module
