---
name: solomon-monolith-backend
description: Implement, refactor, review, and harden Solomon monolith backend services in Go using ViralForge canonical architecture, dependency, ownership, and contract specs. Use this skill when work touches `solomon/` modules, monolith service mapping from `viralForge/specs/service-architecture-map.yaml`, module layering across domain, application, ports, adapters, and transport, outbox event reliability, migrations, cross-runtime boundaries with `mesh`, or test and quality gates for monolith services.
---

# Solomon Monolith Backend

Apply this skill to keep Solomon changes aligned with canonical contracts while preserving modular monolith boundaries.

## Quick Start
1. Read `references/canonical-constraints.md`.
2. Read `references/monolith-service-catalog.md`.
3. Read `references/go-monolith-implementation.md`.
4. Use `references/change-workflow.md` to execute the change.
5. Use `references/quality-gates-and-testing.md` before finishing.

## Reference Loading Rules
- Always load:
  - `references/canonical-constraints.md`
  - `references/monolith-service-catalog.md`
  - `references/go-monolith-implementation.md`
- Load when planning multi-service rollout:
  - `references/service-grouping-and-build-strategy.md`
- Load when validating or reviewing:
  - `references/quality-gates-and-testing.md`
- Load when implementing a concrete task:
  - `references/change-workflow.md`

## Workflow
1. Confirm the target service is monolith-scoped.
2. Identify Solomon module path and canonical dependencies.
3. Implement in layered module structure (`domain -> application -> ports -> adapters -> transport`).
4. Keep ownership boundaries and access modes compliant (`owner_api`, `internal_sql_readonly` only where declared).
5. Apply outbox/idempotency/dedup semantics for mutating and event-driven flows.
6. Add migrations and tests.
7. Run formatting and tests, then report behavior and contract impact.

## Non-Negotiable Rules
- Do not move microservice-scoped services into `solomon`.
- Do not write directly to foreign-owned tables.
- Do not bypass canonical event and dependency contracts.
- Do not place business logic in generic platform utilities.
- Canonical platform implementation path is `solomon/internal/platform/*` only; do not recreate top-level `platform/`.
- Root `solomon/contracts` is for versioned schemas and generated types only.
- Module-private DTOs must live in `contexts/<context>/<service>/transport/...`.
- Do not create centralized microservice client packages under `solomon/integrations`; outbound adapters are module-owned.
- Boundary gates are mandatory: `go run ./scripts/check_boundaries.go` and `golangci-lint run`.
- Use `scripts/newmodule.ps1` or `scripts/newmodule.sh` for scaffolding; do not commit `.gitkeep` scaffolding.

## Completion Checklist
- Monolith boundary is respected.
- Data ownership and DBR access modes are respected.
- Endpoint/event contracts remain compatible and explicit.
- Tests cover behavior and failure paths.
- `gofmt -w .` and `go test ./...` pass.
