package postgresadapter

import (
	"context"

	"github.com/google/uuid"
)

type UUIDGenerator struct{}

func (UUIDGenerator) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}
