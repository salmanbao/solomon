package httpadapter

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/application/commands"
	"solomon/contexts/campaign-editorial/voting-engine/application/queries"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/voting-engine/transport/http"
)

// Handler is the inbound adapter facade used by the HTTP transport layer.
type Handler struct {
	Votes        commands.VoteUseCase
	Leaderboards queries.LeaderboardUseCase
	Logger       *slog.Logger
}

// CreateVoteHandler maps transport input into CreateVote command and converts
// domain/application output back into the HTTP DTO response.
func (h Handler) CreateVoteHandler(
	ctx context.Context,
	userID string,
	idempotencyKey string,
	req httptransport.CreateVoteRequest,
	ipAddress string,
	userAgent string,
) (httptransport.VoteResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("vote create request received",
		"event", "voting_http_create_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"user_id", strings.TrimSpace(userID),
		"submission_id", strings.TrimSpace(req.SubmissionID),
		"campaign_id", strings.TrimSpace(req.CampaignID),
		"round_id", strings.TrimSpace(req.RoundID),
	)
	result, err := h.Votes.CreateVote(ctx, commands.CreateVoteCommand{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
		SubmissionID:   req.SubmissionID,
		CampaignID:     req.CampaignID,
		RoundID:        req.RoundID,
		VoteType:       entities.VoteType(req.VoteType),
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	})
	if err != nil {
		logger.Error("vote create request failed",
			"event", "voting_http_create_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"user_id", strings.TrimSpace(userID),
			"submission_id", strings.TrimSpace(req.SubmissionID),
			"error", err.Error(),
		)
		return httptransport.VoteResponse{}, err
	}
	response := httptransport.VoteResponse{
		VoteID:       result.Vote.VoteID,
		SubmissionID: result.Vote.SubmissionID,
		CampaignID:   result.Vote.CampaignID,
		RoundID:      result.Vote.RoundID,
		UserID:       result.Vote.UserID,
		VoteType:     string(result.Vote.VoteType),
		Weight:       result.Vote.Weight,
		Retracted:    result.Vote.Retracted,
		Replayed:     result.Replayed,
		WasUpdate:    result.WasUpdate,
	}
	logger.Info("vote create request completed",
		"event", "voting_http_create_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"vote_id", response.VoteID,
		"submission_id", response.SubmissionID,
		"user_id", strings.TrimSpace(userID),
		"replayed", response.Replayed,
		"was_update", response.WasUpdate,
	)
	return response, nil
}

