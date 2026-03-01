package errors

import "errors"

var (
	ErrInvalidRequest         = errors.New("invalid request")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyConflict    = errors.New("idempotency key reused with different request")
	ErrNotFound               = errors.New("resource not found")
	ErrConflict               = errors.New("conflict")
	ErrForbidden              = errors.New("forbidden")
	ErrPaymentRequired        = errors.New("payment required")

	ErrPlanNotFound          = errors.New("subscription plan not found")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrPlanInactive          = errors.New("subscription plan is inactive")
	ErrTrialAlreadyUsed      = errors.New("trial already used")
	ErrInvalidTransition     = errors.New("subscription state transition is not allowed")
	ErrDependencyUnavailable = errors.New("required product reference not found")
)
