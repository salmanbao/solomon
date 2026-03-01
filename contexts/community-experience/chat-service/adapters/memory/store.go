package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/community-experience/chat-service/domain/errors"
	"solomon/contexts/community-experience/chat-service/ports"
)

type Store struct {
	mu sync.RWMutex

	messages          map[string]ports.Message
	channelMessages   map[string][]string
	channelSequences  map[string]int64
	idempotency       map[string]ports.IdempotencyRecord
	readStates        map[string]ports.ReadState
	reactions         map[string]map[string]ports.Reaction
	reactionCounters  map[string]map[string]int
	attachments       map[string]ports.Attachment
	attachmentsByMsg  map[string][]string
	pins              map[string]string
	threadLocks       map[string]bool
	serverModerators  map[string]ports.ModeratorSet
	userMutes         map[string]ports.MuteRecord
	reportedMessageID map[string]time.Time

	sequence uint64
}

func NewStore() *Store {
	now := time.Now().UTC().Add(-2 * time.Hour)
	message := ports.Message{
		MessageID:        "msg_001",
		ServerID:         "srv_001",
		ChannelID:        "ch_001",
		UserID:           "creator_001",
		Username:         "creator_001",
		Content:          "Welcome to chat",
		SequenceNumber:   1,
		CreatedAt:        now,
		UpdatedAt:        now,
		Mentions:         []ports.Mention{},
		Embeds:           []ports.Embed{},
		ReactionCounters: map[string]int{},
	}
	return &Store{
		messages: map[string]ports.Message{
			message.MessageID: message,
		},
		channelMessages: map[string][]string{
			"ch_001": {message.MessageID},
		},
		channelSequences: map[string]int64{
			"ch_001": 1,
		},
		idempotency:       make(map[string]ports.IdempotencyRecord),
		readStates:        make(map[string]ports.ReadState),
		reactions:         make(map[string]map[string]ports.Reaction),
		reactionCounters:  map[string]map[string]int{"msg_001": {}},
		attachments:       make(map[string]ports.Attachment),
		attachmentsByMsg:  make(map[string][]string),
		pins:              make(map[string]string),
		threadLocks:       make(map[string]bool),
		serverModerators:  make(map[string]ports.ModeratorSet),
		userMutes:         make(map[string]ports.MuteRecord),
		reportedMessageID: make(map[string]time.Time),
		sequence:          1,
	}
}

func (s *Store) CreateMessage(ctx context.Context, input ports.CreateMessageInput, now time.Time) (ports.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if muted, ok := s.activeMute(input.UserID, input.ServerID, now); ok && muted {
		return ports.Message{}, domainerrors.ErrForbidden
	}

	channelID := strings.TrimSpace(input.ChannelID)
	serverID := strings.TrimSpace(input.ServerID)
	if serverID == "" {
		serverID = "srv_001"
	}

	sequence := s.channelSequences[channelID] + 1
	s.channelSequences[channelID] = sequence
	messageID := "msg_" + s.nextID("m46")
	parsed := strings.TrimSpace(input.Content)

	item := ports.Message{
		MessageID:        messageID,
		ServerID:         serverID,
		ChannelID:        channelID,
		ThreadID:         strings.TrimSpace(input.ThreadID),
		UserID:           input.UserID,
		Username:         defaultString(strings.TrimSpace(input.Username), input.UserID),
		Content:          parsed,
		SequenceNumber:   sequence,
		CreatedAt:        now.UTC(),
		UpdatedAt:        now.UTC(),
		Mentions:         parseMentions(parsed),
		Embeds:           parseEmbeds(parsed),
		ReactionCounters: map[string]int{},
	}
	s.messages[item.MessageID] = item
	s.channelMessages[channelID] = append(s.channelMessages[channelID], item.MessageID)
	s.reactionCounters[item.MessageID] = map[string]int{}
	return cloneMessage(item), nil
}

