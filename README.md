# Solomon Monolithic Backend

`solomon` is the modular monolith backend for ViralForge in the hybrid architecture.

This app hosts all services marked `architecture: monolith` in `viralForge/specs/service-architecture-map.yaml`, while microservices remain independently deployable.

## Source Of Truth

- `viralForge/specs/00-Canonical-Structure.md`
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/service-deployment-profile.md`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `viralForge/specs/dependencies.yaml`

## Monolith Requirements

1. Architecture boundary
- Implement as a modular monolith with clear bounded modules per service.
- Keep each module independent in code ownership and interfaces.

2. Deployment routing
- Only services marked `architecture: monolith` are implemented inside `solomon`.
- Services marked `microservice` must not be absorbed into this runtime.

3. Data model and ownership
- Physical storage model: shared monolith database with service-scoped schemas.
- Logical ownership model: single-writer per owned table/service.
- Canonical ownership must match `DB-01` and `service-data-ownership-map.yaml`.

4. Cross-module data access
- Monolith to monolith read access can use `internal_sql_readonly` only where declared.
- No cross-module direct writes to non-owned tables.
- Cross-boundary reads to microservices must use owner API, event projection, or declared replica view.

5. Contracts and spec alignment
- Every implemented module must follow its `Mxx` spec contract sections.
- Keep `Data Ownership` and `5. Data Model` aligned with generated canonical artifacts.
- Respect `dependencies.yaml` DBR and EVENT relationships.

6. Reliability and operations baseline
- Apply canonical defaults from `00-Canonical-Structure.md`:
  - idempotency for mutating operations
  - outbox for DB write plus event publish consistency
  - retries, backoff, DLQ semantics where specified
  - logs, metrics, tracing, and alerting standards

## Solomon In-Scope Services (Monolith)

- M04-Campaign-Service
- M07-Editor-Dashboard-Service
- M08-Voting-Engine
- M09-Content-Library-Marketplace
- M15-Platform-Fee-Engine
- M20-Super-Admin-Dashboard
- M21-Authorization-Service
- M22-Onboarding-Service
- M23-Campaign-Discovery-Service
- M24-Clipping-Tool-Service
- M26-Submission-Service
- M31-Distribution-Service
- M34-Influencer-Dashboard-Service
- M35-Moderation-Service
- M37-Abuse-Prevention-Service
- M46-Chat-Service
- M47-Gamification-Service
- M48-Reputation-Service
- M49-Community-Health-Service
- M53-Discover-Service
- M60-Product-Service
- M61-Subscription-Service
- M62-Search-Service
- M65-Churn-Prevention-Service
- M74-CMS-Service
- M85-QA-Service
- M86-Admin-Dashboard-Service
- M87-Team-Management-Service
- M88-Localization-Service
- M92-Storefront-Service

## Non-Goals

- Implementing microservice-only modules in this repository folder.
- Redefining ownership, contracts, or dependency rules outside the canonical spec artifacts.

## Folder Structure (Commented)

```text
solomon/                                                # Monolith root for all services classified as architecture: monolith.
|-- README.md                                           # This file; project scope, constraints, and structure guide.
|-- go.mod                                              # Go module definition and dependency boundary for the monolith runtime.
|
|-- apps/                                               # Deploy/runtime packaging layer (entry wiring split by runtime role).
|   |-- README.md                                       # Describes app packaging conventions.
|   |-- api/                                            # Placeholder for API app-level assets/manifests.
|   `-- worker/                                         # Placeholder for async worker app-level assets/manifests.
|
|-- cmd/                                                # Executable entrypoints (thin main packages only).
|   |-- api/
|   |   `-- main.go                                     # Starts HTTP API process and delegates wiring to internal bootstrap.
|   `-- worker/
|       `-- main.go                                     # Starts background worker process for async/event workloads.
|
|-- contexts/                                           # Business bounded contexts (modular monolith domain slices).
|   |-- README.md                                       # Explains context/module layering standards.
|   |-- campaign-editorial/                             # Campaign/content workflows.
|   |-- community-experience/                           # Community, discovery, engagement, trust health.
|   |-- finance-core/                                   # Billing, subscriptions, fees, payouts-adjacent logic.
|   |-- identity-access/                                # Authz/onboarding/account access domain.
|   |-- internal-ops/                                   # Admin, QA, CMS, team/internal tooling.
|   `-- moderation-safety/                              # Abuse detection and moderation workflows.
|
|-- contracts/                                          # Public/stable contracts owned by the monolith boundary.
|   |-- README.md                                       # Contract governance and compatibility notes.
|   |-- api/                                            # HTTP/OpenAPI/API contract definitions.
|   |-- events/                                         # Event schemas/topics emitted/consumed.
|   `-- schemas/                                        # Shared payload/validation schemas.
|
|-- docs/                                               # Technical reference docs for architecture and engineering flow.
|   `-- go-structure-and-data-flow.md                   # Go layering and runtime data-flow explanation.
|
|-- integrations/                                       # Integration adapters and boundary notes.
|   |-- README.md                                       # Integration strategy and ownership notes.
|   |-- external/                                       # Third-party/external provider integration contracts.
|   `-- microservices/                                  # Calls/messages to independent microservices.
|
|-- internal/                                           # Private runtime code (non-importable outside module).
|   |-- README.md                                       # Internal package conventions.
|   |-- app/
|   |   `-- bootstrap/
|   |       `-- bootstrap.go                            # Application composition root (DI/wiring).
|   |-- platform/                                       # Technical infrastructure implementations.
|   |   |-- config/config.go                            # Configuration load/validation bootstrap.
|   |   |-- db/postgres.go                              # Postgres setup and connection lifecycle.
|   |   |-- httpserver/server.go                        # HTTP server construction/middleware lifecycle.
|   |   |-- messaging/kafka.go                          # Message bus client setup and publish/consume plumbing.
|   |   `-- observability/tracing.go                    # Tracing bootstrap and observability hooks.
|   `-- shared/                                         # Cross-context primitives kept intentionally small.
|       |-- events/envelope.go                          # Canonical event envelope model.
|       `-- outbox/outbox.go                            # Outbox primitive for DB+event consistency.
|
|-- migrations/                                         # Schema migration files and migration run guidance.
|   `-- README.md                                       # Migration conventions and ownership rules.
|
|-- platform/                                           # Reserved platform capability surface (domain-agnostic modules).
|   |-- README.md                                       # Platform capability boundaries.
|   |-- cache/                                          # Shared cache capability.
|   |-- config/                                         # Shared config capability.
|   |-- db/                                             # Shared database capability abstractions.
|   |-- eventing/                                       # Eventing primitives and helpers.
|   |-- messaging/                                      # Messaging abstractions and adapters.
|   |-- observability/                                  # Metrics/logging/tracing utilities.
|   `-- security/                                       # Security primitives (authn/authz/crypto policy hooks).
|
`-- tests/                                              # Test suites separated by scope and confidence level.
    |-- README.md                                       # Testing strategy and execution guidance.
    |-- unit/                                            # Fast isolated tests for domain/application logic.
    |-- integration/                                     # DB/message/API boundary integration tests.
    |-- contract/                                        # API/event contract compatibility tests.
    `-- e2e/                                             # End-to-end scenario tests across modules.
```

