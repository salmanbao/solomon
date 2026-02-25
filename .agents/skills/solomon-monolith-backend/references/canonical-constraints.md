# Canonical Constraints

## Source Priority
1. `viralForge/specs/service-architecture-map.yaml`
2. `viralForge/specs/service-data-ownership-map.yaml`
3. `viralForge/specs/dependencies.yaml`
4. `viralForge/specs/DB-01-Data-Contracts.md`
5. `viralForge/specs/DB-02-Shared-Data-Surface.md`
6. `viralForge/specs/00-Canonical-Structure.md`
7. `viralForge/specs/Mxx-*.md` for the target service
8. `solomon/README.md` and `solomon/docs/go-structure-and-data-flow.md`

When conflicts appear, follow this order and treat earlier files as authoritative.

## Runtime Boundary Rules
- Implement only services with `architecture: monolith` inside `solomon`.
- Keep `architecture: microservice` services in `mesh` or external runtimes.
- Never perform direct cross-runtime DB writes.
- For monolith -> microservice reads, use owner API, event projection, or declared replica view.

## Data Ownership Rules
- Use single-writer ownership per canonical table.
- Shared DB in monolith does not relax ownership boundaries.
- Allow direct SQL reads only for declared monolith-to-monolith `internal_sql_readonly` dependencies.
- For cross-context writes, call owner application/service boundary instead of touching owner tables directly.

## Contracts Rules
- Keep endpoint contracts backward-compatible unless spec explicitly allows breaking changes.
- Emit canonical events with canonical names from `viralForge/04-services.md`.
- Use event envelope fields: `event_id`, `event_type`, `occurred_at`, `source_service`, `trace_id`, `schema_version`, `partition_key_path`, `partition_key`, `data`.
- Keep partition key invariant: field at `partition_key_path` equals `partition_key`.

## Reliability Defaults
- Mutating APIs require idempotency key support.
- Event consumers must be idempotent via event dedup storage.
- Use transactional outbox for DB write + domain event emit consistency.
- Apply retries with exponential backoff and bounded attempts.
- Route only domain-class poison events to DLQ.

## Security Defaults
- Enforce authn/authz at adapter boundary.
- Apply RBAC via authorization policies.
- Log access decisions and sensitive operations.
- Keep TLS-in-transit and encrypted-at-rest assumptions in integrations.