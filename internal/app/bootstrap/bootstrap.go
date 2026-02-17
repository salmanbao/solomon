package bootstrap

// Package bootstrap is the composition root.
// Keep construction/wiring here so module code stays framework-agnostic.

type APIApp struct {
	// TODO: HTTP server instance, module handlers, health checks.
}

type WorkerApp struct {
	// TODO: consumers, schedulers, outbox relay.
}

func BuildAPI() (*APIApp, error) {
	// TODO: create infra adapters, then inject ports into modules.
	return &APIApp{}, nil
}

func BuildWorker() (*WorkerApp, error) {
	// TODO: create worker pipelines and retry policies.
	return &WorkerApp{}, nil
}
