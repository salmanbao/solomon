# Quality Gates And Testing

## Test Layers
- Unit: `solomon/tests/unit`
- Integration: `solomon/tests/integration`
- Contract: `solomon/tests/contract`
- E2E: `solomon/tests/e2e`

## Unit Test Expectations
- Table-driven tests for use cases and domain invariants.
- Cover success path, validation failures, boundary values, and idempotency behavior.
- Keep tests deterministic and independent from wall-clock/network.

## Integration Test Expectations
- Validate repository behavior against real schema/migrations.
- Validate transactional outbox write+publish workflow.
- Validate integration adapters (microservice clients, event bus) with controlled fakes/stubs.

## Contract Test Expectations
- Verify endpoint request/response shape stability.
- Verify emitted event envelope, event names, and schema version.
- Verify partition key path/value invariant.

## E2E Test Expectations
- Run campaign/editor/distribution/finance critical user flows for affected modules.
- Include failure scenarios: upstream timeout, duplicate event, retry exhaustion.

## Command Gates
Run from `solomon` root:
```bash
gofmt -w .
go run ./scripts/check_boundaries.go
golangci-lint run
go test ./...
```
Optional stronger gates:
```bash
go test ./... -race
go test ./... -cover
```

## Spec Consistency Gates
When touching event/contract heavy modules, also verify related docs/maps remain aligned:
- `viralForge/specs/dependencies.yaml`
- `viralForge/specs/service-data-ownership-map.yaml`
- service `Mxx` spec
- canonical event registry in `viralForge/04-services.md`

## Release Readiness Checklist
- Module still respects monolith/microservice boundary.
- Table ownership and read access modes remain compliant.
- No undeclared DBR or event dependencies added.
- Backward compatibility maintained for existing APIs/events.
- Migrations reversible or safely staged.
- Observability signals (logs/metrics/traces) included for new flows.
