package queries

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	application "solomon/contexts/campaign-editorial/voting-engine/application"
	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

// LeaderboardUseCase provides read-side projections and aggregations for votes.
type LeaderboardUseCase struct {
	Votes  ports.VoteRepository
	Clock  ports.Clock
	Logger *slog.Logger
}

// SubmissionVotes returns aggregate counters/weight for a single submission.
func (uc LeaderboardUseCase) SubmissionVotes(ctx context.Context, submissionID string) (entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	score, err := scoreSubmission(ctx, uc.Votes, strings.TrimSpace(submissionID))
	if err != nil {
		logger.Error("submission votes query failed",
			"event", "voting_submission_votes_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"submission_id", strings.TrimSpace(submissionID),
			"error", err.Error(),
		)
		return entities.SubmissionScore{}, err
	}
	return score, nil
}

// CampaignLeaderboard ranks submissions within a campaign by weighted score.
func (uc LeaderboardUseCase) CampaignLeaderboard(ctx context.Context, campaignID string) ([]entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	votes, err := uc.Votes.ListVotesByCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		logger.Error("campaign leaderboard query failed",
			"event", "voting_campaign_leaderboard_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"campaign_id", strings.TrimSpace(campaignID),
			"error", err.Error(),
		)
		return nil, err
	}
	return aggregateAndSort(votes), nil
}

// RoundLeaderboard ranks submissions scoped to one voting round.
func (uc LeaderboardUseCase) RoundLeaderboard(ctx context.Context, roundID string) ([]entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	votes, err := uc.Votes.ListVotesByRound(ctx, strings.TrimSpace(roundID))
	if err != nil {
		logger.Error("round leaderboard query failed",
			"event", "voting_round_leaderboard_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"round_id", strings.TrimSpace(roundID),
			"error", err.Error(),
		)
		return nil, err
	}
	return aggregateAndSort(votes), nil
}

// CreatorLeaderboard ranks submissions for a creator across campaigns/rounds.
func (uc LeaderboardUseCase) CreatorLeaderboard(ctx context.Context, creatorID string) ([]entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	votes, err := uc.Votes.ListVotesByCreator(ctx, strings.TrimSpace(creatorID))
	if err != nil {
		logger.Error("creator leaderboard query failed",
			"event", "voting_creator_leaderboard_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"creator_id", strings.TrimSpace(creatorID),
			"error", err.Error(),
		)
		return nil, err
	}
	return aggregateAndSort(votes), nil
}

// RoundResults pairs round metadata with the sorted round leaderboard.
func (uc LeaderboardUseCase) RoundResults(
	ctx context.Context,
	roundID string,
) (entities.VotingRound, []entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	round, err := uc.Votes.GetRound(ctx, strings.TrimSpace(roundID))
	if err != nil {
		logger.Error("round results round lookup failed",
			"event", "voting_round_results_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"round_id", strings.TrimSpace(roundID),
			"error", err.Error(),
		)
		return entities.VotingRound{}, nil, err
	}
	items, err := uc.RoundLeaderboard(ctx, roundID)
	if err != nil {
		logger.Error("round results leaderboard query failed",
			"event", "voting_round_results_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"round_id", strings.TrimSpace(roundID),
			"error", err.Error(),
		)
		return entities.VotingRound{}, nil, err
	}
	return round, items, nil
}

// GlobalTrending applies recency decay to global aggregate scores.
func (uc LeaderboardUseCase) GlobalTrending(ctx context.Context) ([]entities.SubmissionScore, error) {
	logger := application.ResolveLogger(uc.Logger)
	votes, err := uc.Votes.ListVotes(ctx)
	if err != nil {
		logger.Error("global trending query failed",
			"event", "voting_global_trending_query_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"error", err.Error(),
		)
		return nil, err
	}
	now := time.Now().UTC()
	if uc.Clock != nil {
		now = uc.Clock.Now().UTC()
	}
	scores := aggregate(votes)
	for i := range scores {
		scores[i].Weighted = trendingScore(scores[i], now)
	}
	sortScores(scores)
	return scores, nil
}

