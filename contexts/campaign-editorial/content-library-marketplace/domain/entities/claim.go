package entities

import (
	"strings"
	"time"

	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
)

type ClaimType string

const (
	ClaimTypeExclusive    ClaimType = "exclusive"
	ClaimTypeNonExclusive ClaimType = "non_exclusive"
)

type ClaimStatus string

const (
	ClaimStatusActive    ClaimStatus = "active"
	ClaimStatusPublished ClaimStatus = "published"
	ClaimStatusPaid      ClaimStatus = "paid"
	ClaimStatusExpired   ClaimStatus = "expired"
	ClaimStatusCancelled ClaimStatus = "cancelled"
	ClaimStatusFailed    ClaimStatus = "failed"
)

type Claim struct {
	ClaimID   string
	ClipID    string
	UserID    string
	ClaimType ClaimType
	Status    ClaimStatus
	RequestID string
	ClaimedAt time.Time
	ExpiresAt time.Time
	UpdatedAt time.Time
}

func NewClaim(
	claimID string,
	clipID string,
	userID string,
	claimType ClaimType,
	requestID string,
	claimedAt time.Time,
	expiresAt time.Time,
) (Claim, error) {
	if strings.TrimSpace(claimID) == "" ||
		strings.TrimSpace(clipID) == "" ||
		strings.TrimSpace(userID) == "" ||
		strings.TrimSpace(requestID) == "" {
		return Claim{}, domainerrors.ErrInvalidClaimRequest
	}
	if !expiresAt.After(claimedAt) {
		return Claim{}, domainerrors.ErrInvalidClaimRequest
	}
	if claimType != ClaimTypeExclusive && claimType != ClaimTypeNonExclusive {
		return Claim{}, domainerrors.ErrInvalidClaimRequest
	}

	return Claim{
		ClaimID:   claimID,
		ClipID:    clipID,
		UserID:    userID,
		ClaimType: claimType,
		Status:    ClaimStatusActive,
		RequestID: requestID,
		ClaimedAt: claimedAt.UTC(),
		ExpiresAt: expiresAt.UTC(),
		UpdatedAt: claimedAt.UTC(),
	}, nil
}

// OccupiesSlot determines whether the claim should count toward claim capacity.
// Published claims remain slot-occupying even after active expiry checks.
func (c Claim) OccupiesSlot(now time.Time) bool {
	switch c.Status {
	case ClaimStatusActive:
		return !now.UTC().After(c.ExpiresAt)
	case ClaimStatusPublished:
		return true
	default:
		return false
	}
}
