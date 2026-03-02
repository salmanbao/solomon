package errors

import "errors"

var (
	ErrInvalidInput          = errors.New("gamification input is invalid")
	ErrDependencyUnavailable = errors.New("gamification dependency is unavailable")
	ErrIdempotencyKeyMissing = errors.New("idempotency key is required")
	ErrIdempotencyConflict   = errors.New("idempotency key already used with different payload")
)
