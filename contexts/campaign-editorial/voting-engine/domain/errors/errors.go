package errors

import "errors"

var (
	ErrInvalidVoteInput   = errors.New("invalid vote input")
	ErrVoteNotFound       = errors.New("vote not found")
	ErrAlreadyRetracted   = errors.New("vote is already retracted")
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrConflict           = errors.New("vote conflict")
)
