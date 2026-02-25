# Contract Rules

## Module Boundary
- Stable cross-runtime contracts live in `solomon/contracts`.
- Keep contracts versioned under `api/v{n}`, `events/v{n}`, `schemas/v{n}`.
- Generated types under `gen/...` must be derived from versioned schemas.
- Do not place runtime adapters or module-private DTOs in root contracts.

## API Contracts
- Keep existing fields and semantics stable.
- Additive fields are preferred over replacement.
- Validate inputs at adapter boundary.

## Event Contracts
- Keep canonical event names.
- Envelope fields are mandatory.
- Include schema version and stable partition key.

## Dependency Contracts
- If a new DBR or event dependency appears, update maps/specs.
- Avoid hidden dependencies in implementation.

## Testing
- Add contract tests for endpoint shape.
- Add contract tests for event payload and envelope.
