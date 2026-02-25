# Compatibility Checklist

## API
- Old clients can still send requests successfully.
- Old clients can still parse responses.
- New required fields are not introduced abruptly.

## Events
- Existing consumers can parse new payloads.
- Removed/renamed fields are versioned safely.
- Event ordering/partition keys unchanged unless intentional.

## Docs And Maps
- Service `Mxx` spec updated if contract semantics changed.
- `dependencies.yaml` updated when dependency graph changes.
- Ownership maps updated if table contracts changed.
- Cross-runtime consumers import `solomon/contracts` only (no Solomon runtime package coupling).
