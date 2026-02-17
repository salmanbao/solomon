# Internal Go Runtime

This folder contains Go runtime wiring for Solomon:

- `app/bootstrap`: composition root and dependency wiring
- `platform/*`: infra adapters (db, messaging, http, config, observability)
- `shared/*`: shared runtime patterns (event envelope, outbox)

## Runtime Data Flow

1. Inbound request/event enters adapter layer (`platform/httpserver` or worker consumer).
2. Adapter calls application use case in a module.
3. Use case executes domain logic and writes via repository port.
4. Domain event is persisted to outbox in the same transaction when required.
5. Worker relay publishes outbox rows to event bus for other microservices/modules.
