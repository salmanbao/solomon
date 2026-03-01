# ViralForge Full Module Dependency Roadmap (Monolith + Microservice)

## Summary
- Use active service set only: 79 modules total (`49 microservices`, `30 monoliths`).
- Keep deprecated IDs as alias-cleanup scope only (16 deprecated IDs mapped to active successors).
- Canonical relation graph basis:
  - 103 DBR edges
  - 45 event-provider edges
  - 148 total active dependency edges
  - 2 strongly-coupled cycles requiring contract-first implementation groups
- Execution model: dual-track delivery (Monolith track in Solomon + Microservice track in Mesh), synchronized by dependency phase gates.

## Canonical Inputs
- [service-architecture-map.yaml](D:/whop-spec-docs/mesh/viralForge/specs/service-architecture-map.yaml)
- [dependencies.yaml](D:/whop-spec-docs/mesh/viralForge/specs/dependencies.yaml)
- [service-category-map.yaml](D:/whop-spec-docs/mesh/viralForge/specs/service-category-map.yaml)
- [service-data-ownership-map.yaml](D:/whop-spec-docs/mesh/viralForge/specs/service-data-ownership-map.yaml)
- [service-audit-matrix.json](D:/whop-spec-docs/mesh/viralForge/specs/service-audit-matrix.json)
- [services-index.yaml](D:/whop-spec-docs/mesh/services/services-index.yaml)
- [implemented-services.yaml](D:/whop-spec-docs/mesh/tooling/manifests/implemented-services.yaml)

## Deprecated Alias Cleanup (Required Before Phase 0 Exit)
- `M27 -> M08`, `M28 -> M07`, `M29 -> M09`, `M32 -> M11`, `M33 -> M12`, `M40 -> M13`, `M42 -> M14`, `M43 -> M15`, `M59 -> M03`, `M63 -> M50`, `M64 -> M51`, `M75 -> M65`, `M76 -> M16`, `M81 -> M18`, `M82 -> M19`, `M94 -> M30`.
- Rule: no new implementation work on deprecated IDs; all references normalize to successor IDs.

## Hierarchical Roadmap (Independent First, Then Dependents)

### Phase 0: Independent Foundations (No active inbound deps)
- Start order priority inside phase: `M01`, `M09`, `M10`, `M11`, `M13`, `M60`, `M89`, then remaining independent modules.
- Microservice track: `M01, M10, M11, M13, M17, M18, M19, M38, M45, M51, M52, M57, M66, M67, M68, M69, M70, M71, M72, M73, M77, M78, M79, M80, M83, M84, M89, M91, M97`.
- Monolith track: `M09, M20, M46, M49, M60, M62, M65, M74, M85, M88`.
- Exit gate: owner APIs + event contracts published for all phase providers.

### Phase 1: First-Order Dependents
- Microservice track: `M02, M16, M30, M50, M95`.
- Monolith track: `M22, M61, M87`.
- Exit gate: auth/profile/team/subscription/social/consent/referral interfaces stable.

### Phase 2: Monolith Core Access Layer
- Monolith track: `M21, M48, M92`.
- Exit gate: reputation and authorization APIs available to downstream fraud/risk/finance/dashboard flows.

### Phase 3: Mixed Core Cycle Group + Gamification
- Coupled cycle group (implement together): `M04, M06, M08, M15, M26, M41`.
- Additional monolith in this tier: `M47`.
- Exit gate: campaign-submission-voting-reward loop runs end-to-end with idempotent events.

### Phase 4: Second-Order Domain Buildout
- Microservice track: `M12, M39`.
- Monolith track: `M07, M23, M24, M31, M34, M35`.
- Exit gate: moderation, finance, discovery, clipping, distribution surfaces available to downstream phases.

### Phase 5: Financial/Risk Cycle Group + Advanced Consumers
- Coupled cycle group (implement together): `M05, M14, M36, M44`.
- Additional microservices: `M25, M54, M58, M96`.
- Additional monoliths: `M37, M53`.
- Exit gate: payout-risk-resolution-billing loop stable; analytics/recommendation pipelines consuming canonical signals.

### Phase 6: Final Aggregation and Operator UX
- Microservice track: `M03, M55, M56`.
- Monolith track: `M86`.
- Exit gate: platform workflows complete (notifications, dashboards, predictive outputs, admin ops).

## Cycle Handling Protocol (Decision Complete)
- For each cycle group, run 3-step implementation pattern:
1. Lock contracts first (owner APIs, event schemas, idempotency keys, error envelopes).
2. Implement minimal “producer-first” capabilities with consumer-tolerant fallbacks.
3. Enable full behavior with feature flags after integration tests pass for the whole cycle group.
- No service in a cycle is considered complete until all members of that cycle pass the shared integration gate.

## Important API/Interface/Type Changes to Plan
- No new canonical module IDs.
- No deprecated event names; only canonical events from `dependencies.yaml`.
- Enforce `owner_api`/`event_projection` access per DB ownership map; remove any direct cross-service DB assumptions.
- Introduce explicit alias-normalization checks in dependency validation (deprecated ID references fail CI).

## Test Cases and Acceptance Scenarios
1. Graph integrity: no unresolved events, no DBR cycles outside declared cycle groups, no deprecated ID references.
2. Contract compatibility: protobuf/OpenAPI/event schema lint + breaking checks for changed contracts.
3. Phase integration suites:
   - Identity flow: `M01 -> M02 -> M22/M21`.
   - Editorial flow: `M04 -> M06 -> M26 -> M08 -> M41`.
   - Financial flow: `M39 -> M05 -> M36 -> M14 -> M44`.
   - Distribution flow: `M10 -> M30 -> M11 -> M54/M55`.
4. Data ownership checks: no cross-service write violations, declared read mode only.
5. Operational checks: startup order by phase, observability baseline present before dependent launches.

## Assumptions and Defaults
- Chosen: **Active + alias cleanup** scope.
- Chosen: **Dual-track roadmap** (Monolith + Mesh microservices).
- Monolith implementation is tracked as Solomon workstream; mesh repo remains microservice implementation boundary.
- Existing implemented microservices in manifest are treated as already delivered but must still pass phase gates during full-program integration.
