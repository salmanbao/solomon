---
name: solomon-dev-documentation
description: Write and update developer-facing documentation for Solomon monolith modules so engineers can understand implementation details and reason about design decisions and tradeoffs. Use this skill when documenting module behavior, context boundaries, and rationale for architectural choices in solomon.
---

# Solomon Dev Documentation

Use this skill to document Solomon modules and decision rationale for maintainability and future extraction readiness.

## Load First
- `references/documentation-standards.md`
- `references/documentation-workflow.md`
- `references/decision-rationale-template.md`

## Focus Areas
- Module docs in `solomon/contexts/*/*/README.md`.
- Architecture docs in `solomon/docs/`.
- Cross-context dependency and ownership reasoning.
- Decisions that affect modular boundaries and extraction readiness.

## Workflow
1. Read module code across domain/application/ports/adapters/contracts.
2. Record actual behavior, invariants, and boundaries.
3. Explain design decisions and rejected alternatives.
4. Document ownership and dependency implications.
5. Capture testing and operational consequences.
6. Verify wording aligns with canonical specs.

## Non-Negotiables
- Keep docs aligned to current behavior.
- Explain both technical reasoning and tradeoffs.
- Preserve canonical ownership/dependency terminology.
- Avoid undocumented assumptions in critical paths.