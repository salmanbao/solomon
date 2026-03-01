package httpadapter

import (
	"context"
	"log/slog"
	"time"

	"solomon/contexts/community-experience/chat-service/application"
	"solomon/contexts/community-experience/chat-service/ports"
	httptransport "solomon/contexts/community-experience/chat-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) PostMessageHandler(
	ctx context.Context,
	userID string,
	username string,
	idempotencyKey string,
	req httptransport.PostMessageRequest,
) (httptransport.PostMessageResponse, error) {
	item, err := h.Service.PostMessage(ctx, idempotencyKey, ports.CreateMessageInput{
		ServerID:  req.ServerID,
		ChannelID: req.ChannelID,
		ThreadID:  req.ThreadID,
		UserID:    userID,
		Username:  username,
		Content:   req.Content,
	})
	if err != nil {
		return httptransport.PostMessageResponse{}, err
	}
	resp := httptransport.PostMessageResponse{Status: "success"}
	resp.Data.Message = toMessageDTO(item)
	return resp, nil
}

func (h Handler) EditMessageHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.EditMessageRequest,
) (httptransport.EditMessageResponse, error) {
	item, err := h.Service.EditMessage(ctx, idempotencyKey, ports.UpdateMessageInput{
		MessageID: messageID,
		UserID:    userID,
		Content:   req.Content,
	})
	if err != nil {
		return httptransport.EditMessageResponse{}, err
	}
	resp := httptransport.EditMessageResponse{Status: "success"}
	resp.Data.Message = toMessageDTO(item)
	return resp, nil
}

