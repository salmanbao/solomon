// Package main Solomon API process.
//
// @title Solomon API
// @version 1.0
// @description Solomon monolith HTTP API
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log"

	"solomon/internal/app/bootstrap"
)

// API process entrypoint.
// Data flow:
// 1) Load config.
// 2) Build app wiring (ports + adapters + use cases).
// 3) Start HTTP server.
func main() {
	log.Println("solomon api starting")
	app, err := bootstrap.BuildAPI()
	if err != nil {
		log.Fatalf("bootstrap api failed: %v", err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			log.Printf("api shutdown close failed: %v", err)
		}
	}()

	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("solomon api stopped with error: %v", err)
	}
}
