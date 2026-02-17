# Solomon Go Structure And Data Flow

## Why This Layout

This scaffold keeps domain logic isolated from frameworks and infrastructure.
It supports modular growth inside the monolith and clean communication with microservices.

## Top-Level Runtime Layout

- `cmd/api`: HTTP/gRPC process entrypoint
- `cmd/worker`: async processing process entrypoint
- `internal/app/bootstrap`: composition root
- `internal/platform/*`: infra adapters
- `internal/shared/*`: shared technical patterns
- `contexts/*`: bounded-context business modules

## Per-Module Hexagonal Layout

Each module follows:

- `domain/`: business rules and invariants
- `application/`: use cases and orchestration
- `ports/`: interfaces required by use cases
- `adapters/`: concrete implementations (db/http/events)
- `contracts/`: transport and event contract DTOs

## Request Data Flow

1. Transport adapter receives request.
2. Adapter maps request DTO to command/query.
3. Application use case executes domain logic.
4. Use case writes state via repository port.
5. If needed, use case records domain event to outbox in same transaction.
6. Response DTO is mapped back by adapter.

## Event Data Flow To Other Microservices

1. Outbox relay (worker) loads pending outbox rows.
2. Relay publishes canonical event envelope to event bus.
3. On success, outbox row is marked published.
4. On failure, retry policy applies; terminal failures go to DLQ policy.
