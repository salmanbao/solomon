package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrNotFound               = errors.New("resource not found")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrUnauthorizedWebhook    = errors.New("webhook signature is invalid")
)
