package errors

import "errors"

var (
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrInvalidSubmissionInput  = errors.New("invalid submission input")
	ErrInvalidStatusTransition = errors.New("invalid submission status transition")
	ErrUnauthorizedActor       = errors.New("actor is not authorized")
	ErrDuplicateSubmission     = errors.New("duplicate submission")
	ErrUnsupportedPlatform     = errors.New("unsupported platform")
	ErrPlatformNotAllowed      = errors.New("platform not allowed for campaign")
	ErrCampaignNotFound        = errors.New("campaign not found")
	ErrCampaignNotActive       = errors.New("campaign is not active")
	ErrInvalidSubmissionURL    = errors.New("invalid submission url")
	ErrIdempotencyKeyRequired  = errors.New("idempotency key is required")
	ErrIdempotencyKeyConflict  = errors.New("idempotency key conflict")
	ErrAlreadyReported         = errors.New("submission already reported by this user")
)
