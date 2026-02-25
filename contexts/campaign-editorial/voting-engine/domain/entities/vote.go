package entities

import "time"

type VoteType string

const (
	VoteTypeUpvote   VoteType = "upvote"
	VoteTypeDownvote VoteType = "downvote"
)

type Vote struct {
	VoteID                  string
	SubmissionID            string
	CampaignID              string
	RoundID                 string
	UserID                  string
	VoteType                VoteType
	Weight                  float64
	ReputationScoreSnapshot float64
	IPAddress               string
	UserAgent               string
	Retracted               bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
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
	RoundID      string
	Upvotes      int
	Downvotes    int
	Weighted     float64
	FirstVoteAt  time.Time
	LastVoteAt   time.Time
}

type RoundStatus string

const (
	RoundStatusScheduled   RoundStatus = "scheduled"
	RoundStatusActive      RoundStatus = "active"
	RoundStatusClosingSoon RoundStatus = "closing_soon"
	RoundStatusClosed      RoundStatus = "closed"
	RoundStatusArchived    RoundStatus = "archived"
)

type VotingRound struct {
	RoundID    string
	CampaignID string
	Status     RoundStatus
	StartsAt   time.Time
	EndsAt     *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type QuarantineStatus string

const (
	QuarantineStatusPendingReview QuarantineStatus = "pending_review"
	QuarantineStatusApproved      QuarantineStatus = "approved"
	QuarantineStatusRejected      QuarantineStatus = "rejected"
)

type VoteQuarantine struct {
	QuarantineID string
	VoteID       string
	RiskScore    float64
	Reason       string
	Status       QuarantineStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type VoteAnalytics struct {
	TotalVotes          int
	ActiveVotes         int
	RetractedVotes      int
	PendingQuarantined  int
	ApprovedQuarantined int
	RejectedQuarantined int
	UniqueVoters        int
	WeightedScore       float64
}
