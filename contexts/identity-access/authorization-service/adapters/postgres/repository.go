package postgres

import "context"

// Repository adapter for PostgreSQL.
// In production this should use transactions with outbox where required.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

func (r Repository) AssignRole(_ context.Context, _ string, _ string) error {
	// TODO: write assignment row(s).
	return nil
}

func (r Repository) ListPermissions(_ context.Context, _ string) ([]string, error) {
	// TODO: query joined role/permission tables.
	return []string{}, nil
}
