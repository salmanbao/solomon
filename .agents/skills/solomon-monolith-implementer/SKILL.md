---
name: solomon-monolith-implementer
description: Implement and refactor Solomon monolith services in Go while enforcing canonical service architecture, data ownership, DBR dependencies, contracts, and reliability semantics. Use this skill for feature delivery, bug fixes, and module-level implementation inside `solomon/contexts`.
---

# Solomon Monolith Implementer

Use this skill to build or change monolith services safely and quickly.

## Load First
- `references/implementation-playbook.md`
- `references/implementation-checklist.md`

## Optional Supporting References
- `../solomon-monolith-backend/references/canonical-constraints.md`
- `../solomon-monolith-backend/references/monolith-service-catalog.md`
- `../solomon-monolith-backend/references/go-monolith-implementation.md`

## Workflow
1. Confirm the target service is monolith-scoped.
2. Read service Mxx spec plus dependency and ownership entries.
3. Implement in layered module structure.
4. Keep writes inside owner tables and allowed boundaries.
5. Add tests and verify outbox/idempotency where relevant.
6. Run formatting and tests, then summarize contract impact.

## Non-Negotiables
- Do not pull microservice-only services into `solomon`.
- Do not add cross-module writes to foreign-owned tables.
- Do not bypass canonical endpoint/event contracts.