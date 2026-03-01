package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/community-experience/chat-service/domain/errors"
	"solomon/contexts/community-experience/chat-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

func (s Service) PostMessage(
	ctx context.Context,
	idempotencyKey string,
	input ports.CreateMessageInput,
) (ports.Message, error) {
	var out ports.Message
	if strings.TrimSpace(input.ChannelID) == "" || strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.Content) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("post_message", string(payload))
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.CreateMessage(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) EditMessage(
	ctx context.Context,
	idempotencyKey string,
	input ports.UpdateMessageInput,
) (ports.Message, error) {
	var out ports.Message
	if strings.TrimSpace(input.MessageID) == "" || strings.TrimSpace(input.UserID) == "" || strings.TrimSpace(input.Content) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("edit_message", string(payload))
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.UpdateMessage(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) DeleteMessage(
	ctx context.Context,
	idempotencyKey string,
	input ports.DeleteMessageInput,
) (ports.Message, error) {
	var out ports.Message
	if strings.TrimSpace(input.MessageID) == "" || strings.TrimSpace(input.UserID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("delete_message", string(payload))
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.DeleteMessage(ctx, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) ListMessages(ctx context.Context, input ports.ListMessagesInput) ([]ports.Message, error) {
	if strings.TrimSpace(input.ChannelID) == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}
	if input.Limit > 200 {
		input.Limit = 200
	}
	return s.Repo.ListMessages(ctx, input)
}

func (s Service) SearchMessages(ctx context.Context, input ports.SearchInput) ([]ports.SearchResult, int, error) {
	if strings.TrimSpace(input.Query) == "" {
		return nil, 0, domainerrors.ErrInvalidRequest
	}
	if input.Limit <= 0 {
		input.Limit = 10
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	return s.Repo.SearchMessages(ctx, input)
}

func (s Service) MarkRead(ctx context.Context, userID string, channelID string, messageID string) (ports.ReadState, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(channelID) == "" {
		return ports.ReadState{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.UpsertReadState(ctx, userID, channelID, strings.TrimSpace(messageID), s.now())
}

func (s Service) UnreadCount(ctx context.Context, userID string, channelID string, lastReadMessageID string) (int, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(channelID) == "" {
		return 0, domainerrors.ErrInvalidRequest
	}
	return s.Repo.UnreadCount(ctx, userID, channelID, strings.TrimSpace(lastReadMessageID))
}

func (s Service) AddReaction(
	ctx context.Context,
	idempotencyKey string,
	messageID string,
	userID string,
	emoji string,
) (ports.Reaction, map[string]int, error) {
	var reaction ports.Reaction
	var counters map[string]int
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(emoji) == "" {
		return reaction, nil, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return reaction, nil, err
	}
	requestHash := hashStrings("add_reaction", messageID, userID, emoji)
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error {
			var payload struct {
				Reaction ports.Reaction
				Counts   map[string]int
			}
			if err := json.Unmarshal(raw, &payload); err != nil {
				return err
			}
			reaction = payload.Reaction
			counters = payload.Counts
			return nil
		},
		func() ([]byte, error) {
			r, c, err := s.Repo.AddReaction(ctx, messageID, userID, emoji, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(struct {
				Reaction ports.Reaction
				Counts   map[string]int
			}{Reaction: r, Counts: c})
		},
	)
	return reaction, counters, err
}

func (s Service) RemoveReaction(
	ctx context.Context,
	idempotencyKey string,
	messageID string,
	userID string,
	emoji string,
) (map[string]int, error) {
	var counters map[string]int
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(emoji) == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return nil, err
	}
	requestHash := hashStrings("remove_reaction", messageID, userID, emoji)
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error {
			return json.Unmarshal(raw, &counters)
		},
		func() ([]byte, error) {
			result, err := s.Repo.RemoveReaction(ctx, messageID, userID, emoji)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return counters, err
}

func (s Service) PinMessage(ctx context.Context, idempotencyKey string, messageID string, actorID string, reason string) error {
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(actorID) == "" {
		return domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return err
	}
	requestHash := hashStrings("pin_message", messageID, actorID, reason)
	return s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return nil },
		func() ([]byte, error) {
			if err := s.Repo.PinMessage(ctx, messageID, actorID, reason, s.now()); err != nil {
				return nil, err
			}
			return []byte(`{}`), nil
		},
	)
}

func (s Service) ReportMessage(ctx context.Context, idempotencyKey string, messageID string, actorID string, reason string, description string) error {
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(actorID) == "" || strings.TrimSpace(reason) == "" {
		return domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return err
	}
	requestHash := hashStrings("report_message", messageID, actorID, reason, description)
	return s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return nil },
		func() ([]byte, error) {
			if err := s.Repo.ReportMessage(ctx, messageID, actorID, reason, description, s.now()); err != nil {
				return nil, err
			}
			return []byte(`{}`), nil
		},
	)
}

func (s Service) LockThread(ctx context.Context, idempotencyKey string, threadID string, actorID string) error {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(actorID) == "" {
		return domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return err
	}
	requestHash := hashStrings("lock_thread", threadID, actorID)
	return s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return nil },
		func() ([]byte, error) {
			if err := s.Repo.LockThread(ctx, threadID, actorID, s.now()); err != nil {
				return nil, err
			}
			return []byte(`{}`), nil
		},
	)
}

