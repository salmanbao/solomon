package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	domainerrors "solomon/contexts/community-experience/reputation-service/domain/errors"
	"solomon/contexts/community-experience/reputation-service/ports"
)

type profileProjection struct {
	Username string
}

type socialVerificationProjection struct {
	Connected bool
	Verified  bool
}

type Store struct {
	mu sync.RWMutex

	scores map[string]ports.UserReputation

	// DBR:M01-Authentication-Service owner_api read-only projection.
	authUsers map[string]struct{}
	// DBR:M02-Profile-Service owner_api read-only projection.
	profiles map[string]profileProjection
	// DBR:M10-Social-Integration-Verification-Service owner_api read-only projection.
	socialAccounts map[string]socialVerificationProjection
}

func NewStore() *Store {
	now := time.Now().UTC()
	store := &Store{
		scores:         make(map[string]ports.UserReputation),
		authUsers:      make(map[string]struct{}),
		profiles:       make(map[string]profileProjection),
		socialAccounts: make(map[string]socialVerificationProjection),
	}

	for _, userID := range []string{"user_123", "user_456", "user_789", "user_999"} {
		store.authUsers[userID] = struct{}{}
	}
	store.profiles["user_123"] = profileProjection{Username: "janeclips"}
	store.profiles["user_456"] = profileProjection{Username: "creator_jane"}
	store.profiles["user_789"] = profileProjection{Username: "steady_creator"}
	store.profiles["user_999"] = profileProjection{Username: "new_creator"}

	store.socialAccounts["user_123"] = socialVerificationProjection{Connected: true, Verified: true}
	store.socialAccounts["user_456"] = socialVerificationProjection{Connected: true, Verified: true}
	store.socialAccounts["user_789"] = socialVerificationProjection{Connected: true, Verified: false}
	store.socialAccounts["user_999"] = socialVerificationProjection{Connected: false, Verified: false}

	store.scores["user_456"] = buildSeedReputation(
		"user_456",
		95,
		ports.TierPlatinum,
		93,
		8,
		12,
		"improving",
		now.Add(-2*time.Hour),
	)
	store.scores["user_123"] = buildSeedReputation(
		"user_123",
		78,
		ports.TierGold,
		75,
		3,
		-5,
		"improving",
		now.Add(-2*time.Hour),
	)
	store.scores["user_789"] = buildSeedReputation(
		"user_789",
		62,
		ports.TierSilver,
		64,
		-2,
		-4,
		"declining",
		now.Add(-2*time.Hour),
	)

	return store
}

func (s *Store) GetUserReputation(ctx context.Context, userID string) (ports.UserReputation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.ensureDependencyProjectionsLocked(userID); err != nil {
		return ports.UserReputation{}, err
	}

	record, ok := s.scores[userID]
	if !ok {
		return ports.UserReputation{}, domainerrors.ErrNotFound
	}
	return cloneUserReputation(record), nil
}

func (s *Store) GetLeaderboard(ctx context.Context, filter ports.LeaderboardFilter) (ports.Leaderboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orderedUserIDs := s.sortedUsersByScoreLocked()
	allEntries := make([]ports.LeaderboardEntry, 0, len(orderedUserIDs))

	for _, userID := range orderedUserIDs {
		record := s.scores[userID]
		if filter.Tier != "" && record.Tier != filter.Tier {
			continue
		}
		if err := s.ensureDependencyProjectionsLocked(userID); err != nil {
			return ports.Leaderboard{}, err
		}

		username := s.profiles[userID].Username
		entry := ports.LeaderboardEntry{
			Rank:     len(allEntries) + 1,
			UserID:   userID,
			Username: username,
			Tier:     record.Tier,
			Score:    record.ReputationScore,
			Badges:   badgeNames(record.Badges),
			Trend:    formatWeekTrend(record.ScoreTrend.WeekOverWeek),
		}
		allEntries = append(allEntries, entry)
	}

	totalCreators := len(allEntries)
	yourRank := 0
	if viewer := strings.TrimSpace(filter.ViewerUserID); viewer != "" {
		for _, entry := range allEntries {
			if entry.UserID == viewer {
				yourRank = entry.Rank
				break
			}
		}
	}

	if filter.Offset >= totalCreators {
		return ports.Leaderboard{
			Entries:       []ports.LeaderboardEntry{},
			TotalCreators: totalCreators,
			YourRank:      yourRank,
		}, nil
	}

	end := filter.Offset + filter.Limit
	if end > totalCreators {
		end = totalCreators
	}

	entries := make([]ports.LeaderboardEntry, 0, end-filter.Offset)
	for _, item := range allEntries[filter.Offset:end] {
		entries = append(entries, ports.LeaderboardEntry{
			Rank:     item.Rank,
			UserID:   item.UserID,
			Username: item.Username,
			Tier:     item.Tier,
			Score:    item.Score,
			Badges:   append([]string(nil), item.Badges...),
			Trend:    item.Trend,
		})
	}

	return ports.Leaderboard{
		Entries:       entries,
		TotalCreators: totalCreators,
		YourRank:      yourRank,
	}, nil
}

