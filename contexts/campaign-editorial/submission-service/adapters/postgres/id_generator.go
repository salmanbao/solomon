package postgresadapter

import (
	"context"

	"github.com/google/uuid"
)

// UUIDGenerator creates stable UUIDv4 identifiers for M26 entities/events.
type UUIDGenerator struct{}

func (UUIDGenerator) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
