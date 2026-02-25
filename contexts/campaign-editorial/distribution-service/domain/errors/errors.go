package errors

import "errors"

var (
	ErrDistributionItemNotFound = errors.New("distribution item not found")
	ErrDistributionItemExists   = errors.New("distribution item already exists")
	ErrInvalidDistributionInput = errors.New("invalid distribution input")
	ErrInvalidScheduleWindow    = errors.New("scheduled time must be within the next 30 days")
	ErrInvalidTimezone          = errors.New("invalid timezone")
	ErrUnsupportedPlatform      = errors.New("unsupported platform")
	ErrUnauthorizedInfluencer   = errors.New("distribution item is not owned by influencer")
	ErrInvalidStateTransition   = errors.New("invalid distribution state transition")
)