// VoteAnalytics exposes aggregate operational counters used by APIs/dashboards.
func (uc LeaderboardUseCase) VoteAnalytics(ctx context.Context) (entities.VoteAnalytics, error) {
	logger := application.ResolveLogger(uc.Logger)
	votes, err := uc.Votes.ListVotes(ctx)
	if err != nil {
		logger.Error("vote analytics votes lookup failed",
			"event", "voting_analytics_votes_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"error", err.Error(),
		)
		return entities.VoteAnalytics{}, err
	}
	quarantine, err := uc.Votes.ListQuarantines(ctx)
	if err != nil {
		logger.Error("vote analytics quarantines lookup failed",
			"event", "voting_analytics_quarantines_lookup_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "application",
			"error", err.Error(),
		)
		return entities.VoteAnalytics{}, err
	}

	analytics := entities.VoteAnalytics{
		TotalVotes: len(votes),
	}
	uniqueVoters := map[string]struct{}{}
	for _, vote := range votes {
		uniqueVoters[vote.UserID] = struct{}{}
		if vote.Retracted {
			analytics.RetractedVotes++
			continue
		}
		analytics.ActiveVotes++
		analytics.WeightedScore += vote.EffectiveScore()
	}
	analytics.UniqueVoters = len(uniqueVoters)

	for _, item := range quarantine {
		switch item.Status {
		case entities.QuarantineStatusPendingReview:
			analytics.PendingQuarantined++
		case entities.QuarantineStatusApproved:
			analytics.ApprovedQuarantined++
		case entities.QuarantineStatusRejected:
			analytics.RejectedQuarantined++
		}
	}

	logger.Info("voting analytics calculated",
		"event", "voting_analytics_calculated",
		"module", "campaign-editorial/voting-engine",
		"layer", "application",
		"total_votes", analytics.TotalVotes,
		"active_votes", analytics.ActiveVotes,
		"retracted_votes", analytics.RetractedVotes,
		"unique_voters", analytics.UniqueVoters,
	)
	return analytics, nil
}

func scoreSubmission(ctx context.Context, repo ports.VoteRepository, submissionID string) (entities.SubmissionScore, error) {
	votes, err := repo.ListVotesBySubmission(ctx, submissionID)
	if err != nil {
		return entities.SubmissionScore{}, err
	}
	scores := aggregate(votes)
	if len(scores) == 0 {
		return entities.SubmissionScore{
			SubmissionID: submissionID,
		}, nil
	}
	return scores[0], nil
}

func aggregateAndSort(votes []entities.Vote) []entities.SubmissionScore {
	scores := aggregate(votes)
	sortScores(scores)
	return scores
}

func aggregate(votes []entities.Vote) []entities.SubmissionScore {
	bySubmission := make(map[string]entities.SubmissionScore)
	for _, vote := range votes {
		current := bySubmission[vote.SubmissionID]
		current.SubmissionID = vote.SubmissionID
		current.CampaignID = vote.CampaignID
		current.RoundID = vote.RoundID
		if current.FirstVoteAt.IsZero() || vote.CreatedAt.Before(current.FirstVoteAt) {
			current.FirstVoteAt = vote.CreatedAt
		}
		if vote.UpdatedAt.After(current.LastVoteAt) {
			current.LastVoteAt = vote.UpdatedAt
		}
		if !vote.Retracted {
			if vote.VoteType == entities.VoteTypeUpvote {
				current.Upvotes++
			} else if vote.VoteType == entities.VoteTypeDownvote {
				current.Downvotes++
			}
			current.Weighted += vote.EffectiveScore()
		}
		bySubmission[vote.SubmissionID] = current
	}

	items := make([]entities.SubmissionScore, 0, len(bySubmission))
	for _, score := range bySubmission {
		items = append(items, score)
	}
	return items
}

func sortScores(scores []entities.SubmissionScore) {
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Weighted == scores[j].Weighted {
			return scores[i].FirstVoteAt.Before(scores[j].FirstVoteAt)
		}
		return scores[i].Weighted > scores[j].Weighted
	})
}

func trendingScore(score entities.SubmissionScore, now time.Time) float64 {
	weighted24h := score.Weighted
	if score.LastVoteAt.Before(now.Add(-24 * time.Hour)) {
		weighted24h = score.Weighted * 0.3
	}
	hoursSinceLastVote := now.Sub(score.LastVoteAt).Hours()
	if score.LastVoteAt.IsZero() {
		hoursSinceLastVote = 0
	}
	decay := hoursSinceLastVote * 0.1
	return (weighted24h * 0.7) + (score.Weighted * 0.3) - decay
}
