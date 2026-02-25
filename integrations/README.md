# Integrations Boundary

`integrations/` is documentation/policy only.

Rules:
- Do not build a central `integrations/microservices` client layer.
- Each module owns outbound microservice adapters in:
  `contexts/<context>/<service>/adapters/...`
- Shared low-level client primitives (retry, auth headers, tracing middleware) belong in `internal/shared`.
