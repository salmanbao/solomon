package commands

import (
	"encoding/json"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/ports"
)

func newSubmissionEnvelope(
	eventID string,
	eventType string,
	submissionID string,
	occurredAt time.Time,
	data map[string]any,
) (ports.EventEnvelope, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return ports.EventEnvelope{}, err
	}
	return ports.EventEnvelope{
		EventID:          eventID,
		EventType:        eventType,
		OccurredAt:       occurredAt.UTC(),
		SourceService:    "submission-service",
		TraceID:          eventID,
		SchemaVersion:    1,
		PartitionKeyPath: "submission_id",
		PartitionKey:     submissionID,
		Data:             payload,
	}, nil
}
