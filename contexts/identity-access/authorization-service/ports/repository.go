package ports

import "context"

// Repository ports shared by adapters and use cases.
type Repository interface {
	AssignRole(ctx context.Context, userID string, roleID string) error
	ListPermissions(ctx context.Context, userID string) ([]string, error)
}
