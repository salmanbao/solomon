package v1

import (
	"encoding/json"
	"time"
)

// Envelope is the canonical, versioned event envelope for cross-runtime use.
// This package is generated-contract-only and must stay backward compatible.
type Envelope struct {
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	OccurredAt       time.Time       `json:"occurred_at"`
	SourceService    string          `json:"source_service"`
	TraceID          string          `json:"trace_id"`
	SchemaVersion    int             `json:"schema_version"`
	PartitionKeyPath string          `json:"partition_key_path"`
	PartitionKey     string          `json:"partition_key"`
	Data             json.RawMessage `json:"data"`
}
