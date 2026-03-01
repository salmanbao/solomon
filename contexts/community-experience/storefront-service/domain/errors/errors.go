package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrStorefrontNotFound     = errors.New("storefront not found")
	ErrNotFound               = errors.New("resource not found")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrAlreadyPublished       = errors.New("storefront already published")
	ErrPrivateAccessDenied    = errors.New("private storefront access denied")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict")
	ErrDependencyUnavailable  = errors.New("dependency unavailable")
)
