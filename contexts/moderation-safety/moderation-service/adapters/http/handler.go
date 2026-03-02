package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/moderation-safety/moderation-service/application"
	"solomon/contexts/moderation-safety/moderation-service/ports"
	httptransport "solomon/contexts/moderation-safety/moderation-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) ListQueueHandler(ctx context.Context, statusRaw string, limitRaw string, offsetRaw string) (httptransport.QueueResponse, error) {
	filter := ports.QueueFilter{Status: strings.TrimSpace(statusRaw)}
	if parsed, err := strconv.Atoi(strings.TrimSpace(limitRaw)); err == nil {
		filter.Limit = parsed
	}
	if parsed, err := strconv.Atoi(strings.TrimSpace(offsetRaw)); err == nil {
		filter.Offset = parsed
	}
	items, err := h.Service.ListQueue(ctx, filter)
	if err != nil {
		return httptransport.QueueResponse{}, err
	}
	resp := httptransport.QueueResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.Items = make([]struct {
		SubmissionID        string  `json:"submission_id"`
		CampaignID          string  `json:"campaign_id"`
		CreatorID           string  `json:"creator_id"`
		Status              string  `json:"status"`
		RiskScore           float64 `json:"risk_score"`
		ReportCount         int     `json:"report_count"`
		QueuedAt            string  `json:"queued_at"`
		AssignedModeratorID string  `json:"assigned_moderator_id,omitempty"`
	}, 0, len(items))
	for _, item := range items {
		resp.Data.Items = append(resp.Data.Items, struct {
			SubmissionID        string  `json:"submission_id"`
			CampaignID          string  `json:"campaign_id"`
			CreatorID           string  `json:"creator_id"`
			Status              string  `json:"status"`
			RiskScore           float64 `json:"risk_score"`
			ReportCount         int     `json:"report_count"`
			QueuedAt            string  `json:"queued_at"`
			AssignedModeratorID string  `json:"assigned_moderator_id,omitempty"`
		}{
			SubmissionID:        item.SubmissionID,
			CampaignID:          item.CampaignID,
			CreatorID:           item.CreatorID,
			Status:              item.Status,
			RiskScore:           item.RiskScore,
			ReportCount:         item.ReportCount,
			QueuedAt:            item.QueuedAt.UTC().Format(time.RFC3339),
			AssignedModeratorID: item.AssignedModeratorID,
		})
	}
	return resp, nil
}

func (h Handler) ApproveHandler(ctx context.Context, idempotencyKey string, moderatorID string, req httptransport.ApproveRequest) (httptransport.DecisionResponse, error) {
	record, err := h.Service.Approve(ctx, idempotencyKey, moderatorID, ports.ModerationActionInput{
		SubmissionID: strings.TrimSpace(req.SubmissionID),
		CampaignID:   strings.TrimSpace(req.CampaignID),
		Reason:       strings.TrimSpace(req.Reason),
		Notes:        strings.TrimSpace(req.Notes),
	})
	if err != nil {
		return httptransport.DecisionResponse{}, err
	}
	return mapDecisionResponse(record), nil
}

func (h Handler) RejectHandler(ctx context.Context, idempotencyKey string, moderatorID string, req httptransport.RejectRequest) (httptransport.DecisionResponse, error) {
	record, err := h.Service.Reject(ctx, idempotencyKey, moderatorID, ports.ModerationActionInput{
		SubmissionID: strings.TrimSpace(req.SubmissionID),
		CampaignID:   strings.TrimSpace(req.CampaignID),
		Reason:       strings.TrimSpace(req.RejectionReason),
		Notes:        strings.TrimSpace(req.RejectionNotes),
	})
	if err != nil {
		return httptransport.DecisionResponse{}, err
	}
	return mapDecisionResponse(record), nil
}

func (h Handler) FlagHandler(ctx context.Context, idempotencyKey string, moderatorID string, req httptransport.FlagRequest) (httptransport.DecisionResponse, error) {
	record, err := h.Service.Flag(ctx, idempotencyKey, moderatorID, ports.ModerationActionInput{
		SubmissionID: strings.TrimSpace(req.SubmissionID),
		CampaignID:   strings.TrimSpace(req.CampaignID),
		Reason:       strings.TrimSpace(req.FlagReason),
		Notes:        strings.TrimSpace(req.Notes),
		Severity:     strings.TrimSpace(req.Severity),
	})
	if err != nil {
		return httptransport.DecisionResponse{}, err
	}
	return mapDecisionResponse(record), nil
}

func mapDecisionResponse(record ports.DecisionRecord) httptransport.DecisionResponse {
	resp := httptransport.DecisionResponse{Status: "success", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	resp.Data.DecisionID = record.DecisionID
	resp.Data.SubmissionID = record.SubmissionID
	resp.Data.CampaignID = record.CampaignID
	resp.Data.ModeratorID = record.ModeratorID
	resp.Data.Action = record.Action
	resp.Data.Reason = record.Reason
	resp.Data.Notes = record.Notes
	resp.Data.Severity = record.Severity
	resp.Data.QueueStatus = record.QueueStatus
	resp.Data.CreatedAt = record.CreatedAt.UTC().Format(time.RFC3339)
	return resp
}
