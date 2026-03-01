package unit

import (
	"context"
	"errors"
	"testing"

	chatservice "solomon/contexts/community-experience/chat-service"
	domainerrors "solomon/contexts/community-experience/chat-service/domain/errors"
	httptransport "solomon/contexts/community-experience/chat-service/transport/http"
)

func TestChatServicePostMessageIdempotency(t *testing.T) {
	module := chatservice.NewInMemoryModule(nil)
	ctx := context.Background()

	first, err := module.Handler.PostMessageHandler(ctx, "user_123", "user_123", "idem-chat-post-1", httptransport.PostMessageRequest{
		ServerID:  "srv_001",
		ChannelID: "ch_001",
		Content:   "hello @alex https://example.com",
	})
	if err != nil {
		t.Fatalf("first post message failed: %v", err)
	}
	second, err := module.Handler.PostMessageHandler(ctx, "user_123", "user_123", "idem-chat-post-1", httptransport.PostMessageRequest{
		ServerID:  "srv_001",
		ChannelID: "ch_001",
		Content:   "hello @alex https://example.com",
	})
	if err != nil {
		t.Fatalf("replayed post message failed: %v", err)
	}
	if first.Data.Message.MessageID != second.Data.Message.MessageID {
		t.Fatalf("expected idempotent replay to return same message id, got %s and %s", first.Data.Message.MessageID, second.Data.Message.MessageID)
	}

	_, err = module.Handler.PostMessageHandler(ctx, "user_123", "user_123", "idem-chat-post-1", httptransport.PostMessageRequest{
		ServerID:  "srv_001",
		ChannelID: "ch_001",
		Content:   "different payload",
	})
	if !errors.Is(err, domainerrors.ErrIdempotencyConflict) {
		t.Fatalf("expected idempotency conflict, got %v", err)
	}
}

func TestChatServiceMessageSearchAndUnreadCount(t *testing.T) {
	module := chatservice.NewInMemoryModule(nil)
	ctx := context.Background()

	firstPost, err := module.Handler.PostMessageHandler(ctx, "user_500", "user_500", "idem-chat-post-2", httptransport.PostMessageRequest{
		ServerID:  "srv_001",
		ChannelID: "ch_001",
		Content:   "searchable keyword alpha",
	})
	if err != nil {
		t.Fatalf("post message failed: %v", err)
	}
	_, err = module.Handler.PostMessageHandler(ctx, "user_500", "user_500", "idem-chat-post-3", httptransport.PostMessageRequest{
		ServerID:  "srv_001",
		ChannelID: "ch_001",
		Content:   "searchable keyword beta",
	})
	if err != nil {
		t.Fatalf("post message failed: %v", err)
	}

	search, err := module.Handler.SearchMessagesHandler(ctx, "keyword", "ch_001", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(search.Data.Results) < 2 {
		t.Fatalf("expected >=2 search results, got %d", len(search.Data.Results))
	}

	firstID := firstPost.Data.Message.MessageID
	if _, err := module.Handler.MarkReadHandler(ctx, "user_500", httptransport.MarkReadRequest{
		ChannelID: "ch_001",
		MessageID: firstID,
	}); err != nil {
		t.Fatalf("mark read failed: %v", err)
	}

	unread, err := module.Handler.UnreadCountHandler(ctx, "user_500", "ch_001", firstID)
	if err != nil {
		t.Fatalf("unread count failed: %v", err)
	}
	if unread.Data.UnreadCount < 1 {
		t.Fatalf("expected unread count >=1, got %d", unread.Data.UnreadCount)
	}
}
