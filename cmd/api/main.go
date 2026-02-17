package main

import "log"

// API process entrypoint.
// Data flow:
// 1) Load config.
// 2) Build app wiring (ports + adapters + use cases).
// 3) Start HTTP server.
func main() {
	log.Println("solomon api starting")
	// TODO: call bootstrap.BuildAPI() and start server.
}
