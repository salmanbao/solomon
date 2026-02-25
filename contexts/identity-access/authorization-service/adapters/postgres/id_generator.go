package postgresadapter

import (
	"context"

	"github.com/google/uuid"
)

// UUIDGenerator implements ports.IDGenerator using RFC 4122 UUID v4 values.
type UUIDGenerator struct{}

func (UUIDGenerator) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