func (s Service) UpdateModerators(
	ctx context.Context,
	idempotencyKey string,
	serverID string,
	actorID string,
	moderatorIDs []string,
) (ports.ModeratorSet, error) {
	var out ports.ModeratorSet
	if strings.TrimSpace(serverID) == "" || strings.TrimSpace(actorID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(moderatorIDs)
	requestHash := hashStrings("update_moderators", serverID, actorID, string(payload))
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.UpdateModerators(ctx, serverID, actorID, moderatorIDs, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) MuteUser(
	ctx context.Context,
	idempotencyKey string,
	targetUserID string,
	serverID string,
	actorID string,
	duration time.Duration,
	reason string,
) (ports.MuteRecord, error) {
	var out ports.MuteRecord
	if strings.TrimSpace(targetUserID) == "" || strings.TrimSpace(serverID) == "" || strings.TrimSpace(actorID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("mute_user", targetUserID, serverID, actorID, duration.String(), reason)
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.MuteUser(ctx, targetUserID, serverID, actorID, duration, reason, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) AddAttachment(
	ctx context.Context,
	idempotencyKey string,
	messageID string,
	userID string,
	filename string,
	fileSize int64,
	mimeType string,
) (ports.Attachment, error) {
	var out ports.Attachment
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(filename) == "" || fileSize <= 0 {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("add_attachment", messageID, userID, filename, mimeType)
	err := s.runIdempotent(
		ctx,
		idempotencyKey,
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			result, err := s.Repo.AddAttachment(ctx, messageID, userID, filename, fileSize, mimeType, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
	)
	return out, err
}

func (s Service) GetAttachment(ctx context.Context, messageID string, attachmentID string) (ports.Attachment, error) {
	if strings.TrimSpace(messageID) == "" || strings.TrimSpace(attachmentID) == "" {
		return ports.Attachment{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetAttachment(ctx, messageID, attachmentID)
}

func (s Service) ExportMessages(ctx context.Context, serverID string, channelID string, limit int) ([]ports.Message, error) {
	if strings.TrimSpace(serverID) == "" && strings.TrimSpace(channelID) == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}
	return s.Repo.ExportMessages(ctx, serverID, channelID, limit)
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) requireIdempotency(key string) error {
	if strings.TrimSpace(key) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	return nil
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("chat service idempotent operation committed",
		"event", "chat_service_idempotent_operation_committed",
		"module", "community-experience/chat-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