func (s *Store) UpdateMessage(ctx context.Context, input ports.UpdateMessageInput, now time.Time) (ports.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.messages[input.MessageID]
	if !ok {
		return ports.Message{}, domainerrors.ErrMessageNotFound
	}
	if item.DeletedAt != nil {
		return ports.Message{}, domainerrors.ErrConflict
	}
	if item.UserID != input.UserID {
		return ports.Message{}, domainerrors.ErrForbidden
	}
	if now.UTC().After(item.CreatedAt.UTC().Add(5 * time.Minute)) {
		return ports.Message{}, domainerrors.ErrConflict
	}
	item.Content = strings.TrimSpace(input.Content)
	item.UpdatedAt = now.UTC()
	item.Edited = true
	item.Mentions = parseMentions(item.Content)
	item.Embeds = parseEmbeds(item.Content)
	s.messages[item.MessageID] = item
	return cloneMessage(item), nil
}

func (s *Store) DeleteMessage(ctx context.Context, input ports.DeleteMessageInput, now time.Time) (ports.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.messages[input.MessageID]
	if !ok {
		return ports.Message{}, domainerrors.ErrMessageNotFound
	}
	if item.DeletedAt != nil {
		return cloneMessage(item), nil
	}
	if item.UserID != input.UserID && !canModerate(input.UserID) {
		return ports.Message{}, domainerrors.ErrForbidden
	}
	ts := now.UTC()
	item.DeletedAt = &ts
	item.DeletedByUserID = input.UserID
	item.DeletionReason = strings.TrimSpace(input.Reason)
	item.Content = "[Deleted]"
	item.UpdatedAt = ts
	s.messages[item.MessageID] = item
	return cloneMessage(item), nil
}

