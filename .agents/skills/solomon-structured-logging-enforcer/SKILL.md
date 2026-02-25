---
name: solomon-structured-logging-enforcer
description: Add and maintain structured logging in Solomon modules whenever files are modified or new files are introduced. Use this skill during feature work, refactors, or bug fixes that touch handlers, adapters, application use cases, workers, or platform wiring so logs remain consistent, queryable, and boundary-safe.
---

# Solomon Structured Logging Enforcer

Use this skill to apply structured logging as part of every code change, not as a follow-up task.

## Load First
- `references/structured-logging-playbook.md`

## Workflow
1. Collect changed and newly added files.
2. Filter to runtime Go code paths (`cmd`, `internal`, `contexts`).
3. Add structured logs at request/job boundaries and error branches.
4. Keep logs in adapters/application layers, never domain entities/value objects.
5. Ensure stable field names and avoid sensitive payloads.
6. Run tests and include logging changes in the same commit.

## Repo Anchors
- `cmd/api/main.go`
- `cmd/worker/main.go`
- `internal/platform/observability/tracing.go`
- `internal/platform/httpserver/server.go`
- `contexts/*/*/adapters/**/*.go`
- `contexts/*/*/application/**/*.go`

## Non-Negotiables
- Do not add logs in domain models or low-level utility helpers that should remain pure.
- Do not log secrets, tokens, raw PII, or full payload bodies.
- Do not use inconsistent keys for the same concept across modules.
- Prefer structured key-value logging over unstructured message concatenation.
