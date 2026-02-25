# High Risk Patterns

## Critical
- Cross-module writes into non-owned tables.
- Event schema or name drift from canonical contracts.
- Missing idempotency for mutating operations.

## High
- Direct adapter/framework leakage into domain.
- Non-transactional DB write + event publish sequence.
- Breaking API changes without versioning strategy.

## Medium
- Missing boundary-case tests.
- Weak timeout/retry policy on remote calls.
- New dependency not reflected in canonical maps.