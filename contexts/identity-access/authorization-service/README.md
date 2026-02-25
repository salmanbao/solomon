# Authorization Service

M21 module for RBAC authorization flows.

## Implemented baseline
- Permission check and batch check use-cases with deny-by-default behavior.
- Role listing, grant, and revoke command flows.
- Delegation creation with expiry validation.
- Idempotency handling for mutating commands.
- Transactional outbox persistence contracts and relay worker primitive.
- GORM PostgreSQL adapter and deterministic in-memory adapter.

## HTTP routes
- `POST /api/authz/v1/check`
- `POST /api/authz/v1/check-batch`
- `GET /api/authz/v1/users/{user_id}/roles`
- `POST /api/authz/v1/users/{user_id}/roles/grant`
- `POST /api/authz/v1/users/{user_id}/roles/revoke`
- `POST /api/authz/v1/delegations`
