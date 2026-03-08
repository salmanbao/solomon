package errors

import "errors"

var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrNotFound              = errors.New("not found")
	ErrConflict              = errors.New("conflict")
	ErrIdempotencyRequired   = errors.New("idempotency key required")
	ErrIdempotencyConflict   = errors.New("idempotency key reused with different payload")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
	ErrUnsupportedAction     = errors.New("unsupported action")
)
