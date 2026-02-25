# Migration Rules

## Ownership
- Change only owner-managed tables unless explicitly approved.
- Keep `single-writer` ownership intact.

## Safe Change Order
1. Add new columns/tables/indexes (non-breaking).
2. Deploy code that writes new shape while reading old+new.
3. Backfill existing data.
4. Switch reads to new shape.
5. Remove old fields in a later release.

## SQL Rules
- Include idempotent guards where practical.
- Add indexes concurrently when required for large tables.
- Keep lock duration minimal.

## Verification
- Test migration up/down where possible.
- Validate query plans for new indexes.
- Validate outbox/event producers continue to work.