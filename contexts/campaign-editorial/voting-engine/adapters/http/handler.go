package httpadapter

import (
	"context"
	"log/slog"

	"solomon/contexts/campaign-editorial/voting-engine/application/commands"
	"solomon/contexts/campaign-editorial/voting-engine/application/queries"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/voting-engine/transport/http"
)

type Handler struct {
	Votes        commands.VoteUseCase
	Leaderboards queries.LeaderboardUseCase
	Logger       *slog.Logger
}

func (h Handler) CreateVoteHandler(
	ctx context.Context,
	userID string,
	idempotencyKey string,
	req httptransport.CreateVoteRequest,
) (httptransport.VoteResponse, error) {
	result, err := h.Votes.CreateVote(ctx, commands.CreateVoteCommand{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		SubmissionID:   req.SubmissionID,
		CampaignID:     req.CampaignID,
		VoteType:       entities.VoteType(req.VoteType),
	})
	if err != nil {
		return httptransport.VoteResponse{}, err
	}
	return httptransport.VoteResponse{
		VoteID:       result.Vote.VoteID,
		SubmissionID: result.Vote.SubmissionID,
		CampaignID:   result.Vote.CampaignID,
		UserID:       result.Vote.UserID,
		VoteType:     string(result.Vote.VoteType),
		Weight:       result.Vote.Weight,
		Retracted:    result.Vote.Retracted,
		Replayed:     result.Replayed,
	}, nil
}

func (h Handler) RetractVoteHandler(ctx context.Context, voteID string, userID string) error {
	return h.Votes.RetractVote(ctx, commands.RetractVoteCommand{
		VoteID: voteID,
		UserID: userID,
	})
}

func (h Handler) SubmissionVotesHandler(ctx context.Context, submissionID string) (httptransport.SubmissionVotesResponse, error) {
	score, err := h.Leaderboards.SubmissionVotes(ctx, submissionID)
	if err != nil {
		return httptransport.SubmissionVotesResponse{}, err
	}
	return httptransport.SubmissionVotesResponse{
		SubmissionID: score.SubmissionID,
		Upvotes:      score.Upvotes,
		Downvotes:    score.Downvotes,
		Weighted:     score.Weighted,
	}, nil
}

func (h Handler) CampaignLeaderboardHandler(ctx context.Context, campaignID string) (httptransport.LeaderboardResponse, error) {
	scores, err := h.Leaderboards.CampaignLeaderboard(ctx, campaignID)
	if err != nil {
		return httptransport.LeaderboardResponse{}, err
	}
	return httptransport.LeaderboardResponse{
		Items: mapLeaderboard(scores),
	}, nil
}

func (h Handler) TrendingLeaderboardHandler(ctx context.Context) (httptransport.LeaderboardResponse, error) {
	scores, err := h.Leaderboards.GlobalTrending(ctx)
	if err != nil {
		return httptransport.LeaderboardResponse{}, err
	}
	return httptransport.LeaderboardResponse{
		Items: mapLeaderboard(scores),
	}, nil
}

func (h Handler) CreatorLeaderboardHandler(ctx context.Context, _ string) (httptransport.LeaderboardResponse, error) {
	return h.TrendingLeaderboardHandler(ctx)
}

func (h Handler) RoundResultsHandler(ctx context.Context, roundID string) (httptransport.RoundResultsResponse, error) {
	scores, err := h.Leaderboards.GlobalTrending(ctx)
	if err != nil {
		return httptransport.RoundResultsResponse{}, err
	}
	return httptransport.RoundResultsResponse{
		RoundID: roundID,
		Items:   mapLeaderboard(scores),
	}, nil
}

func (h Handler) VoteAnalyticsHandler(ctx context.Context) (httptransport.LeaderboardResponse, error) {
	return h.TrendingLeaderboardHandler(ctx)
}

func (h Handler) QuarantineActionHandler(_ context.Context, _ string, _ string) error {
	return nil
}

func mapLeaderboard(scores []entities.SubmissionScore) []httptransport.LeaderboardItem {
	items := make([]httptransport.LeaderboardItem, 0, len(scores))
	for _, score := range scores {
		items = append(items, httptransport.LeaderboardItem{
			SubmissionID: score.SubmissionID,
			CampaignID:   score.CampaignID,
			Weighted:     score.Weighted,
			Upvotes:      score.Upvotes,
			Downvotes:    score.Downvotes,
		})
	}
	return items
}
