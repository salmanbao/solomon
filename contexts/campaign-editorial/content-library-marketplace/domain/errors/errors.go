package errors

import "errors"

var (
	ErrClipNotFound             = errors.New("clip not found")
	ErrClaimNotFound            = errors.New("claim not found")
	ErrInvalidClaimRequest      = errors.New("invalid claim request")
	ErrInvalidListFilter        = errors.New("invalid list filter")
	ErrClipUnavailable          = errors.New("clip is not claimable")
	ErrExclusiveClaimConflict   = errors.New("exclusive clip is already claimed")
	ErrClaimLimitReached        = errors.New("clip claim limit reached")
	ErrClaimRequired            = errors.New("active claim required")
	ErrDownloadLimitReached     = errors.New("download limit reached")
	ErrIdempotencyKeyConflict   = errors.New("idempotency key reused with different request")
	ErrDuplicateRequestID       = errors.New("request_id already used")
	ErrRepositoryInvariantBroke = errors.New("repository invariant violated")
)
