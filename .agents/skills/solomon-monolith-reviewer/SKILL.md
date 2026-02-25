---
name: solomon-monolith-reviewer
description: Review Solomon monolith backend changes for bugs, regressions, ownership violations, contract drift, and insufficient tests. Use this skill when asked for code review, spec compliance review, or release risk assessment for monolith services.
---

# Solomon Monolith Reviewer

Use this skill for high-signal review findings focused on correctness and risk.

## Load First
- `references/review-checklist.md`
- `references/high-risk-patterns.md`

## Optional Supporting References
- `../solomon-monolith-backend/references/canonical-constraints.md`
- `../solomon-monolith-backend/references/quality-gates-and-testing.md`
- `../solomon-monolith-backend/references/monolith-service-catalog.md`

## Review Workflow
1. Map changed files to module and service ownership.
2. Check behavior against service spec and canonical maps.
3. Prioritize findings by severity.
4. Verify test coverage for changed behavior.
5. Report concrete findings with file and line references.

## Output Rules
- Findings first, sorted by severity.
- Mention missing tests or residual risks explicitly.
- Keep summary short after findings.