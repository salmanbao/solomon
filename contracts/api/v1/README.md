# API Contracts v1

Versioned API schemas (OpenAPI/proto/JSON schema) owned by the contracts module.
Only stable, consumer-facing contract artifacts belong here.

## Implemented Module Contracts

- `content-library-marketplace.openapi.json`
  - Covers implemented M09 HTTP routes under `/library/*`.
- `authorization-service.openapi.json`
  - Covers implemented M21 HTTP routes under `/api/authz/v1/*`.
  - Note: M21 spec defines additional endpoints not yet implemented in runtime; they are intentionally excluded from this contract file until delivered.
- `submission-service.openapi.json`
  - Covers implemented M26 HTTP routes under `/submissions*` and `/dashboard/*`.
- `voting-engine.openapi.json`
  - Covers implemented M08 HTTP routes under `/v1/votes*`, `/v1/leaderboards/*`, `/v1/rounds/*`, `/v1/analytics/votes`, and `/v1/quarantine/*`.
- `super-admin-dashboard.openapi.json`
  - Covers implemented M20 HTTP routes under `/api/admin/v1/*`.
