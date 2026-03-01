package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MentionDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type EmbedDTO struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

type MessageDTO struct {
	MessageID      string         `json:"message_id"`
	ServerID       string         `json:"server_id,omitempty"`
	ChannelID      string         `json:"channel_id"`
	ThreadID       string         `json:"thread_id,omitempty"`
	UserID         string         `json:"user_id"`
	Username       string         `json:"username,omitempty"`
	Content        string         `json:"content"`
	SequenceNumber int64          `json:"sequence_number,omitempty"`
	Mentions       []MentionDTO   `json:"mentions,omitempty"`
	Embeds         []EmbedDTO     `json:"embeds,omitempty"`
	Reactions      map[string]int `json:"reactions,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at,omitempty"`
	Edited         bool           `json:"edited"`
	DeletedAt      string         `json:"deleted_at,omitempty"`
}

type PostMessageRequest struct {
	ServerID  string `json:"server_id,omitempty"`
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id,omitempty"`
	Content   string `json:"content"`
}

type PostMessageResponse struct {
	Status string `json:"status"`
	Data   struct {
		Message MessageDTO `json:"message"`
	} `json:"data"`
}

type EditMessageRequest struct {
	Content string `json:"content"`
}

type EditMessageResponse struct {
	Status string `json:"status"`
	Data   struct {
		Message MessageDTO `json:"message"`
	} `json:"data"`
}

type DeleteMessageRequest struct {
	Reason string `json:"reason,omitempty"`
}

type DeleteMessageResponse struct {
	Status string `json:"status"`
	Data   struct {
		MessageID string `json:"message_id"`
		DeletedAt string `json:"deleted_at"`
	} `json:"data"`
}

type ListMessagesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Messages []MessageDTO `json:"messages"`
		Limit    int          `json:"limit"`
	} `json:"data"`
}

type SearchMessagesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Results []struct {
			MessageID string `json:"message_id"`
			ChannelID string `json:"channel_id"`
			Username  string `json:"username"`
			Content   string `json:"content"`
			Snippet   string `json:"snippet"`
			CreatedAt string `json:"created_at"`
		} `json:"results"`
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	} `json:"data"`
}

type MarkReadRequest struct {
	MessageID string `json:"message_id"`
	ChannelID string `json:"channel_id"`
}

type MarkReadResponse struct {
	Status string `json:"status"`
	Data   struct {
		ChannelID         string `json:"channel_id"`
		LastReadMessageID string `json:"last_read_message_id,omitempty"`
		LastReadAt        string `json:"last_read_at"`
	} `json:"data"`
}

type UnreadCountResponse struct {
	Status string `json:"status"`
	Data   struct {
		ChannelID   string `json:"channel_id"`
		UnreadCount int    `json:"unread_count"`
	} `json:"data"`
}

type ReactionRequest struct {
	Emoji string `json:"emoji"`
}

type ReactionResponse struct {
	Status string `json:"status"`
	Data   struct {
		MessageID string         `json:"message_id"`
		Reactions map[string]int `json:"reactions"`
	} `json:"data"`
}

type PinMessageRequest struct {
	Reason string `json:"reason,omitempty"`
}

type ReportMessageRequest struct {
	Reason      string `json:"reason"`
	Description string `json:"description,omitempty"`
}

type GenericOKResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type MuteUserRequest struct {
	ServerID string `json:"server_id"`
	Duration string `json:"duration,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type UpdateModeratorsRequest struct {
	ModeratorIDs []string `json:"moderator_ids"`
}

type AddAttachmentRequest struct {
	Filename string `json:"filename"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

type AddAttachmentResponse struct {
	Status string `json:"status"`
	Data   struct {
		AttachmentID string `json:"attachment_id"`
		MessageID    string `json:"message_id"`
		Filename     string `json:"filename"`
		FileSize     int64  `json:"file_size"`
		MimeType     string `json:"mime_type"`
		URL          string `json:"url"`
		ScanResult   string `json:"scan_result"`
		ScannedAt    string `json:"scanned_at"`
	} `json:"data"`
}

type GetAttachmentResponse struct {
	Status string `json:"status"`
	Data   struct {
		AttachmentID string `json:"attachment_id"`
		MessageID    string `json:"message_id"`
		Filename     string `json:"filename"`
		FileSize     int64  `json:"file_size"`
		MimeType     string `json:"mime_type"`
		URL          string `json:"url"`
	} `json:"data"`
}

type ExportMessagesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Count    int          `json:"count"`
		Messages []MessageDTO `json:"messages"`
	} `json:"data"`
}
