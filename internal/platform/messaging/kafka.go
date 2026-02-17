package messaging

// Kafka is the event bus adapter used by worker/outbox relay.
// Publish only canonical envelopes and track retries/DLQ where required.
type Kafka struct {}

func NewKafka(_ []string) (*Kafka, error) {
	return &Kafka{}, nil
}
