---
name: solomon-gorm-postgres-enforcer
description: Enforce GORM-only PostgreSQL data access in Solomon and remove direct SQL query usage that increases SQL-injection risk. Use when changing files under `contexts/*/*/adapters/postgres`, `internal/platform/db`, repositories, transactions, query filters, or any Go code that touches PostgreSQL reads/writes.
---

# Solomon GORM Postgres Enforcer

Use this skill to harden PostgreSQL access against injection-prone query patterns.

## Load First
- `references/gorm-postgres-policy.md`
- `references/gorm-remediation-checklist.md`

## Workflow
1. Identify PostgreSQL-touching files (`adapters/postgres`, repositories, DB wiring).
2. Run `scripts/check_gorm_postgres.py` from repository root.
3. Replace direct SQL usage with GORM query builder patterns.
4. Keep all dynamic values parameterized (`?` placeholders), never string-concatenated.
5. Use `db.WithContext(ctx)` and `db.Transaction(...)` for request-scoped DB access.
6. Re-run checker and `go test ./...` before finalizing.

## Enforcement Rules
- Do not import `database/sql` in module PostgreSQL adapters (except low-level platform bootstrap code that opens connections).
- Do not use direct SQL execution APIs in app/module code:
  - `Query`, `QueryContext`, `QueryRow`, `QueryRowContext`
  - `Exec`, `ExecContext`
  - `Prepare`, `PrepareContext`
- Do not use `fmt.Sprintf`/string concatenation to build SQL fragments.
- Prefer GORM methods (`Where`, `First`, `Find`, `Create`, `Updates`, `Delete`, `Clauses`, `Model`).
- Keep model mapping explicit and avoid exposing GORM models to domain entities directly.

## Allowed Exceptions
- Repository-wide SQL migration files in `migrations/*.sql`.
- One-off exceptions only when explicitly approved; annotate the line with:
  - `// gorm-postgres-enforcer: allow-raw-sql <reason>`

## Validation Commands
```powershell
python .agents/skills/solomon-gorm-postgres-enforcer/scripts/check_gorm_postgres.py .
go test ./...
```

## Non-Negotiables
- Never bypass parameter binding for user-influenced data.
- Keep domain/application logic independent of ORM-specific concerns.
- If replacing SQL changes behavior, add or update tests to preserve existing semantics.
