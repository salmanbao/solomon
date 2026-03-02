package errors

import "errors"

var (
	ErrInvalidInput          = errors.New("platform fee input is invalid")
	ErrIdempotencyKeyMissing = errors.New("idempotency key is required")
	ErrIdempotencyConflict   = errors.New("idempotency key already used with different payload")
	ErrNotFound              = errors.New("platform fee calculation not found")
)
