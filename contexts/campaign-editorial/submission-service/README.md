# Submission Service (M26)

## Responsibility and Boundary
`contexts/campaign-editorial/submission-service` owns submission intake and review workflow:
- create submission
- approve/reject transitions
- reporting and flagged state progression
- dashboard and analytics query surfaces

Boundary rules:
- owner-write on M26 tables only (`submissions`, `submissions_audit`, `submission_flags`, `submission_reports`, `bulk_submission_operations`, `view_snapshots`)
- no direct writes to M04/M06 tables

## Inbound Adapters and Contracts
- Transport adapter: `adapters/http/handler.go`
- DTOs: `transport/http/http_dto.go`
- API routing: `/submissions*`, `/dashboard/*` in `internal/platform/httpserver/server.go`
- API contract artifact: `contracts/api/v1/submission-service.openapi.json`

## Use-Case Flow and Invariants
- `CreateSubmissionUseCase`: validates required campaign/platform/url fields and uniqueness guard.
- `ReviewSubmissionUseCase`: allows approval/rejection from `pending` or `flagged` only.
- `ReportSubmissionUseCase`: appends report and increments report counter; pending submissions become flagged.
- Query use case provides creator and brand dashboard summaries.

Core invariants:
- duplicate active submission (same creator + campaign + post URL) blocked
- review operations require actor identity
- status transitions are explicit and validated

## Owned Data and Read Dependencies
Migration:
- `migrations/20260225_0004_m26_submission_service.sql`

Canonical references:
- `viralForge/specs/service-data-ownership-map.yaml` for M26 owned tables
- `viralForge/specs/dependencies.yaml` for declared DBR/EVENT edges

## Event and Outbox Behavior
Canonical emitted events for M26 remain:
- `submission.created`, `submission.approved`, `submission.rejected`, `submission.flagged`, `submission.view_locked`, `submission.cancelled`

This delivery focuses on module behavior and contracts; transactional outbox relay wiring is a next reliability slice.

## Failure Handling and Idempotency
- Domain errors are mapped to transport errors in `internal/platform/httpserver/server.go`
- Duplicate submission and invalid transitions return conflict-level responses
- Validation failures return bad-request semantics

## Testing Coverage Map
Unit tests:
- `tests/unit/submission_service_test.go`
  - create + approve flow
  - duplicate guard behavior

## Decision Rationale
### Decision
Implement M26 as a pure layered module with deterministic in-memory adapter first.

### Context
M26 is dependency-critical for M08 voting and reward lifecycle while M04/M06 integration remains evolving.

### Alternatives Considered
- Full event/outbox worker in initial slice: rejected to avoid delaying multi-service MVP enablement.
- Flat handler/repository design: rejected due boundary and extraction-readiness risks.

### Tradeoffs
- Faster functional coverage and testability.
- Event relays and external integration stubs require follow-up.

### Consequences
- Core submission lifecycle behavior is ready and contract-stable.
- Operational hardening can be added without changing handler signatures.

### Evidence
- Code: `contexts/campaign-editorial/submission-service/*`
- Routing: `internal/platform/httpserver/server.go`
- Migration: `migrations/20260225_0004_m26_submission_service.sql`

