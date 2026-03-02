package errors

import "errors"

var (
	ErrInvalidInput        = errors.New("invalid input")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrIdempotencyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
)
