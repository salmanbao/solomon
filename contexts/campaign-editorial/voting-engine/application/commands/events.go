package commands

import (
	"encoding/json"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

func newVotingEnvelope(
	eventID string,
	eventType string,
	submissionID string,
	occurredAt time.Time,
	data map[string]any,
) (ports.EventEnvelope, error) {
	// Command-side events are partitioned by submission for stable ordering on
	// submission-scoped consumers.
	payload, err := json.Marshal(data)
	if err != nil {
		return ports.EventEnvelope{}, err
	}
	return ports.EventEnvelope{
		EventID:          eventID,
		EventType:        eventType,
		OccurredAt:       occurredAt.UTC(),
		SourceService:    "voting-engine",
		TraceID:          eventID,
		SchemaVersion:    1,
		PartitionKeyPath: "submission_id",
		PartitionKey:     submissionID,
		Data:             payload,
	}, nil
}
