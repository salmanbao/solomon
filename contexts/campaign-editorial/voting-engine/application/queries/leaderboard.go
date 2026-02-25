package queries

import (
	"context"
	"sort"
	"strings"

	"solomon/contexts/campaign-editorial/voting-engine/domain/entities"
	"solomon/contexts/campaign-editorial/voting-engine/ports"
)

type LeaderboardUseCase struct {
	Votes ports.VoteRepository
}

func (uc LeaderboardUseCase) SubmissionVotes(ctx context.Context, submissionID string) (entities.SubmissionScore, error) {
	return scoreSubmission(ctx, uc.Votes, strings.TrimSpace(submissionID))
}

func (uc LeaderboardUseCase) CampaignLeaderboard(ctx context.Context, campaignID string) ([]entities.SubmissionScore, error) {
	votes, err := uc.Votes.ListVotesByCampaign(ctx, strings.TrimSpace(campaignID))
	if err != nil {
		return nil, err
	}
	scores := aggregate(votes)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Weighted == scores[j].Weighted {
			return scores[i].SubmissionID < scores[j].SubmissionID
		}
		return scores[i].Weighted > scores[j].Weighted
	})
	return scores, nil
}

func (uc LeaderboardUseCase) GlobalTrending(ctx context.Context) ([]entities.SubmissionScore, error) {
	votes, err := uc.Votes.ListVotesByCampaign(ctx, "")
	if err != nil {
		return nil, err
	}
	scores := aggregate(votes)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Weighted == scores[j].Weighted {
			return scores[i].SubmissionID < scores[j].SubmissionID
		}
		return scores[i].Weighted > scores[j].Weighted
	})
	return scores, nil
}

func scoreSubmission(ctx context.Context, repo ports.VoteRepository, submissionID string) (entities.SubmissionScore, error) {
	votes, err := repo.ListVotesBySubmission(ctx, submissionID)
	if err != nil {
		return entities.SubmissionScore{}, err
	}
	score := entities.SubmissionScore{
		SubmissionID: submissionID,
	}
	for _, vote := range votes {
		score.CampaignID = vote.CampaignID
		if vote.Retracted {
			continue
		}
		if vote.VoteType == entities.VoteTypeUpvote {
			score.Upvotes++
		} else if vote.VoteType == entities.VoteTypeDownvote {
			score.Downvotes++
		}
		score.Weighted += vote.EffectiveScore()
	}
	return score, nil
}

func aggregate(votes []entities.Vote) []entities.SubmissionScore {
	bySubmission := make(map[string]entities.SubmissionScore)
	for _, vote := range votes {
		current := bySubmission[vote.SubmissionID]
		current.SubmissionID = vote.SubmissionID
		current.CampaignID = vote.CampaignID
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
