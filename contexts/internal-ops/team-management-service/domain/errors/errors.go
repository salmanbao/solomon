package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrNotFound               = errors.New("resource not found")
	ErrTeamNotFound           = errors.New("team not found")
	ErrInviteNotFound         = errors.New("invite not found")
	ErrMemberNotFound         = errors.New("member not found")
	ErrForbidden              = errors.New("forbidden")
	ErrConflict               = errors.New("conflict")
	ErrInviteExpired          = errors.New("invite expired")
	ErrOwnerTransferRequired  = errors.New("owner transfer required")
	ErrMFARequired            = errors.New("mfa required")
	ErrIdempotencyKeyRequired = errors.New("idempotency key required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict")
	ErrDependencyUnavailable  = errors.New("dependency unavailable")
)