func (h Handler) DeleteMessageHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.DeleteMessageRequest,
) (httptransport.DeleteMessageResponse, error) {
	item, err := h.Service.DeleteMessage(ctx, idempotencyKey, ports.DeleteMessageInput{
		MessageID: messageID,
		UserID:    userID,
		Reason:    req.Reason,
	})
	if err != nil {
		return httptransport.DeleteMessageResponse{}, err
	}
	resp := httptransport.DeleteMessageResponse{Status: "success"}
	resp.Data.MessageID = item.MessageID
	if item.DeletedAt != nil {
		resp.Data.DeletedAt = item.DeletedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) ListMessagesHandler(
	ctx context.Context,
	channelID string,
	beforeMessageID string,
	afterSequence int64,
	limit int,
) (httptransport.ListMessagesResponse, error) {
	items, err := h.Service.ListMessages(ctx, ports.ListMessagesInput{
		ChannelID:       channelID,
		BeforeMessageID: beforeMessageID,
		AfterSequence:   afterSequence,
		Limit:           limit,
	})
	if err != nil {
		return httptransport.ListMessagesResponse{}, err
	}
	resp := httptransport.ListMessagesResponse{Status: "success"}
	resp.Data.Messages = make([]httptransport.MessageDTO, 0, len(items))
	for _, item := range items {
		resp.Data.Messages = append(resp.Data.Messages, toMessageDTO(item))
	}
	resp.Data.Limit = limit
	if resp.Data.Limit <= 0 {
		resp.Data.Limit = 50
	}
	return resp, nil
}

func (h Handler) SearchMessagesHandler(
	ctx context.Context,
	query string,
	channelID string,
	limit int,
) (httptransport.SearchMessagesResponse, error) {
	results, total, err := h.Service.SearchMessages(ctx, ports.SearchInput{
		Query:     query,
		ChannelID: channelID,
		Limit:     limit,
	})
	if err != nil {
		return httptransport.SearchMessagesResponse{}, err
	}
	resp := httptransport.SearchMessagesResponse{Status: "success"}
	resp.Data.Results = make([]struct {
		MessageID string `json:"message_id"`
		ChannelID string `json:"channel_id"`
		Username  string `json:"username"`
		Content   string `json:"content"`
		Snippet   string `json:"snippet"`
		CreatedAt string `json:"created_at"`
	}, 0, len(results))
	for _, item := range results {
		resp.Data.Results = append(resp.Data.Results, struct {
			MessageID string `json:"message_id"`
			ChannelID string `json:"channel_id"`
			Username  string `json:"username"`
			Content   string `json:"content"`
			Snippet   string `json:"snippet"`
			CreatedAt string `json:"created_at"`
		}{
			MessageID: item.MessageID,
			ChannelID: item.ChannelID,
			Username:  item.Username,
			Content:   item.Content,
			Snippet:   item.Snippet,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	resp.Data.Total = total
	resp.Data.Page = 1
	resp.Data.Limit = limit
	if resp.Data.Limit <= 0 {
		resp.Data.Limit = 10
	}
	return resp, nil
}

func (h Handler) MarkReadHandler(
	ctx context.Context,
	userID string,
	req httptransport.MarkReadRequest,
) (httptransport.MarkReadResponse, error) {
	state, err := h.Service.MarkRead(ctx, userID, req.ChannelID, req.MessageID)
	if err != nil {
		return httptransport.MarkReadResponse{}, err
	}
	resp := httptransport.MarkReadResponse{Status: "success"}
	resp.Data.ChannelID = state.ChannelID
	resp.Data.LastReadMessageID = state.LastReadMessageID
	resp.Data.LastReadAt = state.LastReadAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) UnreadCountHandler(
	ctx context.Context,
	userID string,
	channelID string,
	lastReadMessageID string,
) (httptransport.UnreadCountResponse, error) {
	count, err := h.Service.UnreadCount(ctx, userID, channelID, lastReadMessageID)
	if err != nil {
		return httptransport.UnreadCountResponse{}, err
	}
	resp := httptransport.UnreadCountResponse{Status: "success"}
	resp.Data.ChannelID = channelID
	resp.Data.UnreadCount = count
	return resp, nil
}

func (h Handler) AddReactionHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.ReactionRequest,
) (httptransport.ReactionResponse, error) {
	_, counts, err := h.Service.AddReaction(ctx, idempotencyKey, messageID, userID, req.Emoji)
	if err != nil {
		return httptransport.ReactionResponse{}, err
	}
	resp := httptransport.ReactionResponse{Status: "success"}
	resp.Data.MessageID = messageID
	resp.Data.Reactions = counts
	return resp, nil
}

func (h Handler) RemoveReactionHandler(
	ctx context.Context,
	userID string,
	messageID string,
	emoji string,
	idempotencyKey string,
) (httptransport.ReactionResponse, error) {
	counts, err := h.Service.RemoveReaction(ctx, idempotencyKey, messageID, userID, emoji)
	if err != nil {
		return httptransport.ReactionResponse{}, err
	}
	resp := httptransport.ReactionResponse{Status: "success"}
	resp.Data.MessageID = messageID
	resp.Data.Reactions = counts
	return resp, nil
}

func (h Handler) PinMessageHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.PinMessageRequest,
) (httptransport.GenericOKResponse, error) {
	if err := h.Service.PinMessage(ctx, idempotencyKey, messageID, userID, req.Reason); err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	return httptransport.GenericOKResponse{Status: "success", Data: map[string]string{"message_id": messageID}}, nil
}

func (h Handler) ReportMessageHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.ReportMessageRequest,
) (httptransport.GenericOKResponse, error) {
	if err := h.Service.ReportMessage(ctx, idempotencyKey, messageID, userID, req.Reason, req.Description); err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	return httptransport.GenericOKResponse{Status: "success", Data: map[string]string{"message_id": messageID}}, nil
}

func (h Handler) LockThreadHandler(
	ctx context.Context,
	userID string,
	threadID string,
	idempotencyKey string,
) (httptransport.GenericOKResponse, error) {
	if err := h.Service.LockThread(ctx, idempotencyKey, threadID, userID); err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	return httptransport.GenericOKResponse{Status: "success", Data: map[string]string{"thread_id": threadID}}, nil
}

func (h Handler) UpdateModeratorsHandler(
	ctx context.Context,
	userID string,
	serverID string,
	idempotencyKey string,
	req httptransport.UpdateModeratorsRequest,
) (httptransport.GenericOKResponse, error) {
	result, err := h.Service.UpdateModerators(ctx, idempotencyKey, serverID, userID, req.ModeratorIDs)
	if err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	return httptransport.GenericOKResponse{
		Status: "success",
		Data: map[string]any{
			"server_id":     result.ServerID,
			"moderator_ids": result.ModeratorIDs,
			"updated_at":    result.UpdatedAt.UTC().Format(time.RFC3339),
		},
	}, nil
}

func (h Handler) MuteUserHandler(
	ctx context.Context,
	userID string,
	targetUserID string,
	idempotencyKey string,
	req httptransport.MuteUserRequest,
) (httptransport.GenericOKResponse, error) {
	duration, err := time.ParseDuration(req.Duration)
	if req.Duration == "" {
		duration = 24 * time.Hour
		err = nil
	}
	if err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	record, err := h.Service.MuteUser(ctx, idempotencyKey, targetUserID, req.ServerID, userID, duration, req.Reason)
	if err != nil {
		return httptransport.GenericOKResponse{}, err
	}
	payload := map[string]any{
		"mute_id":        record.MuteID,
		"user_id":        record.UserID,
		"server_id":      record.ServerID,
		"muted_by_user":  record.MutedByUserID,
		"reason":         record.Reason,
		"created_at":     record.CreatedAt.UTC().Format(time.RFC3339),
		"muted_until":    "",
		"default_window": "24h",
	}
	if record.MutedUntil != nil {
		payload["muted_until"] = record.MutedUntil.UTC().Format(time.RFC3339)
	}
	return httptransport.GenericOKResponse{Status: "success", Data: payload}, nil
}

func (h Handler) AddAttachmentHandler(
	ctx context.Context,
	userID string,
	messageID string,
	idempotencyKey string,
	req httptransport.AddAttachmentRequest,
) (httptransport.AddAttachmentResponse, error) {
	item, err := h.Service.AddAttachment(ctx, idempotencyKey, messageID, userID, req.Filename, req.FileSize, req.MimeType)
	if err != nil {
		return httptransport.AddAttachmentResponse{}, err
	}
	resp := httptransport.AddAttachmentResponse{Status: "success"}
	resp.Data.AttachmentID = item.AttachmentID
	resp.Data.MessageID = item.MessageID
	resp.Data.Filename = item.Filename
	resp.Data.FileSize = item.FileSize
	resp.Data.MimeType = item.MimeType
	resp.Data.URL = item.URL
	resp.Data.ScanResult = item.ScanResult
	resp.Data.ScannedAt = item.ScannedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) GetAttachmentHandler(
	ctx context.Context,
	messageID string,
	attachmentID string,
) (httptransport.GetAttachmentResponse, error) {
	item, err := h.Service.GetAttachment(ctx, messageID, attachmentID)
	if err != nil {
		return httptransport.GetAttachmentResponse{}, err
	}
	resp := httptransport.GetAttachmentResponse{Status: "success"}
	resp.Data.AttachmentID = item.AttachmentID
	resp.Data.MessageID = item.MessageID
	resp.Data.Filename = item.Filename
	resp.Data.FileSize = item.FileSize
	resp.Data.MimeType = item.MimeType
	resp.Data.URL = item.URL
	return resp, nil
}

func (h Handler) ExportMessagesHandler(
	ctx context.Context,
	serverID string,
	channelID string,
	limit int,
) (httptransport.ExportMessagesResponse, error) {
	items, err := h.Service.ExportMessages(ctx, serverID, channelID, limit)
	if err != nil {
		return httptransport.ExportMessagesResponse{}, err
	}
	resp := httptransport.ExportMessagesResponse{Status: "success"}
	resp.Data.Messages = make([]httptransport.MessageDTO, 0, len(items))
	for _, item := range items {
		resp.Data.Messages = append(resp.Data.Messages, toMessageDTO(item))
	}
	resp.Data.Count = len(resp.Data.Messages)
	return resp, nil
}

func toMessageDTO(item ports.Message) httptransport.MessageDTO {
	dto := httptransport.MessageDTO{
		MessageID:      item.MessageID,
		ServerID:       item.ServerID,
		ChannelID:      item.ChannelID,
		ThreadID:       item.ThreadID,
		UserID:         item.UserID,
		Username:       item.Username,
		Content:        item.Content,
		SequenceNumber: item.SequenceNumber,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
		Edited:         item.Edited,
		Reactions:      item.ReactionCounters,
	}
	if item.DeletedAt != nil {
		dto.DeletedAt = item.DeletedAt.UTC().Format(time.RFC3339)
	}
	dto.Mentions = make([]httptransport.MentionDTO, 0, len(item.Mentions))
	for _, m := range item.Mentions {
		dto.Mentions = append(dto.Mentions, httptransport.MentionDTO{
			UserID:   m.UserID,
			Username: m.Username,
		})
	}
	dto.Embeds = make([]httptransport.EmbedDTO, 0, len(item.Embeds))
	for _, e := range item.Embeds {
		dto.Embeds = append(dto.Embeds, httptransport.EmbedDTO{
			URL:         e.URL,
			Title:       e.Title,
			Description: e.Description,
			ImageURL:    e.ImageURL,
		})
	}
	return dto
}
