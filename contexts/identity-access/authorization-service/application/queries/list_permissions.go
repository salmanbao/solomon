package queries

import "context"

// PermissionReader is query-side read port.
type PermissionReader interface {
	ListPermissions(ctx context.Context, userID string) ([]string, error)
}

// ListPermissionsUseCase keeps query logic in application layer.
type ListPermissionsUseCase struct {
	Reader PermissionReader
}

func (u ListPermissionsUseCase) Execute(ctx context.Context, userID string) ([]string, error) {
	return u.Reader.ListPermissions(ctx, userID)
}
