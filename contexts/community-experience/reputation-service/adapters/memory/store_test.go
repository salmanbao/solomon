package memory

import (
	"context"
	"errors"
	"testing"

	domainerrors "solomon/contexts/community-experience/reputation-service/domain/errors"
	"solomon/contexts/community-experience/reputation-service/ports"
)

func TestGetUserReputationReturnsSeedData(t *testing.T) {
	store := NewStore()

	item, err := store.GetUserReputation(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if item.UserID != "user_123" {
		t.Fatalf("expected user_123, got %s", item.UserID)
	}
	if item.Tier != ports.TierGold {
		t.Fatalf("expected gold tier, got %s", item.Tier)
	}
	if item.ReputationScore <= 0 {
		t.Fatalf("expected positive score, got %d", item.ReputationScore)
	}
}

func TestGetUserReputationRequiresDependencyProjection(t *testing.T) {
	store := NewStore()

	store.mu.Lock()
	delete(store.profiles, "user_123")
	store.mu.Unlock()

	_, err := store.GetUserReputation(context.Background(), "user_123")
	if !errors.Is(err, domainerrors.ErrDependencyUnavailable) {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}
}

func TestGetLeaderboardFiltersByTier(t *testing.T) {
	store := NewStore()

	board, err := store.GetLeaderboard(context.Background(), ports.LeaderboardFilter{
		Tier:         ports.TierGold,
		Limit:        50,
		Offset:       0,
		ViewerUserID: "user_123",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if board.TotalCreators != 1 {
		t.Fatalf("expected 1 creator in gold tier, got %d", board.TotalCreators)
	}
	if len(board.Entries) != 1 {
		t.Fatalf("expected 1 leaderboard entry, got %d", len(board.Entries))
	}
	if board.Entries[0].UserID != "user_123" {
		t.Fatalf("expected user_123 leaderboard entry, got %s", board.Entries[0].UserID)
	}
	if board.YourRank != 1 {
		t.Fatalf("expected viewer rank 1, got %d", board.YourRank)
	}
}
