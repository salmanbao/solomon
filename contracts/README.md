# Contracts Module

`contracts/` is a separate Go module (`solomon/contracts`) for cross-runtime contracts.

## What Lives Here

- Versioned API schemas: `api/v{n}/`
- Versioned event schemas: `events/v{n}/`
- Versioned shared schemas: `schemas/v{n}/`
- Generated Go contract types only: `gen/...`

## What Must Not Live Here

- Module-local transport DTOs used only inside a monolith module
- Runtime adapters, repositories, or business logic
- Monolith wiring code

Module-local DTOs belong in each service module at:
`contexts/<context>/<service>/transport/...`

## Versioning Rules

- Additive-only changes inside an existing major version.
- Breaking changes require a new major version directory (for example `v2`).
- Generated types in `gen/...` must map to versioned schemas and remain backward compatible.
