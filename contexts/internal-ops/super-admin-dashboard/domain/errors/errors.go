package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrNotFound               = errors.New("resource not found")
	ErrConflict               = errors.New("conflict")
	ErrForbidden              = errors.New("forbidden")
	ErrUnprocessable          = errors.New("unprocessable request")

	ErrUserNotFound               = errors.New("user not found")
	ErrCampaignNotFound           = errors.New("campaign not found")
	ErrSubmissionNotFound         = errors.New("submission not found")
	ErrFlagNotFound               = errors.New("feature flag not found")
	ErrImpersonationNotFound      = errors.New("impersonation not found")
	ErrImpersonationAlreadyActive = errors.New("impersonation already active")
	ErrImpersonationAlreadyEnded  = errors.New("impersonation already ended")
	ErrAlreadyBanned              = errors.New("user already banned")
	ErrNotBanned                  = errors.New("user is not currently banned")
	ErrBulkActionConflict         = errors.New("bulk action conflict")
)