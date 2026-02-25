package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateVoteRequest struct {
	SubmissionID string `json:"submission_id"`
	CampaignID   string `json:"campaign_id"`
	RoundID      string `json:"round_id,omitempty"`
	VoteType     string `json:"vote_type"`
}

type VoteResponse struct {
	VoteID       string  `json:"vote_id"`
	SubmissionID string  `json:"submission_id"`
	CampaignID   string  `json:"campaign_id"`
	RoundID      string  `json:"round_id,omitempty"`
	UserID       string  `json:"user_id"`
	VoteType     string  `json:"vote_type"`
	Weight       float64 `json:"weight"`
	Retracted    bool    `json:"retracted"`
	Replayed     bool    `json:"replayed"`
	WasUpdate    bool    `json:"was_update"`
}

type SubmissionVotesResponse struct {
	SubmissionID string  `json:"submission_id"`
	Upvotes      int     `json:"upvotes"`
	Downvotes    int     `json:"downvotes"`
	Weighted     float64 `json:"weighted_score"`
}

type LeaderboardItem struct {
	SubmissionID string  `json:"submission_id"`
	CampaignID   string  `json:"campaign_id"`
	RoundID      string  `json:"round_id,omitempty"`
	Weighted     float64 `json:"weighted_score"`
	Upvotes      int     `json:"upvotes"`
	Downvotes    int     `json:"downvotes"`
	Rank         int     `json:"rank"`
}

type LeaderboardResponse struct {
	Items []LeaderboardItem `json:"items"`
}

type RoundResultsResponse struct {
	RoundID    string            `json:"round_id"`
	CampaignID string            `json:"campaign_id"`
	Status     string            `json:"status"`
	Closed     bool              `json:"closed"`
	Items      []LeaderboardItem `json:"items"`
}

type VoteAnalyticsResponse struct {
	TotalVotes          int     `json:"total_votes"`
	ActiveVotes         int     `json:"active_votes"`
	RetractedVotes      int     `json:"retracted_votes"`
	UniqueVoters        int     `json:"unique_voters"`
	PendingQuarantined  int     `json:"pending_quarantined"`
	ApprovedQuarantined int     `json:"approved_quarantined"`
	RejectedQuarantined int     `json:"rejected_quarantined"`
	WeightedScore       float64 `json:"weighted_score"`
}

type QuarantineActionRequest struct {
	Action string `json:"action"`
}
