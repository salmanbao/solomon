package outbox

// Outbox row persisted inside the same DB transaction as state changes.
// Worker relay reads pending rows and publishes to message bus.
type Message struct {
	ID         string
	EventType  string
	Payload    []byte
	Status     string // pending, published, failed
	RetryCount int
}
