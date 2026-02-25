package errors

import "errors"

var (
	ErrInvalidPermission      = errors.New("invalid permission")
	ErrInvalidUserID          = errors.New("invalid user id")
	ErrInvalidRoleID          = errors.New("invalid role id")
	ErrInvalidAdminID         = errors.New("invalid admin id")
	ErrInvalidDelegation      = errors.New("invalid delegation")
	ErrRoleNotFound           = errors.New("role not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrRoleAlreadyAssigned    = errors.New("role already assigned")
	ErrRoleNotAssigned        = errors.New("role not assigned")
	ErrForbidden              = errors.New("forbidden")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key conflict")
)
