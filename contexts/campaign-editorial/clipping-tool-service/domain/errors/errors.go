package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrNotFound               = errors.New("resource not found")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrDependencyUnavailable  = errors.New("dependency unavailable")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrProjectExporting       = errors.New("project already exporting")
)