func (s *Store) ensureDependencyProjectionsLocked(userID string) error {
	userID = strings.TrimSpace(userID)
	if _, ok := s.authUsers[userID]; !ok {
		return domainerrors.ErrDependencyUnavailable
	}
	profile, ok := s.profiles[userID]
	if !ok || strings.TrimSpace(profile.Username) == "" {
		return domainerrors.ErrDependencyUnavailable
	}
	if _, ok := s.socialAccounts[userID]; !ok {
		return domainerrors.ErrDependencyUnavailable
	}
	return nil
}

func (s *Store) sortedUsersByScoreLocked() []string {
	type candidate struct {
		UserID string
		Score  int
	}
	candidates := make([]candidate, 0, len(s.scores))
	for userID, score := range s.scores {
		candidates = append(candidates, candidate{
			UserID: userID,
			Score:  score.ReputationScore,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].UserID < candidates[j].UserID
		}
		return candidates[i].Score > candidates[j].Score
	})

	userIDs := make([]string, 0, len(candidates))
	for _, item := range candidates {
		userIDs = append(userIDs, item.UserID)
	}
	return userIDs
}

func buildSeedReputation(
	userID string,
	score int,
	tier ports.Tier,
	previousScore int,
	weekOverWeek int,
	monthOverMonth int,
	direction string,
	calculatedAt time.Time,
) ports.UserReputation {
	return ports.UserReputation{
		UserID:          userID,
		ReputationScore: score,
		Tier:            tier,
		TierProgress: ports.TierProgress{
			CurrentPoints:    score,
			NextTierPoints:   nextTierThreshold(tier),
			PointsToNextTier: pointsToNextTier(score, tier),
		},
		PreviousScore: previousScore,
		ScoreTrend: ports.ScoreTrend{
			WeekOverWeek:   weekOverWeek,
			MonthOverMonth: monthOverMonth,
			Direction:      direction,
		},
		ScoreBreakdown: ports.ScoreBreakdown{
			ApprovalRate: ports.ScoreComponent{
				Value:        92,
				Weight:       0.20,
				Contribution: 18.4,
			},
			ViewVelocity: ports.ScoreComponent{
				Value:        15000,
				Weight:       0.15,
				Contribution: 15,
			},
			EarningsConsistency: ports.ScoreComponent{
				Value:        "stable",
				Weight:       0.15,
				Contribution: 12,
			},
			SupportSatisfaction: ports.ScoreComponent{
				Value:        8.5,
				Weight:       0.10,
				Contribution: 8.5,
			},
			ModerationRecord: ports.ScoreComponent{
				Value:        "clean",
				Weight:       0.20,
				Contribution: 20,
			},
			CommunitySentiment: ports.ScoreComponent{
				Value:        0.65,
				Weight:       0.15,
				Contribution: 9.75,
			},
		},
		Badges: []ports.Badge{
			{
				BadgeID:   "badge_top_performer",
				BadgeName: "Top Performer",
				EarnedAt:  "2026-01-15",
				Category:  "permanent",
				Rarity:    "legendary",
				IconURL:   "https://cdn.whop.com/reputation/badges/top_performer.svg",
			},
		},
		CalculatedAt:        calculatedAt.UTC(),
		NextRecalculationAt: calculatedAt.UTC().Add(24 * time.Hour),
	}
}

func nextTierThreshold(tier ports.Tier) int {
	switch tier {
	case ports.TierBronze:
		return 50
	case ports.TierSilver:
		return 75
	case ports.TierGold:
		return 90
	default:
		return 100
	}
}

func pointsToNextTier(score int, tier ports.Tier) int {
	threshold := nextTierThreshold(tier)
	if threshold <= score || threshold == 100 {
		return 0
	}
	return threshold - score
}

func badgeNames(badges []ports.Badge) []string {
	names := make([]string, 0, len(badges))
	for _, badge := range badges {
		if strings.TrimSpace(badge.BadgeID) != "" {
			names = append(names, badge.BadgeID)
		}
	}
	return names
}

func formatWeekTrend(delta int) string {
	return fmt.Sprintf("%+d vs week ago", delta)
}

func cloneUserReputation(item ports.UserReputation) ports.UserReputation {
	out := item
	out.Badges = make([]ports.Badge, 0, len(item.Badges))
	for _, badge := range item.Badges {
		out.Badges = append(out.Badges, badge)
	}
	return out
}

var _ ports.Repository = (*Store)(nil)
