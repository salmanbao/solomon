# Extraction Signals

## Signals A Module Is Not Extraction-Ready
- Cross-context table writes.
- Broad shared package imports across contexts.
- Business logic spread across adapters/platform helpers.
- Undocumented DBR/event dependencies.
- Contracts tightly bound to internal ORM models.

## Signals A Module Is Extraction-Ready
- Clear module API and event contracts.
- Ports isolate outbound dependencies.
- Owned data model is explicit and bounded.
- Tests run module behavior independently.
- Operational concerns (timeouts/retries/idempotency) are explicit.