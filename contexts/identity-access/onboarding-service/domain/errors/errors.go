package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrNotFound               = errors.New("resource not found")
	ErrFlowNotFound           = errors.New("flow not found")
	ErrProgressNotFound       = errors.New("progress not found")
	ErrStepNotFound           = errors.New("step not found")
	ErrStepAlreadyCompleted   = errors.New("step already completed")
	ErrFlowAlreadyCompleted   = errors.New("flow already completed")
	ErrResumeNotAllowed       = errors.New("resume not allowed")
	ErrConflict               = errors.New("conflict")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict")
	ErrSchemaInvalid          = errors.New("event schema invalid")
	ErrUnknownRole            = errors.New("unknown role")
	ErrDependencyUnavailable  = errors.New("dependency unavailable")
)
