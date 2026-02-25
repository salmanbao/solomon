# Swagger Playbook

## Goal
Document Solomon HTTP endpoints with Swagger annotations and expose a working Swagger UI route from the API router.

## Baseline
1. Read `cmd/api/main.go` and `internal/platform/httpserver/server.go`.
2. Identify endpoint handlers under `contexts/*/*/adapters/http`.
3. Identify request/response DTOs under `contexts/*/*/transport/http`.

## Preferred Libraries
1. Generator CLI: `github.com/swaggo/swag/cmd/swag`
2. UI handler: `github.com/swaggo/http-swagger`
3. Static files dependency: `github.com/swaggo/files`

Use existing project choices first if Swagger tooling is already present.

## API Metadata Annotations
Add or update API-level annotations in `cmd/api/main.go` (or the same package file used for `swag init -g`):

```go
// Package main Solomon API process.
//
// @title Solomon API
// @version 1.0
// @description Solomon monolith HTTP API
// @BasePath /
package main
```

Keep metadata minimal and accurate.

## Endpoint Annotation Pattern
Annotate each HTTP operation near the adapter handler used by the route.

```go
// AssignRole godoc
// @Summary Assign a role to a user
// @Tags authorization
// @Accept json
// @Produce json
// @Param request body httptransport.AssignRoleRequest true "Assign role payload"
// @Success 200 {object} httptransport.AssignRoleResponse
// @Failure 400 {object} map[string]string
// @Router /v1/authorization/roles/assign [post]
func (h Handler) AssignRole(ctx context.Context, userID string, roleID string) error { ... }
```

Rules:
1. Use DTO types from `transport/http` for request/response schemas.
2. Keep `@Router` path and method identical to actual router registration.
3. Reflect real status codes and error shapes.

## Generate Swagger Artifacts
Run from repository root:

```powershell
go run github.com/swaggo/swag/cmd/swag@latest init `
  -g cmd/api/main.go `
  -o internal/platform/httpserver/docs `
  --parseDependency `
  --parseInternal
```

Expected generated files:
1. `internal/platform/httpserver/docs/docs.go`
2. `internal/platform/httpserver/docs/swagger.json`
3. `internal/platform/httpserver/docs/swagger.yaml`

Commit generated artifacts with the endpoint changes.

## Router Integration Patterns
If using `net/http` with `ServeMux`:

```go
import (
  httpSwagger "github.com/swaggo/http-swagger"
  _ "solomon/internal/platform/httpserver/docs"
)

mux.Handle("/swagger/", httpSwagger.WrapHandler)
```

If using `chi`:

```go
r.Get("/swagger/*", httpSwagger.Handler(
  httpSwagger.URL("/swagger/doc.json"),
))
```

Keep Swagger route wiring in `internal/platform/httpserver/server.go` or equivalent transport bootstrap code.

## Validation Checklist
1. `go test ./...` passes.
2. API starts without panic after importing generated docs package.
3. `/swagger/index.html` loads.
4. UI operation list matches implemented endpoints.
5. DTO fields in Swagger match JSON tags in transport structs.
