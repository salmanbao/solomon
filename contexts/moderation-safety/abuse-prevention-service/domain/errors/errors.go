package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different payload")
	ErrThreatNotFound         = errors.New("abuse threat not found")
)
