// Package authorization implements M21 Authorization Service inside Solomon.
//
// Layering:
// - domain: core entities, invariants, errors
// - application: commands/queries/workers using explicit ports
// - ports: stable boundaries for persistence/cache/events
// - adapters: concrete HTTP, memory, postgres, and event publisher implementations
// - transport: module-private DTOs for HTTP contracts
//
// Boundary notes:
// - Keep this module self-contained under identity-access context.
// - Do not import other context adapters into domain/application.
// - Cross-service reads must follow canonical DBR definitions from viralForge specs.
package authorization
