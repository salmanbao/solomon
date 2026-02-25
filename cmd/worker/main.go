package main

import (
	"context"
	"log"

	"solomon/internal/app/bootstrap"
)

// Worker process entrypoint.
// Data flow:
// 1) Load config.
// 2) Build app wiring.
// 3) Start consumers/schedulers (outbox relay, retries, async jobs).
func main() {
	log.Println("solomon worker starting")
	app, err := bootstrap.BuildWorker()
	if err != nil {
		log.Fatalf("bootstrap worker failed: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("worker shutdown close failed: %v", err)
		}
	}()

	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("solomon worker stopped with error: %v", err)
	}
}
