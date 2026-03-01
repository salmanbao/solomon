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
	ErrSoldOut                = errors.New("product sold out")

	ErrProductNotFound  = errors.New("product not found")
	ErrPurchaseNotFound = errors.New("purchase not found")
)
