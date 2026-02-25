# Documentation Standards (Solomon)

## Audience
Write for backend developers working in Solomon contexts.

## Required Sections For Module Docs
1. Module responsibility and context boundary.
2. Inbound adapters and exposed contracts.
3. Use-case flow and domain invariants.
4. Owned data and allowed read dependencies.
5. Event and outbox behavior.
6. Failure handling and idempotency approach.
7. Testing coverage map.
8. Decision rationale and tradeoffs.

## Writing Rules
- Use module paths and concrete code references.
- Separate facts from assumptions.
- Document constraints that prevent unsafe coupling.
- Keep docs concise but explicit enough for safe changes.

## Canonical References
- `viralForge/specs/service-architecture-map.yaml`
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- `viralForge/specs/DB-01-Data-Contracts.md`
- `viralForge/specs/DB-02-Shared-Data-Surface.md`
- `solomon/docs/go-structure-and-data-flow.md`