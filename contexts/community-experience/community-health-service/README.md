# Community Health Service

Module scaffold for Solomon monolith.

## Structure
- domain/: entities, value objects, domain services, invariants
- pplication/: use cases, command/query handlers, orchestration
- ports/: repository, event, and client interfaces
- dapters/: DB, HTTP/gRPC, event bus, cache implementations
- contracts/: request/response/event contracts local to this module
