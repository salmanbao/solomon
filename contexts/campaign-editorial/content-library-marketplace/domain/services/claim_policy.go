package services

import (
	"time"

	"solomon/contexts/campaign-editorial/content-library-marketplace/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
)

// EvaluateClaimEligibility enforces exclusivity, claim limit, and active clip constraints.
func EvaluateClaimEligibility(
	clip entities.Clip,
	existingClaims []entities.Claim,
	userID string,
	now time.Time,
) (*entities.Claim, error) {
	if !clip.IsClaimable() {
		return nil, domainerrors.ErrClipUnavailable
	}

	occupyingCount := 0
	for _, claim := range existingClaims {
		if !claim.OccupiesSlot(now) {
			continue
		}

		if claim.UserID == userID {
			existing := claim
			return &existing, nil
		}
		occupyingCount++
	}

	limit := clip.EffectiveClaimLimit()
	if clip.Exclusivity == entities.ClipExclusivityExclusive && occupyingCount > 0 {
		return nil, domainerrors.ErrExclusiveClaimConflict
	}
	if clip.Exclusivity == entities.ClipExclusivityNonExclusive && occupyingCount >= limit {
		return nil, domainerrors.ErrClaimLimitReached
	}

	return nil, nil
}
