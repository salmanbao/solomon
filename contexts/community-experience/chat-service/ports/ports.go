package ports

import (
	"context"
	"time"
)

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type Mention struct {
	UserID   string
	Username string
}

type Embed struct {
	URL         string
	Title       string
	Description string
	ImageURL    string
}

type Message struct {
	MessageID        string
	ServerID         string
	ChannelID        string
	ThreadID         string
	UserID           string
	Username         string
	Content          string
	SequenceNumber   int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Edited           bool
	DeletedAt        *time.Time
	DeletedByUserID  string
	DeletionReason   string
	Mentions         []Mention
	Embeds           []Embed
	ReactionCounters map[string]int
}

type SearchResult struct {
	MessageID string
	ChannelID string
	Username  string
	Content   string
	Snippet   string
	CreatedAt time.Time
}

type ReadState struct {
	ReadStateID       string
	UserID            string
	ChannelID         string
	LastReadMessageID string
	LastReadAt        time.Time
}

type Reaction struct {
	ReactionID string
	MessageID  string
	UserID     string
	Emoji      string
	CreatedAt  time.Time
}

type Attachment struct {
	AttachmentID string
	MessageID    string
	UserID       string
	Filename     string
	FileSize     int64
	MimeType     string
	URL          string
	ScanResult   string
	ScannedAt    time.Time
}

type MuteRecord struct {
	MuteID        string
	UserID        string
	ServerID      string
	MutedByUserID string
	MutedUntil    *time.Time
	Reason        string
	CreatedAt     time.Time
}

type ModeratorSet struct {
	ServerID     string
	ModeratorIDs []string
	UpdatedBy    string
	UpdatedAt    time.Time
}

type CreateMessageInput struct {
	ServerID  string
	ChannelID string
	ThreadID  string
	UserID    string
	Username  string
	Content   string
}

type UpdateMessageInput struct {
	MessageID string
	UserID    string
	Content   string
}

type DeleteMessageInput struct {
	MessageID string
	UserID    string
	Reason    string
}

type ListMessagesInput struct {
	ChannelID       string
	BeforeMessageID string
	AfterSequence   int64
	Limit           int
}

type SearchInput struct {
	Query     string
	ChannelID string
	Limit     int
}

type Repository interface {
	CreateMessage(ctx context.Context, input CreateMessageInput, now time.Time) (Message, error)
	UpdateMessage(ctx context.Context, input UpdateMessageInput, now time.Time) (Message, error)
	DeleteMessage(ctx context.Context, input DeleteMessageInput, now time.Time) (Message, error)
	ListMessages(ctx context.Context, input ListMessagesInput) ([]Message, error)
	SearchMessages(ctx context.Context, input SearchInput) ([]SearchResult, int, error)
	UpsertReadState(ctx context.Context, userID string, channelID string, lastReadMessageID string, now time.Time) (ReadState, error)
	UnreadCount(ctx context.Context, userID string, channelID string, lastReadMessageID string) (int, error)
	AddReaction(ctx context.Context, messageID string, userID string, emoji string, now time.Time) (Reaction, map[string]int, error)
	RemoveReaction(ctx context.Context, messageID string, userID string, emoji string) (map[string]int, error)
	PinMessage(ctx context.Context, messageID string, actorID string, reason string, now time.Time) error
	ReportMessage(ctx context.Context, messageID string, actorID string, reason string, description string, now time.Time) error
	LockThread(ctx context.Context, threadID string, actorID string, now time.Time) error
	UpdateModerators(ctx context.Context, serverID string, actorID string, moderatorIDs []string, now time.Time) (ModeratorSet, error)
	MuteUser(ctx context.Context, targetUserID string, serverID string, actorID string, duration time.Duration, reason string, now time.Time) (MuteRecord, error)
	AddAttachment(ctx context.Context, messageID string, userID string, filename string, fileSize int64, mimeType string, now time.Time) (Attachment, error)
	GetAttachment(ctx context.Context, messageID string, attachmentID string) (Attachment, error)
	ExportMessages(ctx context.Context, serverID string, channelID string, limit int) ([]Message, error)
}
