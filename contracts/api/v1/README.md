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
  - Covers implemented M20 ownership slice under `/api/admin/v1/impersonation/*`, `/api/admin/v1/users/*`, `/api/admin/v1/campaigns/*`, `/api/admin/v1/submissions/*`, `/api/admin/v1/feature-flags*`, `/api/admin/v1/analytics/*`, and `/api/admin/v1/audit-logs*`.
- `admin-dashboard-service.openapi.json`
  - Covers implemented M86 control-plane endpoints:
    - `/api/admin/v1/actions/log`
    - `/api/admin/v1/identity/roles/grant`
    - `/api/admin/v1/moderation/decisions`
    - `/api/admin/v1/abuse-prevention/lockouts/{user_id}/release`
    - `/api/admin/v1/finance/refunds`
    - `/api/admin/v1/finance/billing/invoices/{invoice_id}/refund`
    - `/api/admin/v1/finance/rewards/recalculate`
    - `/api/admin/v1/finance/affiliates/{affiliate_id}/suspend`
    - `/api/admin/v1/finance/affiliates/{affiliate_id}/attributions`
    - `/api/admin/v1/finance/payouts/{payout_id}/retry`
    - `/api/admin/v1/compliance/disputes/{dispute_id}/resolve`
    - `/api/admin/v1/compliance/disputes/{dispute_id}/reopen`
    - `/api/admin/v1/compliance/consent/{user_id}`
    - `/api/admin/v1/compliance/consent/{user_id}/update`
    - `/api/admin/v1/compliance/consent/{user_id}/withdraw`
    - `/api/admin/v1/compliance/exports`
    - `/api/admin/v1/compliance/exports/{request_id}`
    - `/api/admin/v1/compliance/deletion-requests`
    - `/api/admin/v1/compliance/retention/legal-holds`
    - `/api/admin/v1/compliance/legal-holds/check`
    - `/api/admin/v1/compliance/legal-holds/{hold_id}/release`
    - `/api/admin/v1/compliance/legal/compliance-scans`
    - `/api/admin/v1/support/tickets/{ticket_id}`
    - `/api/admin/v1/support/tickets/search`
    - `/api/admin/v1/support/tickets/{ticket_id}/assign`
    - `/api/admin/v1/creator-workflow/editor/campaigns/{campaign_id}/save`
    - `/api/admin/v1/creator-workflow/clipping/projects/{project_id}/export`
    - `/api/admin/v1/creator-workflow/auto-clipping/models/deploy`
    - `/api/admin/v1/integrations/keys/rotate`
    - `/api/admin/v1/integrations/workflows/test`
    - `/api/admin/v1/integrations/webhooks/{webhook_id}/replay`
    - `/api/admin/v1/integrations/webhooks/{webhook_id}/disable`
    - `/api/admin/v1/integrations/webhooks/{webhook_id}/deliveries`
    - `/api/admin/v1/integrations/webhooks/{webhook_id}/analytics`
    - `/api/admin/v1/platform-ops/migrations/plans`
    - `/api/admin/v1/platform-ops/migrations/runs`
- `abuse-prevention-service.openapi.json`
  - Covers implemented M37 owner endpoints under `/api/v1/auth/*` and `/api/v1/admin/abuse-threats*`.
- `moderation-service.openapi.json`
  - Covers implemented M35 HTTP routes under `/api/moderation/*`.
- `product-service.openapi.json`
  - Covers implemented M60 HTTP routes under `/api/v1/products*`, discovery/search, and user data endpoints.
- `chat-service.openapi.json`
  - Covers implemented M46 HTTP routes under `/api/v1/chat/*`.
