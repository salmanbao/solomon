# Implementation Checklist

## Before Coding
- Service is `architecture: monolith`.
- Owned tables and read dependencies are identified.
- Endpoint and event contracts are identified.

## Before Merge
- Layer boundaries preserved.
- No undeclared DBR/event dependency introduced.
- Tests added for success/failure/boundary.
- `gofmt -w .` and `go test ./...` pass.
- Migration safety reviewed if schema changed.