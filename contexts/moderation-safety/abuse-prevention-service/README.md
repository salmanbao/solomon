# M37-Abuse-Prevention-Service

Configuration declaration: no runtime config; inherits platform defaults.

Monolith abuse-prevention surface routed from `internal/platform/httpserver`.

## Canonical Dependency Alignment
- DBR provider: `M12-Fraud-Detection-Engine` via owner API projection.
- No direct cross-service DB reads/writes.

## API Surface (current)
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/challenge/{id}`
- `GET /api/v1/admin/abuse-threats`
- `POST /api/v1/admin/abuse-threats/{user_id}/lockout/release`

## Contract Notes
- Canonical error envelope is returned for all failures.
- Challenge mutation enforces `Idempotency-Key`.
- Admin threat endpoint requires bearer auth plus `X-Admin-Id`.
- Admin lockout release is idempotent, role-restricted, and emits owner-side abuse audit records.

## Production Reliability Boundary
- API bootstrap wiring now uses persistent owner-side adapters for lockout state (`abuse_lockout_history`), idempotency (`abuse_idempotency`), and audit logs (`abuse_audit_log`).
- `NewInMemoryModule` remains available for isolated tests.
