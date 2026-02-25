# Structured Logging Playbook

## Goal
Ensure every modified or newly added module file has appropriate structured logging at operational boundaries and error paths.

## Change Detection
Start by listing working tree changes:

```powershell
git status --short
```

Focus on:
1. Modified `.go` files.
2. Newly added `.go` files.
3. Runtime paths: `cmd/`, `internal/`, `contexts/`.

## Placement Rules
Add logs in:
1. HTTP/server entrypoints and request handling adapters.
2. Worker consumers, schedulers, and event handlers.
3. Application use-case orchestration where side effects and failures happen.
4. Infrastructure adapters (db, messaging, external calls) at failure boundaries.

Avoid logs in:
1. Domain entities/value objects.
2. Pure deterministic helpers with no side effects.
3. Tight loops where repeated logs create noise without actionable value.

## Field Standards
Use stable keys:
1. `event`: short operation name (`"assign_role_started"`).
2. `module`: `<context>/<service>` when possible.
3. `layer`: `transport`, `adapter`, `application`, `worker`, `platform`.
4. `request_id` or `correlation_id` when available.
5. Entity IDs only (`user_id`, `campaign_id`), never secret values.
6. `error` field on failures.

Prefer:

```go
logger.Info("assign role started",
  "event", "assign_role_started",
  "module", "identity-access/authorization-service",
  "layer", "application",
  "user_id", cmd.UserID,
  "role_id", cmd.RoleID,
)
```

Avoid:

```go
log.Printf("assign role start user=%s role=%s", cmd.UserID, cmd.RoleID)
```

## Consistency Rules
1. Keep the same key names across modules.
2. Keep level semantics consistent:
   - `Debug`: diagnostic details
   - `Info`: lifecycle milestones
   - `Warn`: recoverable anomalies
   - `Error`: failed operation
3. Keep message text concise; put details into structured fields.

## Security and Privacy
Never log:
1. Access tokens, secrets, API keys.
2. Passwords or authentication challenge material.
3. Raw payload bodies from auth, payment, or personal profile flows.
4. Full SQL queries containing user data unless sanitized.

## Minimum Per-Change Logging Checklist
For each changed/new runtime file:
1. At least one start/success boundary log for major operation paths.
2. Error logs for all returned errors with context fields.
3. Structured fields use shared key naming conventions.
4. No sensitive data emitted.

## Validation
1. Run `go test ./...`.
2. Run any project lint step if available.
3. Verify logs compile with current logger interface in that package.
4. Ensure new logs do not break module boundaries.
