# Decoupling Playbook

## Stepwise Refactor Pattern
1. Introduce interface ports for external dependencies.
2. Move domain rules from adapters into domain/application layers.
3. Encapsulate data access behind repository/query ports.
4. Add translation layer for external DTOs/events.
5. Add contract tests before and after refactor.

## Boundary Hardening Tactics
- Define anti-corruption adapters for upstream systems.
- Limit shared utility packages to technical primitives only.
- Keep module events explicit and versioned.
- Separate read models from write models when needed.

## Exit Criteria
- Module can run behind a well-defined API/event boundary.
- Data ownership boundaries do not rely on in-process shortcuts.
- Coupling is low enough to extract without rewriting core rules.