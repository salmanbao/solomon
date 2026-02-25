package errors

import "errors"

var (
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrInvalidSubmissionInput  = errors.New("invalid submission input")
	ErrInvalidStatusTransition = errors.New("invalid submission status transition")
	ErrUnauthorizedActor       = errors.New("actor is not authorized")
	ErrDuplicateSubmission     = errors.New("duplicate submission")
)
