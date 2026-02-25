package errors

import "errors"

var (
	ErrInvalidVoteInput        = errors.New("invalid vote input")
	ErrVoteNotFound            = errors.New("vote not found")
	ErrAlreadyRetracted        = errors.New("vote is already retracted")
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrCampaignNotFound        = errors.New("campaign not found")
	ErrCampaignNotActive       = errors.New("campaign is not active")
	ErrRoundNotFound           = errors.New("voting round not found")
	ErrRoundClosed             = errors.New("voting round is closed")
	ErrSelfVoteForbidden       = errors.New("self voting is forbidden")
	ErrConflict                = errors.New("vote conflict")
	ErrIdempotencyKeyRequired  = errors.New("idempotency key is required")
	ErrIdempotencyConflict     = errors.New("idempotency key conflict")
	ErrQuarantineNotFound      = errors.New("vote quarantine item not found")
	ErrInvalidQuarantineAction = errors.New("invalid quarantine action")
	ErrQuarantineResolved      = errors.New("quarantine item is already resolved")
)
