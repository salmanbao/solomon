package errors

import "errors"

var (
	ErrCampaignNotFound       = errors.New("campaign not found")
	ErrInvalidCampaignInput   = errors.New("invalid campaign input")
	ErrCampaignNotEditable    = errors.New("campaign cannot be edited in current state")
	ErrInvalidStateTransition = errors.New("invalid campaign state transition")
	ErrInvalidBudgetIncrease  = errors.New("budget increase must be positive")
	ErrMediaNotFound          = errors.New("media not found")
	ErrMediaAlreadyConfirmed  = errors.New("media already confirmed")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyConflict = errors.New("idempotency key conflict")
)
