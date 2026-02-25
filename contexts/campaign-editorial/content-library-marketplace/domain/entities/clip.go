package entities

import "time"

type ClipExclusivity string

const (
	ClipExclusivityExclusive    ClipExclusivity = "exclusive"
	ClipExclusivityNonExclusive ClipExclusivity = "non_exclusive"
)

type ClipStatus string

const (
	ClipStatusActive   ClipStatus = "active"
	ClipStatusPaused   ClipStatus = "paused"
	ClipStatusArchived ClipStatus = "archived"
)

type Clip struct {
	ClipID          string
	CampaignID      string
	SubmissionID    string
	Title           string
	Description     string
	Niche           string
	DurationSeconds int
	PreviewURL      string
	DownloadAssetID string
	Exclusivity     ClipExclusivity
	ClaimLimit      int
	Views7d         int
	Votes7d         int
	EngagementRate  float64
	Status          ClipStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (c Clip) IsClaimable() bool {
	return c.Status == ClipStatusActive
}

// EffectiveClaimLimit normalizes clip claim capacity:
// exclusive clips are always single-slot; non-exclusive defaults to 50.
func (c Clip) EffectiveClaimLimit() int {
	if c.Exclusivity == ClipExclusivityExclusive {
		return 1
	}
	if c.ClaimLimit <= 0 {
		return 50
	}
	return c.ClaimLimit
}