func (h Handler) RetractVoteHandler(
	ctx context.Context,
	voteID string,
	userID string,
	idempotencyKey string,
) error {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("vote retract request received",
		"event", "voting_http_retract_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"vote_id", strings.TrimSpace(voteID),
		"user_id", strings.TrimSpace(userID),
	)
	if err := h.Votes.RetractVote(ctx, commands.RetractVoteCommand{
		VoteID:          voteID,
		UserID:          userID,
		IdempotencyKey:  idempotencyKey,
		RetractionCause: "user_retracted",
	}); err != nil {
		logger.Error("vote retract request failed",
			"event", "voting_http_retract_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"vote_id", strings.TrimSpace(voteID),
			"user_id", strings.TrimSpace(userID),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("vote retract request completed",
		"event", "voting_http_retract_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"vote_id", strings.TrimSpace(voteID),
		"user_id", strings.TrimSpace(userID),
	)
	return nil
}

func (h Handler) SubmissionVotesHandler(ctx context.Context, submissionID string) (httptransport.SubmissionVotesResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("submission votes request received",
		"event", "voting_http_submission_votes_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"submission_id", strings.TrimSpace(submissionID),
	)
	score, err := h.Leaderboards.SubmissionVotes(ctx, submissionID)
	if err != nil {
		logger.Error("submission votes request failed",
			"event", "voting_http_submission_votes_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"submission_id", strings.TrimSpace(submissionID),
			"error", err.Error(),
		)
		return httptransport.SubmissionVotesResponse{}, err
	}
	response := httptransport.SubmissionVotesResponse{
		SubmissionID: score.SubmissionID,
		Upvotes:      score.Upvotes,
		Downvotes:    score.Downvotes,
		Weighted:     score.Weighted,
	}
	logger.Info("submission votes request completed",
		"event", "voting_http_submission_votes_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"submission_id", response.SubmissionID,
		"upvotes", response.Upvotes,
		"downvotes", response.Downvotes,
	)
	return response, nil
}

func (h Handler) CampaignLeaderboardHandler(ctx context.Context, campaignID string) (httptransport.LeaderboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("campaign leaderboard request received",
		"event", "voting_http_campaign_leaderboard_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"campaign_id", strings.TrimSpace(campaignID),
	)
	scores, err := h.Leaderboards.CampaignLeaderboard(ctx, campaignID)
	if err != nil {
		logger.Error("campaign leaderboard request failed",
			"event", "voting_http_campaign_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"campaign_id", strings.TrimSpace(campaignID),
			"error", err.Error(),
		)
		return httptransport.LeaderboardResponse{}, err
	}
	response := httptransport.LeaderboardResponse{Items: mapLeaderboard(scores)}
	logger.Info("campaign leaderboard request completed",
		"event", "voting_http_campaign_leaderboard_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"campaign_id", strings.TrimSpace(campaignID),
		"item_count", len(response.Items),
	)
	return response, nil
}

func (h Handler) RoundLeaderboardHandler(ctx context.Context, roundID string) (httptransport.LeaderboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("round leaderboard request received",
		"event", "voting_http_round_leaderboard_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"round_id", strings.TrimSpace(roundID),
	)
	scores, err := h.Leaderboards.RoundLeaderboard(ctx, roundID)
	if err != nil {
		logger.Error("round leaderboard request failed",
			"event", "voting_http_round_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"round_id", strings.TrimSpace(roundID),
			"error", err.Error(),
		)
		return httptransport.LeaderboardResponse{}, err
	}
	response := httptransport.LeaderboardResponse{Items: mapLeaderboard(scores)}
	logger.Info("round leaderboard request completed",
		"event", "voting_http_round_leaderboard_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"round_id", strings.TrimSpace(roundID),
		"item_count", len(response.Items),
	)
	return response, nil
}

func (h Handler) TrendingLeaderboardHandler(ctx context.Context) (httptransport.LeaderboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("trending leaderboard request received",
		"event", "voting_http_trending_leaderboard_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
	)
	scores, err := h.Leaderboards.GlobalTrending(ctx)
	if err != nil {
		logger.Error("trending leaderboard request failed",
			"event", "voting_http_trending_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"error", err.Error(),
		)
		return httptransport.LeaderboardResponse{}, err
	}
	response := httptransport.LeaderboardResponse{Items: mapLeaderboard(scores)}
	logger.Info("trending leaderboard request completed",
		"event", "voting_http_trending_leaderboard_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"item_count", len(response.Items),
	)
	return response, nil
}

func (h Handler) CreatorLeaderboardHandler(ctx context.Context, creatorID string) (httptransport.LeaderboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("creator leaderboard request received",
		"event", "voting_http_creator_leaderboard_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"creator_id", strings.TrimSpace(creatorID),
	)
	scores, err := h.Leaderboards.CreatorLeaderboard(ctx, creatorID)
	if err != nil {
		logger.Error("creator leaderboard request failed",
			"event", "voting_http_creator_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"creator_id", strings.TrimSpace(creatorID),
			"error", err.Error(),
		)
		return httptransport.LeaderboardResponse{}, err
	}
	response := httptransport.LeaderboardResponse{Items: mapLeaderboard(scores)}
	logger.Info("creator leaderboard request completed",
		"event", "voting_http_creator_leaderboard_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"creator_id", strings.TrimSpace(creatorID),
		"item_count", len(response.Items),
	)
	return response, nil
}

func (h Handler) RoundResultsHandler(ctx context.Context, roundID string) (httptransport.RoundResultsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("round results request received",
		"event", "voting_http_round_results_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"round_id", strings.TrimSpace(roundID),
	)
	round, scores, err := h.Leaderboards.RoundResults(ctx, roundID)
	if err != nil {
		logger.Error("round results request failed",
			"event", "voting_http_round_results_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"round_id", strings.TrimSpace(roundID),
			"error", err.Error(),
		)
		return httptransport.RoundResultsResponse{}, err
	}
	response := httptransport.RoundResultsResponse{
		RoundID:    round.RoundID,
		CampaignID: round.CampaignID,
		Status:     string(round.Status),
		Closed:     round.Status == entities.RoundStatusClosed || round.Status == entities.RoundStatusArchived,
		Items:      mapLeaderboard(scores),
	}
	logger.Info("round results request completed",
		"event", "voting_http_round_results_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"round_id", response.RoundID,
		"campaign_id", response.CampaignID,
		"item_count", len(response.Items),
	)
	return response, nil
}

func (h Handler) VoteAnalyticsHandler(ctx context.Context) (httptransport.VoteAnalyticsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("vote analytics request received",
		"event", "voting_http_analytics_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
	)
	analytics, err := h.Leaderboards.VoteAnalytics(ctx)
	if err != nil {
		logger.Error("vote analytics request failed",
			"event", "voting_http_analytics_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"error", err.Error(),
		)
		return httptransport.VoteAnalyticsResponse{}, err
	}
	response := httptransport.VoteAnalyticsResponse{
		TotalVotes:          analytics.TotalVotes,
		ActiveVotes:         analytics.ActiveVotes,
		RetractedVotes:      analytics.RetractedVotes,
		UniqueVoters:        analytics.UniqueVoters,
		PendingQuarantined:  analytics.PendingQuarantined,
		ApprovedQuarantined: analytics.ApprovedQuarantined,
		RejectedQuarantined: analytics.RejectedQuarantined,
		WeightedScore:       analytics.WeightedScore,
	}
	logger.Info("vote analytics request completed",
		"event", "voting_http_analytics_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"total_votes", response.TotalVotes,
		"active_votes", response.ActiveVotes,
		"retracted_votes", response.RetractedVotes,
	)
	return response, nil
}

func (h Handler) QuarantineActionHandler(
	ctx context.Context,
	quarantineID string,
	action string,
	userID string,
	idempotencyKey string,
) error {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("quarantine action request received",
		"event", "voting_http_quarantine_action_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"quarantine_id", strings.TrimSpace(quarantineID),
		"action", strings.TrimSpace(action),
		"actor_id", strings.TrimSpace(userID),
	)
	if err := h.Votes.ApplyQuarantineAction(ctx, commands.QuarantineActionCommand{
		QuarantineID:   quarantineID,
		Action:         action,
		ActorID:        userID,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		logger.Error("quarantine action request failed",
			"event", "voting_http_quarantine_action_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "adapter",
			"quarantine_id", strings.TrimSpace(quarantineID),
			"action", strings.TrimSpace(action),
			"actor_id", strings.TrimSpace(userID),
			"error", err.Error(),
		)
		return err
	}
	logger.Info("quarantine action request completed",
		"event", "voting_http_quarantine_action_completed",
		"module", "campaign-editorial/voting-engine",
		"layer", "adapter",
		"quarantine_id", strings.TrimSpace(quarantineID),
		"action", strings.TrimSpace(action),
		"actor_id", strings.TrimSpace(userID),
	)
	return nil
}

func mapLeaderboard(scores []entities.SubmissionScore) []httptransport.LeaderboardItem {
	items := make([]httptransport.LeaderboardItem, 0, len(scores))
	for idx, score := range scores {
		items = append(items, httptransport.LeaderboardItem{
			SubmissionID: score.SubmissionID,
			CampaignID:   score.CampaignID,
			RoundID:      score.RoundID,
			Weighted:     score.Weighted,
			Upvotes:      score.Upvotes,
			Downvotes:    score.Downvotes,
			Rank:         idx + 1,
		})
	}
	return items
}
