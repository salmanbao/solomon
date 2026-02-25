package httpadapter

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/application/commands"
	"solomon/contexts/campaign-editorial/submission-service/application/queries"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

type Handler struct {
	CreateSubmission commands.CreateSubmissionUseCase
	ReviewSubmission commands.ReviewSubmissionUseCase
	ReportSubmission commands.ReportSubmissionUseCase
	Queries          queries.QueryUseCase
	Logger           *slog.Logger
}

func (h Handler) CreateSubmissionHandler(
	ctx context.Context,
	userID string,
	req httptransport.CreateSubmissionRequest,
) (httptransport.CreateSubmissionResponse, error) {
	item, err := h.CreateSubmission.Execute(ctx, commands.CreateSubmissionCommand{
		CreatorID:  userID,
		CampaignID: req.CampaignID,
		Platform:   req.Platform,
		PostURL:    req.PostURL,
	})
	if err != nil {
		return httptransport.CreateSubmissionResponse{}, err
	}
	return httptransport.CreateSubmissionResponse{
		Submission: mapSubmission(item),
	}, nil
}

func (h Handler) GetSubmissionHandler(ctx context.Context, submissionID string) (httptransport.GetSubmissionResponse, error) {
	item, err := h.Queries.GetSubmission(ctx, submissionID)
	if err != nil {
		return httptransport.GetSubmissionResponse{}, err
	}
	return httptransport.GetSubmissionResponse{
		Submission: mapSubmission(item),
	}, nil
}

func (h Handler) ListSubmissionsHandler(
	ctx context.Context,
	creatorID string,
	campaignID string,
	status string,
) (httptransport.ListSubmissionsResponse, error) {
	items, err := h.Queries.ListSubmissions(ctx, queries.ListSubmissionsQuery{
		CreatorID:  creatorID,
		CampaignID: campaignID,
		Status:     status,
	})
	if err != nil {
		return httptransport.ListSubmissionsResponse{}, err
	}
	result := make([]httptransport.SubmissionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, mapSubmission(item))
	}
	return httptransport.ListSubmissionsResponse{Items: result}, nil
}

func (h Handler) ApproveSubmissionHandler(
	ctx context.Context,
	actorID string,
	submissionID string,
	req httptransport.ApproveSubmissionRequest,
) error {
	return h.ReviewSubmission.Approve(ctx, commands.ApproveSubmissionCommand{
		SubmissionID: submissionID,
		ActorID:      actorID,
		Reason:       req.Reason,
	})
}

func (h Handler) RejectSubmissionHandler(
	ctx context.Context,
	actorID string,
	submissionID string,
	req httptransport.RejectSubmissionRequest,
) error {
	return h.ReviewSubmission.Reject(ctx, commands.RejectSubmissionCommand{
		SubmissionID: submissionID,
		ActorID:      actorID,
		Reason:       req.Reason,
		Notes:        req.Notes,
	})
}

func (h Handler) ReportSubmissionHandler(
	ctx context.Context,
	reporterID string,
	submissionID string,
	req httptransport.ReportSubmissionRequest,
) error {
	return h.ReportSubmission.Execute(ctx, commands.ReportSubmissionCommand{
		SubmissionID: submissionID,
		ReporterID:   reporterID,
		Reason:       req.Reason,
		Description:  req.Description,
	})
}

func (h Handler) CreatorDashboardHandler(ctx context.Context, creatorID string) (httptransport.DashboardResponse, error) {
	summary, err := h.Queries.CreatorDashboard(ctx, creatorID)
	if err != nil {
		return httptransport.DashboardResponse{}, err
	}
	return mapDashboard(summary), nil
}

func (h Handler) BrandDashboardHandler(ctx context.Context, campaignID string) (httptransport.DashboardResponse, error) {
	summary, err := h.Queries.BrandDashboard(ctx, campaignID)
	if err != nil {
		return httptransport.DashboardResponse{}, err
	}
	return mapDashboard(summary), nil
}

func (h Handler) AnalyticsHandler(ctx context.Context, submissionID string) (httptransport.AnalyticsResponse, error) {
	item, err := h.Queries.GetSubmission(ctx, submissionID)
	if err != nil {
		return httptransport.AnalyticsResponse{}, err
	}
	return httptransport.AnalyticsResponse{
		SubmissionID: item.SubmissionID,
		ViewCount:    0,
		Reported:     item.ReportedCount,
	}, nil
}

func mapSubmission(item entities.Submission) httptransport.SubmissionDTO {
	dto := httptransport.SubmissionDTO{
		SubmissionID:  item.SubmissionID,
		CampaignID:    item.CampaignID,
		CreatorID:     item.CreatorID,
		Platform:      item.Platform,
		PostURL:       item.PostURL,
		Status:        string(item.Status),
		CreatedAt:     item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     item.UpdatedAt.Format(time.RFC3339),
		ReportedCount: item.ReportedCount,
	}
	if item.ApprovedAt != nil {
		dto.ApprovedAt = item.ApprovedAt.Format(time.RFC3339)
	}
	if item.RejectedAt != nil {
		dto.RejectedAt = item.RejectedAt.Format(time.RFC3339)
	}
	dto.ApprovedByUserID = item.ApprovedByUserID
	dto.ApprovalReason = item.ApprovalReason
	dto.RejectionReason = item.RejectionReason
	dto.RejectionNotes = item.RejectionNotes
	return dto
}

func mapDashboard(summary queries.DashboardSummary) httptransport.DashboardResponse {
	return httptransport.DashboardResponse{
		Total:    summary.Total,
		Pending:  summary.Pending,
		Approved: summary.Approved,
		Rejected: summary.Rejected,
		Flagged:  summary.Flagged,
	}
}

func (h Handler) logDebug(message string) {
	application.ResolveLogger(h.Logger).Debug(message,
		"event", "submission_transport_debug",
		"module", "campaign-editorial/submission-service",
		"layer", "transport",
	)
}
