---
name: solomon-monolith-migrations
description: Design and implement safe Solomon database migrations for monolith services with single-writer ownership, backward compatibility, and staged rollout practices. Use this skill when schema or data evolution is required.
---

# Solomon Monolith Migrations

Use this skill for schema and data changes in monolith services.

## Load First
- `references/migration-rules.md`
- `references/rollout-patterns.md`

## Optional Supporting References
- `../solomon-monolith-backend/references/canonical-constraints.md`
- `../solomon-monolith-backend/references/monolith-service-catalog.md`

## Workflow
1. Confirm table ownership and affected readers.
2. Design additive migration first.
3. Plan dual-read/dual-write if needed.
4. Implement migration scripts and backfill steps.
5. Validate with integration tests and rollback plan.

## Non-Negotiables
- Do not break existing readers in one step.
- Do not move ownership implicitly via migration.
- Do not perform destructive changes without staged rollout.