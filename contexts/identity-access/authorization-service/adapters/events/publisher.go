package events

import "context"

// Publisher adapter for domain events.
// Recommended: write to outbox table and let worker relay publish.
type Publisher struct{}

func NewPublisher() *Publisher { return &Publisher{} }

func (p Publisher) PublishRoleAssigned(_ context.Context, _ string, _ string) error {
	// TODO: enqueue role assignment event.
	return nil
}
