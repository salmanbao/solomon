package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateVoteRequest struct {
	SubmissionID string `json:"submission_id"`
	CampaignID   string `json:"campaign_id"`
	VoteType     string `json:"vote_type"`
}

type VoteResponse struct {
	VoteID       string  `json:"vote_id"`
	SubmissionID string  `json:"submission_id"`
	CampaignID   string  `json:"campaign_id"`
	UserID       string  `json:"user_id"`
	VoteType     string  `json:"vote_type"`
	Weight       float64 `json:"weight"`
	Retracted    bool    `json:"retracted"`
	Replayed     bool    `json:"replayed"`
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
	Weighted     float64 `json:"weighted_score"`
	Upvotes      int     `json:"upvotes"`
	Downvotes    int     `json:"downvotes"`
}

type LeaderboardResponse struct {
	Items []LeaderboardItem `json:"items"`
}

type RoundResultsResponse struct {
	RoundID string            `json:"round_id"`
	Items   []LeaderboardItem `json:"items"`
}