func (s *Store) ListMessages(ctx context.Context, input ports.ListMessagesInput) ([]ports.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := append([]string(nil), s.channelMessages[input.ChannelID]...)
	if len(ids) == 0 {
		return []ports.Message{}, nil
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}

	startSeq := int64(0)
	if input.AfterSequence > 0 {
		startSeq = input.AfterSequence
	}
	if input.BeforeMessageID != "" {
		if before, ok := s.messages[input.BeforeMessageID]; ok {
			startSeq = before.SequenceNumber - int64(limit) - 1
			if startSeq < 0 {
				startSeq = 0
			}
		}
	}

	items := make([]ports.Message, 0, len(ids))
	for _, id := range ids {
		item, ok := s.messages[id]
		if !ok {
			continue
		}
		if item.SequenceNumber <= startSeq {
			continue
		}
		items = append(items, cloneMessage(item))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SequenceNumber > items[j].SequenceNumber
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) SearchMessages(ctx context.Context, input ports.SearchInput) ([]ports.SearchResult, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := strings.ToLower(strings.TrimSpace(input.Query))
	if query == "" {
		return nil, 0, domainerrors.ErrInvalidRequest
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	results := make([]ports.SearchResult, 0)
	for _, item := range s.messages {
		if input.ChannelID != "" && item.ChannelID != input.ChannelID {
			continue
		}
		if item.DeletedAt != nil {
			continue
		}
		content := strings.ToLower(item.Content)
		if !strings.Contains(content, query) {
			continue
		}
		snippet := item.Content
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		results = append(results, ports.SearchResult{
			MessageID: item.MessageID,
			ChannelID: item.ChannelID,
			Username:  item.Username,
			Content:   item.Content,
			Snippet:   snippet,
			CreatedAt: item.CreatedAt,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	total := len(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return results, total, nil
}

func (s *Store) UpsertReadState(
	ctx context.Context,
	userID string,
	channelID string,
	lastReadMessageID string,
	now time.Time,
) (ports.ReadState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := readStateKey(userID, channelID)
	item, ok := s.readStates[key]
	if !ok {
		item = ports.ReadState{
			ReadStateID: "read_" + s.nextID("m46"),
			UserID:      userID,
			ChannelID:   channelID,
		}
	}
	item.LastReadMessageID = strings.TrimSpace(lastReadMessageID)
	item.LastReadAt = now.UTC()
	s.readStates[key] = item
	return item, nil
}

func (s *Store) UnreadCount(ctx context.Context, userID string, channelID string, lastReadMessageID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if lastReadMessageID == "" {
		if state, ok := s.readStates[readStateKey(userID, channelID)]; ok {
			lastReadMessageID = state.LastReadMessageID
		}
	}
	lastSeq := int64(0)
	if lastReadMessageID != "" {
		if item, ok := s.messages[lastReadMessageID]; ok {
			lastSeq = item.SequenceNumber
		}
	}
	count := 0
	for _, id := range s.channelMessages[channelID] {
		item, ok := s.messages[id]
		if !ok || item.DeletedAt != nil {
			continue
		}
		if item.SequenceNumber > lastSeq {
			count++
		}
	}
	return count, nil
}

func (s *Store) AddReaction(
	ctx context.Context,
	messageID string,
	userID string,
	emoji string,
	now time.Time,
) (ports.Reaction, map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[messageID]; !ok {
		return ports.Reaction{}, nil, domainerrors.ErrMessageNotFound
	}
	key := userID + "|" + strings.TrimSpace(emoji)
	if _, ok := s.reactions[messageID]; !ok {
		s.reactions[messageID] = make(map[string]ports.Reaction)
	}
	if _, ok := s.reactionCounters[messageID]; !ok {
		s.reactionCounters[messageID] = make(map[string]int)
	}
	reaction, exists := s.reactions[messageID][key]
	if !exists {
		reaction = ports.Reaction{
			ReactionID: "rxn_" + s.nextID("m46"),
			MessageID:  messageID,
			UserID:     userID,
			Emoji:      strings.TrimSpace(emoji),
			CreatedAt:  now.UTC(),
		}
		s.reactions[messageID][key] = reaction
		s.reactionCounters[messageID][reaction.Emoji]++
	}
	item := s.messages[messageID]
	item.ReactionCounters = cloneCounter(s.reactionCounters[messageID])
	s.messages[messageID] = item
	return reaction, cloneCounter(s.reactionCounters[messageID]), nil
}

func (s *Store) RemoveReaction(ctx context.Context, messageID string, userID string, emoji string) (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[messageID]; !ok {
		return nil, domainerrors.ErrMessageNotFound
	}
	key := userID + "|" + strings.TrimSpace(emoji)
	if reactionsByMsg, ok := s.reactions[messageID]; ok {
		if reaction, exists := reactionsByMsg[key]; exists {
			delete(reactionsByMsg, key)
			if counters, ok := s.reactionCounters[messageID]; ok {
				if counters[reaction.Emoji] > 1 {
					counters[reaction.Emoji]--
				} else {
					delete(counters, reaction.Emoji)
				}
			}
		}
	}
	item := s.messages[messageID]
	item.ReactionCounters = cloneCounter(s.reactionCounters[messageID])
	s.messages[messageID] = item
	return cloneCounter(s.reactionCounters[messageID]), nil
}

func (s *Store) PinMessage(ctx context.Context, messageID string, actorID string, reason string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[messageID]; !ok {
		return domainerrors.ErrMessageNotFound
	}
	if !canModerate(actorID) {
		return domainerrors.ErrForbidden
	}
	s.pins[messageID] = strings.TrimSpace(reason)
	return nil
}

func (s *Store) ReportMessage(
	ctx context.Context,
	messageID string,
	actorID string,
	reason string,
	description string,
	now time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[messageID]; !ok {
		return domainerrors.ErrMessageNotFound
	}
	s.reportedMessageID[messageID+"|"+actorID+"|"+reason+"|"+description] = now.UTC()
	return nil
}

func (s *Store) LockThread(ctx context.Context, threadID string, actorID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(threadID) == "" {
		return domainerrors.ErrInvalidRequest
	}
	if !canModerate(actorID) {
		return domainerrors.ErrForbidden
	}
	s.threadLocks[threadID] = true
	return nil
}

func (s *Store) UpdateModerators(
	ctx context.Context,
	serverID string,
	actorID string,
	moderatorIDs []string,
	now time.Time,
) (ports.ModeratorSet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !canModerate(actorID) {
		return ports.ModeratorSet{}, domainerrors.ErrForbidden
	}
	set := ports.ModeratorSet{
		ServerID:     serverID,
		ModeratorIDs: dedupeList(moderatorIDs),
		UpdatedBy:    actorID,
		UpdatedAt:    now.UTC(),
	}
	s.serverModerators[serverID] = set
	return set, nil
}

func (s *Store) MuteUser(
	ctx context.Context,
	targetUserID string,
	serverID string,
	actorID string,
	duration time.Duration,
	reason string,
	now time.Time,
) (ports.MuteRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !canModerate(actorID) {
		return ports.MuteRecord{}, domainerrors.ErrForbidden
	}
	var mutedUntil *time.Time
	if duration > 0 {
		ts := now.UTC().Add(duration)
		mutedUntil = &ts
	}
	record := ports.MuteRecord{
		MuteID:        "mute_" + s.nextID("m46"),
		UserID:        targetUserID,
		ServerID:      serverID,
		MutedByUserID: actorID,
		MutedUntil:    mutedUntil,
		Reason:        strings.TrimSpace(reason),
		CreatedAt:     now.UTC(),
	}
	s.userMutes[muteKey(targetUserID, serverID)] = record
	return record, nil
}

func (s *Store) AddAttachment(
	ctx context.Context,
	messageID string,
	userID string,
	filename string,
	fileSize int64,
	mimeType string,
	now time.Time,
) (ports.Attachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.messages[messageID]; !ok {
		return ports.Attachment{}, domainerrors.ErrMessageNotFound
	}
	if fileSize > 50*1024*1024 {
		return ports.Attachment{}, domainerrors.ErrInvalidRequest
	}
	attachment := ports.Attachment{
		AttachmentID: "att_" + s.nextID("m46"),
		MessageID:    messageID,
		UserID:       userID,
		Filename:     strings.TrimSpace(filename),
		FileSize:     fileSize,
		MimeType:     strings.TrimSpace(mimeType),
		URL:          "https://cdn.viralforge.local/chat/" + messageID + "/" + strings.TrimSpace(filename),
		ScanResult:   "CLEAN",
		ScannedAt:    now.UTC(),
	}
	s.attachments[attachment.AttachmentID] = attachment
	s.attachmentsByMsg[messageID] = append(s.attachmentsByMsg[messageID], attachment.AttachmentID)
	return attachment, nil
}

func (s *Store) GetAttachment(ctx context.Context, messageID string, attachmentID string) (ports.Attachment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.attachments[attachmentID]
	if !ok || item.MessageID != messageID {
		return ports.Attachment{}, domainerrors.ErrAttachmentNotFound
	}
	return item, nil
}

func (s *Store) ExportMessages(ctx context.Context, serverID string, channelID string, limit int) ([]ports.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 1000
	}
	items := make([]ports.Message, 0)
	for _, item := range s.messages {
		if serverID != "" && item.ServerID != serverID {
			continue
		}
		if channelID != "" && item.ChannelID != channelID {
			continue
		}
		items = append(items, cloneMessage(item))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) NewID(ctx context.Context) (string, error) {
	return s.nextID("m46"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func (s *Store) activeMute(userID string, serverID string, now time.Time) (bool, bool) {
	record, ok := s.userMutes[muteKey(userID, serverID)]
	if !ok {
		return false, false
	}
	if record.MutedUntil == nil {
		return true, true
	}
	if now.UTC().Before(record.MutedUntil.UTC()) {
		return true, true
	}
	return false, true
}

func canModerate(userID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(userID))
	return strings.HasPrefix(normalized, "mod_") ||
		strings.HasPrefix(normalized, "creator_") ||
		strings.HasPrefix(normalized, "admin_")
}

func cloneMessage(in ports.Message) ports.Message {
	out := in
	out.Mentions = append([]ports.Mention(nil), in.Mentions...)
	out.Embeds = append([]ports.Embed(nil), in.Embeds...)
	out.ReactionCounters = cloneCounter(in.ReactionCounters)
	return out
}

func cloneCounter(in map[string]int) map[string]int {
	if in == nil {
		return map[string]int{}
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func parseMentions(content string) []ports.Mention {
	words := strings.Fields(content)
	out := make([]ports.Mention, 0)
	for _, word := range words {
		if len(word) < 2 || word[0] != '@' {
			continue
		}
		username := strings.Trim(word[1:], ".,!?;:")
		if username == "" {
			continue
		}
		out = append(out, ports.Mention{
			UserID:   username,
			Username: username,
		})
	}
	return out
}

func parseEmbeds(content string) []ports.Embed {
	words := strings.Fields(content)
	out := make([]ports.Embed, 0)
	for _, word := range words {
		candidate := strings.Trim(word, ".,!?;:")
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			out = append(out, ports.Embed{
				URL:   candidate,
				Title: "Link Preview",
			})
		}
	}
	return out
}

func dedupeList(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func readStateKey(userID string, channelID string) string {
	return userID + "|" + channelID
}

func muteKey(userID string, serverID string) string {
	return userID + "|" + serverID
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
