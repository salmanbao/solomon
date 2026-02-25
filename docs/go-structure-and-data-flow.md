# Solomon Go Structure And Data Flow

## Runtime Layout

- `cmd/api`: API process entrypoint
- `cmd/worker`: worker/outbox process entrypoint
- `internal/app/bootstrap`: composition root
- `internal/platform/*`: canonical concrete platform implementations
- `internal/shared/*`: shared technical helpers only
- `contexts/*`: bounded-context service modules
- `contracts/`: separate contracts Go module (`solomon/contracts`)
- `deploy/`: non-Go deployment assets

Top-level `platform/` is intentionally removed. `internal/platform/` is the only platform layer.

## Module Layout

Each module in `contexts/<context>/<service>` follows:

- `domain/`
- `application/`
- `ports/`
- `adapters/`
- `transport/` (module-private transport DTOs/mappers)

`transport/` replaced module-local `contracts/` to avoid confusion with stable cross-runtime contracts.

## Contracts Governance

Stable cross-runtime contracts live in the separate `contracts` module:

- `contracts/api/v{n}`
- `contracts/events/v{n}`
- `contracts/schemas/v{n}`
- `contracts/gen/...` generated types

Do not place module-private HTTP/event DTOs in root `contracts/`.

## Data Flow

1. Adapter accepts request/event.
2. Adapter maps transport DTO to application command/query.
3. Application executes domain logic through ports.
4. Owner-table writes occur through module-owned adapters.
5. Outbox row is persisted in the same transaction as state mutation.
6. Worker relays outbox events with retry and idempotency safeguards.

## Enforced Boundary Rules

- `domain` allowlist: stdlib + same-module `domain` imports only.
- `application` allowlist: stdlib + same-module `application`, `domain`, `ports`, and `solomon/contracts`.
- Cross-module imports (`solomon/contexts/<other-module>/...`) are forbidden.
- `domain` and `application` must not import `adapters`, `internal/*`, or `integrations/*`.

Enforcement runs via:

- `.golangci.yml` (`depguard`)
- `scripts/check_boundaries.go` (strict module/layer checks)

## Negative Example (Forbidden)

```go
// contexts/community-experience/discover-service/application/usecase.go
import "solomon/contexts/identity-access/authorization-service/adapters/http" // forbidden
```

Reason: application layer importing another module's adapter violates both layer and module boundaries.
