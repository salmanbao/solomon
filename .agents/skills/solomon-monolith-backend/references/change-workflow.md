# Change Workflow

## 1. Intake
- Identify target service/module ID from request.
- Confirm it is monolith in `service-architecture-map.yaml`.
- Find module path in `solomon/contexts`.

## 2. Contract Read
- Read target `Mxx-*.md`.
- Read dependency rows in `dependencies.yaml`.
- Read ownership row in `service-data-ownership-map.yaml`.
- Read `DB-01` and `DB-02` sections for owned/shared entities.

## 3. Plan The Change
- Define impacted use cases and invariants.
- List tables written vs read-only dependencies.
- List endpoint/event contract changes.
- Decide if migration is required.

## 4. Implement
- Update domain/application logic first.
- Update ports and adapters.
- Add migration files if schema changes.
- Wire module in bootstrap if new components are introduced.

## 5. Verify
- Run unit/integration/contract tests for touched modules.
- Run `gofmt -w .` and `go test ./...`.
- Run `go run ./scripts/check_boundaries.go`.
- Run `golangci-lint run`.
- Verify event payload and idempotency behavior where applicable.

## 6. Report
- Summarize changed files and behavior deltas.
- Explicitly call out ownership/dependency/contract impacts.
- Flag any unresolved assumptions or follow-up work.
