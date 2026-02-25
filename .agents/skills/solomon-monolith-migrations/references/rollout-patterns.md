# Rollout Patterns

## Additive Schema Rollout
- Release A: additive schema only.
- Release B: code writes both old/new.
- Release C: code reads new.
- Release D: cleanup old schema.

## Backfill Rollout
- Chunked processing by primary key/time window.
- Track checkpoint progress.
- Make backfill idempotent and restart-safe.

## Rollback Strategy
- Keep old read path until confidence is established.
- Feature-flag new behavior when risk is high.
- Prepare restore scripts for irreversible mistakes.