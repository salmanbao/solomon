package events

import "time"

// Envelope is the shared event shape used in Solomon.
// Align fields with repository canonical event contract.
type Envelope struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	SourceService  string    `json:"source_service"`
	OccurredAtUTC  time.Time `json:"occurred_at_utc"`
	CorrelationID  string    `json:"correlation_id"`
	EntityType     string    `json:"entity_type"`
	EntityID       string    `json:"entity_id"`
	PayloadVersion int       `json:"payload_version"`
	Payload        any       `json:"payload"`
}
