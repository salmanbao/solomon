# Service Grouping And Build Strategy

## Solomon Context Map (Monolith Services)
- `campaign-editorial`: M04, M07, M08, M09, M23, M24, M26, M31, M34
- `community-experience`: M46, M47, M48, M49, M53, M60, M61, M62, M88, M92
- `finance-core`: M15
- `identity-access`: M21, M22
- `internal-ops`: M20, M65, M74, M85, M86, M87
- `moderation-safety`: M35, M37

## Recommended Build Sequence (Dependency-First)
1. Identity and admin primitives: M21, M22, M87, M20
2. Campaign/editorial core: M04, M26, M08, M07
3. Discovery and distribution: M09, M23, M24, M31, M34, M53
4. Community/trust layers: M46, M47, M48, M49
5. Commerce experience: M60, M61, M62, M92
6. Ops and support modules: M74, M65, M85, M86, M88
7. Finance orchestration in monolith scope: M15
8. Safety controls: M35, M37

## Extraction-Aware Design
Even while in monolith:
- Keep module APIs and events explicit and versioned.
- Keep dependency direction narrow (no broad shared util coupling).
- Hide storage details behind ports and repositories.
- Treat each module as independently testable and potentially extractable.

## Backlog Slicing Pattern
For each service, split work into slices:
1. Contracts and schema alignment
2. Domain/use-case implementation
3. Adapter implementation (HTTP/DB/events)
4. Integration and contract tests
5. Operational hardening (metrics, retries, alerts)

This slicing keeps delivery incremental while preserving architecture quality.