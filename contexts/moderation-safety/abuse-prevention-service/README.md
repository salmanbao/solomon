# M37-Abuse-Prevention-Service

Monolith abuse-prevention surface routed from `internal/platform/httpserver`.

## Canonical Dependency Alignment
- DBR provider: `M12-Fraud-Detection-Engine` via owner API projection.
- No direct cross-service DB reads/writes.

## API Surface (current)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/challenge/{id}`
- `GET /api/v1/admin/abuse-threats`

## Contract Notes
- Canonical error envelope is returned for all failures.
- Challenge mutation enforces `Idempotency-Key`.
- Admin threat endpoint requires bearer auth plus `X-Admin-Id`.
