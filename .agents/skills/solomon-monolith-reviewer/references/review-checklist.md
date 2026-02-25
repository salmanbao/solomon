# Review Checklist

## Correctness
- Domain invariants preserved.
- State transitions valid.
- Error handling and retries correct.
- Idempotency behavior preserved.

## Architecture
- Layer boundaries respected.
- No hidden cross-context coupling.
- No direct writes to foreign-owned tables.

## Contracts
- Endpoint compatibility preserved.
- Event names and envelope fields canonical.
- Dependency declarations still accurate.

## Data
- Migrations are additive/safe or phased.
- Query/read path uses allowed access mode.

## Tests
- Unit tests for business logic.
- Integration tests for persistence/outbox.
- Contract tests for endpoint/event changes.