---
name: solomon-swagger-integrator
description: Write and maintain Swagger/OpenAPI docs for Solomon Go HTTP endpoints, generate Swagger artifacts, and integrate Swagger UI routes in the monolith router. Use this skill when adding or changing API handlers, request/response DTOs, route registration, or API metadata in `cmd/api` and `internal/platform/httpserver`.
---

# Solomon Swagger Integrator

Use this skill to keep endpoint behavior, generated Swagger artifacts, and router exposure aligned in one change set.

## Load First
- `references/swagger-playbook.md`

## Workflow
1. Identify changed handlers and transport DTOs.
2. Add or update Swagger annotations for endpoints and models.
3. Ensure API-level metadata annotations exist at the API entrypoint.
4. Generate Swagger docs artifacts and commit them.
5. Wire Swagger UI/docs route in router setup.
6. Verify docs render and project tests still pass.

## Repo Anchors
- `cmd/api/main.go`
- `internal/platform/httpserver/server.go`
- `contexts/*/*/adapters/http/*.go`
- `contexts/*/*/transport/http/*.go`

## Non-Negotiables
- Keep module boundaries intact: no business logic in router or generated docs packages.
- Keep endpoint docs in sync with DTOs and status codes from the same commit.
- Keep Swagger route registration in monolith runtime wiring, not inside domain/application layers.
