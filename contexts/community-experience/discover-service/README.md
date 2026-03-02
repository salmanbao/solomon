# M53-Discover-Service

Discover aggregation surface for monolith consumers.

## Canonical Dependency Alignment
- DBR provider: `M23-Campaign-Discovery-Service`.
- No direct writes to provider-owned tables.

## API Surface (current)
- `GET /api/v1/discover/feed`
- `GET /api/v1/discover?tab=all|campaigns&cursor=...&limit=...`

## Compatibility
- `/api/v1/discover` remains backward-compatible for product-discovery callers
  when M53 query parameters are not provided.
