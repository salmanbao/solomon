package workers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

func hashPayload(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// newVotingEnvelope builds canonical envelopes for worker-produced events.
// Workers pass the partition key path explicitly because it varies by topic.
func newVotingEnvelope(
	eventID string,
	eventType string,
	partitionKey string,
	partitionKeyPath string,
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
		SourceService:    "voting-engine",
		TraceID:          eventID,
		SchemaVersion:    1,
		PartitionKeyPath: partitionKeyPath,
		PartitionKey:     partitionKey,
		Data:             payload,
	}, nil
}
