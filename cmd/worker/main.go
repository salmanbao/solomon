package main

import "log"

// Worker process entrypoint.
// Data flow:
// 1) Load config.
// 2) Build app wiring.
// 3) Start consumers/schedulers (outbox relay, retries, async jobs).
func main() {
	log.Println("solomon worker starting")
	// TODO: call bootstrap.BuildWorker() and run worker loops.
}
