package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrNotFound               = errors.New("resource not found")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrRateLimited            = errors.New("rate limit exceeded")

	ErrMessageNotFound    = errors.New("message not found")
	ErrAttachmentNotFound = errors.New("attachment not found")
)
