package commands

import (
	"encoding/json"
	"time"

	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

func newCampaignEnvelope(
	eventID string,
	eventType string,
	campaignID string,
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
		SourceService:    "campaign-service",
		TraceID:          eventID,
		SchemaVersion:    1,
		PartitionKeyPath: "campaign_id",
		PartitionKey:     campaignID,
		Data:             payload,
	}, nil
}