### Context Module Template (Used Across Services)

```text
contexts/<context-name>/<service-name>/                # One service module inside a bounded context.
|-- README.md                                          # Service intent, boundaries, and mapping to Mxx spec.
|-- module.go                                          # (Optional) module registration/wiring entry for this service.
|-- domain/                                            # Entities, value objects, domain services (pure business logic).
|-- application/                                       # Use-cases (commands/queries/orchestration).
|-- ports/                                             # Inbound/outbound interfaces owned by the module.
|-- adapters/                                          # Interface implementations (HTTP, DB, events, external calls).
`-- contracts/                                         # Module-local DTOs/events/public payload contracts.
```

### Implemented Example Module (Current)

```text
contexts/identity-access/authorization-service/        # Most complete reference implementation in scaffold.
|-- README.md                                          # Service boundary and implementation notes.
|-- module.go                                          # Registers authorization module dependencies.
|-- domain/entities/role.go                            # Role aggregate/entity rules.
|-- domain/valueobjects/user_id.go                     # Strongly typed user identity value object.
|-- domain/services/policy_engine.go                   # Core authorization decision logic.
|-- application/commands/assign_role.go                # Write use-case for role assignment.
|-- application/queries/list_permissions.go            # Read use-case for effective permissions.
|-- ports/repository.go                                # Persistence contract required by application/domain.
|-- ports/event_publisher.go                           # Domain event publishing contract.
|-- adapters/postgres/repository.go                    # Postgres implementation of repository port.
|-- adapters/http/handler.go                           # HTTP adapter exposing authorization endpoints.
|-- adapters/events/publisher.go                       # Event bus adapter for authorization events.
`-- contracts/http_dto.go                              # Request/response DTO contracts for HTTP boundary.
```
