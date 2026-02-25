package entities

import "time"

type VoteType string

const (
	VoteTypeUpvote   VoteType = "upvote"
	VoteTypeDownvote VoteType = "downvote"
)

type Vote struct {
	VoteID       string
	SubmissionID string
	CampaignID   string
	UserID       string
	VoteType     VoteType
	Weight       float64
	Retracted    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (v Vote) EffectiveScore() float64 {
	if v.Retracted {
		return 0
	}
	if v.VoteType == VoteTypeDownvote {
		return -1 * v.Weight
	}
	return v.Weight
}

type SubmissionScore struct {
	SubmissionID string
	CampaignID   string
	Upvotes      int
	Downvotes    int
	Weighted     float64
}
