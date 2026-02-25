# Voting Engine (M08)

## Responsibility and Context Boundary
`contexts/campaign-editorial/voting-engine` implements MVP voting behavior for submissions:
- vote create/update/retract with idempotency and one-vote-per-identity constraints
- campaign, round, creator, and trending leaderboard reads
- round results and vote analytics reads
- quarantine moderation action handling
- outbox relay and event consumers for declared dependencies

M08 is monolith-scoped in `viralForge/specs/service-architecture-map.yaml`. Canonical ownership remains:
- **M08 owned canonical tables:** none (`DB-01`)
- runtime uses legacy successor surface (`votes`, `voting_rounds`, `vote_quarantine`) plus reliability tables for outbox/dedup/idempotency

## Inbound Adapters and Contracts
- HTTP adapter: `adapters/http/handler.go`
- HTTP DTOs: `transport/http/http_dto.go`
- Routes wired in `internal/platform/httpserver/server.go`:
  - `POST /v1/votes`
  - `DELETE /v1/votes/{vote_id}`
  - `GET /v1/votes/submissions/{submission_id}`
  - `GET /v1/votes/leaderboard` (legacy compatibility)
  - `GET /v1/leaderboards/campaign/{campaign_id}`
  - `GET /v1/leaderboards/round/{round_id}`
  - `GET /v1/leaderboards/trending`
  - `GET /v1/leaderboards/creator/{user_id}`
  - `GET /v1/rounds/{round_id}/results`
  - `GET /v1/analytics/votes`
  - `POST /v1/quarantine/{quarantine_id}/action`
- API contract artifact: `contracts/api/v1/voting-engine.openapi.json`

## Use-Case Flow and Domain Invariants
- `CreateVote`:
  - requires `Idempotency-Key`
  - validates submission/campaign eligibility via read-only DBR projections
  - blocks self-voting
  - enforces per-identity uniqueness:
    - with round: `(user_id, submission_id, round_id)`
    - without round: `(user_id, submission_id)`
  - snapshots reputation score and applies weight tier (`1.0x`, `1.5x`, `2.0x`, `3.0x`)
  - emits `vote.created` or `vote.updated` via outbox
- `RetractVote`:
  - requires `Idempotency-Key`
  - soft-retracts vote (`retracted = true`)
  - emits `vote.retracted` via outbox
- `ApplyQuarantineAction`:
  - requires `Idempotency-Key`
  - supports `approve` / `reject` only
  - updates quarantine status and vote active/retracted state
  - emits `vote.updated` or `vote.retracted`
- Leaderboard queries:
  - sort by weighted score descending
  - tie-break by oldest first vote (submission age proxy in vote stream)
  - trending applies formula from M08 spec with decay

## Owned Data and Read Dependencies
M08 read dependencies (from `dependencies.yaml` / `DB-02`):
- `DBR:M04-Campaign-Service` via `internal_sql_readonly` (`campaigns`)
- `DBR:M26-Submission-Service` via `internal_sql_readonly` (`submissions`)
- `DBR:M48-Reputation-Service` via `internal_sql_readonly` (`user_reputation_scores`; fallback weight if unavailable)
- `DBR:M01-Authentication-Service` via `owner_api` (identity context supplied by API layer)

Writes stay inside voting surface tables only.

## Event and Outbox Behavior
- Emitted events (outbox-backed):
  - `vote.created`
  - `vote.updated`
  - `vote.retracted`
  - `voting_round.closed`
- Consumed events:
  - `submission.approved` (dedup + acknowledge path)
  - `submission.rejected` (dedup + bulk vote retraction)
  - `campaign.paused` (transition active rounds to `closing_soon`)
  - `campaign.completed` (close rounds and emit `voting_round.closed`)
- Worker components:
  - `application/workers/outbox_relay.go`
  - `application/workers/submission_lifecycle_consumer.go`
  - `application/workers/campaign_state_consumer.go`
- Bootstrap wiring:
  - API: postgres-backed module in `internal/app/bootstrap/bootstrap.go`
  - Worker: consumers + outbox relay in `internal/app/bootstrap/bootstrap.go`

## Failure Handling and Idempotency
- API idempotency persisted in `voting_engine_idempotency` with request hash validation.
- Event consumer dedupe persisted in `voting_event_dedup` (`event_id` + payload hash).
- Outbox publish retries happen in worker polling loop; pending rows remain until publish + ack.
- Reputation lookup failure degrades to default weight `1.0x`.

## Testing Coverage Map
- `tests/unit/voting_engine_test.go`
  - create replay, round-aware voting, retract flow
- `tests/unit/voting_engine_workers_test.go`
  - submission/campaign consumer side effects and outbox emissions
- `tests/unit/voting_engine_contracts_test.go`
  - API path/method contract coverage
  - event schema set coverage
  - emitted envelope consistency checks

## Decision Rationale
### Decision
Implement M08 as postgres-backed module with explicit outbox + dedup workers while preserving memory adapter parity for unit tests.

### Context
M08 is in MVP milestone M2 and depends on M04/M26 read-only surfaces; it also has cross-service event contracts requiring reliable delivery semantics.

### Alternatives Considered
- Keep M08 in-memory only: rejected because it cannot satisfy outbox/event consumer reliability requirements.
- Add direct cross-module writes into submission/campaign tables: rejected due canonical ownership boundaries.

### Tradeoffs
- Improves contract fidelity and operational reliability.
- Adds more module plumbing (outbox/dedup tables and worker wiring).

### Consequences
- M08 now supports declared event contracts and idempotent mutation flows end-to-end.
- Extraction readiness is preserved because integration points remain in ports and adapter seams.

### Evidence
- Spec: `viralForge/specs/M08-Voting-Engine.md`
- Dependency map: `viralForge/specs/dependencies.yaml`
- Ownership map: `viralForge/specs/service-data-ownership-map.yaml`
- Implementation: `contexts/campaign-editorial/voting-engine/*`
- Wiring: `internal/app/bootstrap/bootstrap.go`
