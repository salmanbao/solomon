# Implementation Playbook

## Inputs
- Service spec: `viralForge/specs/Mxx-*.md`
- Dependency map: `viralForge/specs/dependencies.yaml`
- Ownership map: `viralForge/specs/service-data-ownership-map.yaml`
- Solomon module path: `solomon/contexts/<context>/<service>`

## Delivery Steps
1. Define use-cases from spec requirements.
2. Implement domain invariants and entities.
3. Implement application handlers with explicit ports.
4. Add adapters (HTTP/DB/events/integration).
5. Add migrations in `solomon/migrations` if schema changes.
6. Wire bootstrap and entrypoints if new module pieces are added.
7. Add unit/integration/contract tests.

## Code Rules
- `context.Context` first parameter.
- Constructor injection for dependencies.
- Wrapped errors with `%w`.
- No adapter/framework imports in domain.

## Reliability Rules
- Mutating endpoints must be idempotent.
- DB write + event emit uses transactional outbox.
- Event consumers dedupe by `event_id`.