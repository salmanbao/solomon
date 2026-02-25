package events

import contractsv1 "solomon/contracts/gen/events/v1"

// Envelope reuses the versioned contracts module type to avoid drift.
type Envelope = contractsv1.Envelope
