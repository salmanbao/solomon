---
name: solomon-monolith-contracts
description: Define and evolve Solomon monolith API and event contracts while preserving backward compatibility, canonical event envelope requirements, and dependency map correctness. Use this skill for endpoint changes, event payload changes, and contract governance.
---

# Solomon Monolith Contracts

Use this skill for API and event contract work in monolith modules.

## Load First
- `references/contract-rules.md`
- `references/compatibility-checklist.md`

## Optional Supporting References
- `../solomon-monolith-backend/references/canonical-constraints.md`
- `../solomon-monolith-backend/references/monolith-service-catalog.md`

## Workflow
1. Identify the contract delta (API/event/config).
2. Check canonical constraints and dependency implications.
3. Place stable contracts in `solomon/contracts` and version or extend without breaking consumers.
4. Update producers/consumers and tests.
5. Document compatibility notes in PR summary.

## Non-Negotiables
- Canonical event names and envelope fields remain valid.
- Partition key path/value invariant remains valid.
- Breaking changes require explicit migration/versioning plan.
- `solomon/contracts` must stay runtime-agnostic (schemas + generated types only).
- Do not place module-private transport DTOs in root contracts; use module `transport/`.
- Consumers in other runtimes must import `solomon/contracts`, not Solomon runtime packages.
