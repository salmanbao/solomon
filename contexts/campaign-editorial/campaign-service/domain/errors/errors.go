package errors

import "errors"

var (
	ErrCampaignNotFound       = errors.New("campaign not found")
	ErrInvalidCampaignInput   = errors.New("invalid campaign input")
	ErrCampaignNotEditable    = errors.New("campaign cannot be edited in current state")
	ErrCampaignEditRestricted = errors.New("campaign edit restricted in current state")
	ErrInvalidStateTransition = errors.New("invalid campaign state transition")
	ErrInvalidBudgetIncrease  = errors.New("budget increase must be positive")
	ErrMediaNotFound          = errors.New("media not found")
	ErrMediaAlreadyConfirmed  = errors.New("media already confirmed")
	ErrMediaFileTooLarge      = errors.New("media file exceeds 500MB limit")
	ErrUnsupportedMediaType   = errors.New("unsupported media content type")
	ErrMediaLimitReached      = errors.New("campaign media limit reached")
	ErrDeadlineTooSoon        = errors.New("deadline must be at least 7 days from now")
	ErrMissingReadyMedia      = errors.New("upload and process at least 1 media file")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyConflict = errors.New("idempotency key conflict")
)
