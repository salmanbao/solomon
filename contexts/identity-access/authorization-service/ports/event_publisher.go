package ports

import "context"

// EventPublisher publishes module events through outbox/event bus adapter.
type EventPublisher interface {
	PublishRoleAssigned(ctx context.Context, userID string, roleID string) error
}
