package errors

import "errors"

var (
	ErrDistributionItemNotFound = errors.New("distribution item not found")
	ErrInvalidDistributionInput = errors.New("invalid distribution input")
	ErrInvalidScheduleWindow    = errors.New("scheduled time must be within the next 30 days")
	ErrInvalidStateTransition   = errors.New("invalid distribution state transition")
)
