---
name: solomon-monolith-extraction-readiness
description: Refactor and design Solomon monolith modules to stay extraction-ready with explicit contracts, minimized coupling, and clean bounded context seams. Use this skill for boundary hardening and future microservice extraction preparation.
---

# Solomon Extraction Readiness

Use this skill when preparing monolith modules for future extraction.

## Load First
- `references/extraction-signals.md`
- `references/decoupling-playbook.md`

## Optional Supporting References
- `../solomon-monolith-backend/references/monolith-service-catalog.md`
- `../solomon-monolith-backend/references/service-grouping-and-build-strategy.md`

## Workflow
1. Identify coupling and boundary leaks.
2. Introduce explicit ports/contracts at seams.
3. Isolate module-owned writes and state transitions.
4. Replace implicit data coupling with explicit APIs/events.
5. Add characterization and contract tests.

## Non-Negotiables
- Domain logic stays framework-free.
- Cross-context access is explicit and auditable.
- Extraction does not require data ownership rewrites at cutover time.