.PHONY: test lint check-boundaries

GOLANGCI_LINT ?= golangci-lint

test:
	go test ./...

check-boundaries:
	go run ./scripts/check_boundaries.go

lint: check-boundaries
	$(GOLANGCI_LINT) run
