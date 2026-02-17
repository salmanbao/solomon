---
name: "Solomon Monolith Implementation Guide"
description: "Repository-specific instructions for implementing ViralForge monolith services inside Solomon."
category: "Backend Service"
lastUpdated: "2026-02-17"
---

# Solomon Monolith Implementation Guide

## Mission
Implement only the services classified as `architecture: monolith` in `viralForge/specs/service-architecture-map.yaml` inside `solomon`.

This guide defines how to implement modules in Solomon's modular monolith while preserving clean boundaries with mesh microservices.

## Scope and Runtime Boundary

### In scope (`solomon`)
- Monolith services and modules under `solomon/contexts`
- Shared monolith platform code under `solomon/internal` and `solomon/platform`
- App entrypoints under `solomon/cmd` and packaging under `solomon/apps`
- Contracts owned by monolith boundary under `solomon/contracts`

### Out of scope (`solomon`)
- Services classified as `architecture: microservice` (must stay in `mesh`)
- Cross-runtime contract rewrites not requested in canonical specs
- Direct data ownership changes that conflict with `DB-01` / `service-data-ownership-map.yaml`

### Mesh Boundary (Required)
- Monolith and microservices are separate runtimes.
- Use owner API/event projection/declared replica view for cross-boundary reads.
- Never introduce direct cross-runtime DB writes.

## Source of Truth
Always read and follow these before implementation:
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- Service spec: `viralForge/specs/Mxx-*.md`
- `solomon/README.md`

## Solomon Architecture Rules

### Module structure
Each monolith module follows:
- `domain -> application -> ports -> adapters -> contracts`

Path pattern:
- `solomon/contexts/<context>/<service>/domain`
- `solomon/contexts/<context>/<service>/application`
- `solomon/contexts/<context>/<service>/ports`
- `solomon/contexts/<context>/<service>/adapters`
- `solomon/contexts/<context>/<service>/contracts`

Optional:
- `solomon/contexts/<context>/<service>/module.go` for module wiring/registration

### Current bounded contexts
- `campaign-editorial`
- `community-experience`
- `finance-core`
- `identity-access`
- `internal-ops`
- `moderation-safety`

### Entrypoints
- API process: `solomon/cmd/api/main.go`
- Worker process: `solomon/cmd/worker/main.go`
- Composition root: `solomon/internal/app/bootstrap/bootstrap.go`

## Data and Ownership Rules
- Physical model: shared monolith database.
- Logical model: single-writer ownership per service/table.
- No cross-module direct writes to non-owned tables.
- Monolith internal reads may use `internal_sql_readonly` only when declared.
- Keep module data model sections aligned with canonical DB contracts.

## Dependency and Integration Rules
- Respect canonical DBR and EVENT relationships from `dependencies.yaml`.
- Do not introduce hidden dependencies between contexts.
- For async workflows, prefer outbox/event patterns defined in canonical specs.
- Keep cross-context contracts explicit and version-stable.

## Implementation Workflow
1. Confirm target service is `architecture: monolith`.
2. Read its `Mxx` spec, DBR/EVENT dependencies, and ownership model.
3. Implement module layers in the correct context path.
4. Wire module into Solomon bootstrap/entrypoints as needed.
5. Add or update tests in:
   - `solomon/tests/unit`
   - `solomon/tests/integration`
   - `solomon/tests/contract`
   - `solomon/tests/e2e` (when scenario-level validation is needed)
6. Validate contract and data ownership alignment against canonical specs.

## Definition of Done
- Service remains inside Solomon monolith boundary.
- Layering is preserved (`domain` does not depend on adapters).
- Ownership and access modes match canonical DB contracts.
- Dependency behavior matches `dependencies.yaml`.
- Module and tests are consistent with Solomon context conventions.

## Do Not Do
- Do not move microservice-only modules from mesh into Solomon.
- Do not add cross-module direct writes to foreign-owned data.
- Do not bypass canonical event and dependency contracts.
- Do not add business logic to low-level platform utility packages.
