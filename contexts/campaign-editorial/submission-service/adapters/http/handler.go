package httpadapter

import (
	"context"
	"log/slog"
	"time"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/application/commands"
	"solomon/contexts/campaign-editorial/submission-service/application/queries"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	httptransport "solomon/contexts/campaign-editorial/submission-service/transport/http"
)

type Handler struct {
	CreateSubmission commands.CreateSubmissionUseCase
	ReviewSubmission commands.ReviewSubmissionUseCase
	ReportSubmission commands.ReportSubmissionUseCase
	BulkOperation    commands.BulkOperationUseCase
	Queries          queries.QueryUseCase
	Logger           *slog.Logger
}

// CreateSubmissionHandler godoc
// @Summary Create submission
// @Description Creates a new campaign submission for the authenticated creator.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string true "Creator user id"
// @Param request body httptransport.CreateSubmissionRequest true "Submission create payload"
// @Success 201 {object} httptransport.CreateSubmissionResponse
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 409 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions [post]
func (h Handler) CreateSubmissionHandler(
	ctx context.Context,
	userID string,
	req httptransport.CreateSubmissionRequest,
) (httptransport.CreateSubmissionResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	logger.Info("submission create request received",
		"event", "submission_create_request_received",
		"module", "campaign-editorial/submission-service",
		"layer", "adapter",
		"creator_id", userID,
		"campaign_id", req.CampaignID,
		"platform", req.Platform,
	)
	result, err := h.CreateSubmission.Execute(ctx, commands.CreateSubmissionCommand{
		IdempotencyKey: req.IdempotencyKey,
		CreatorID:      userID,
		CampaignID:     req.CampaignID,
		Platform:       req.Platform,
		PostURL:        req.PostURL,
		CpvRate:        req.CpvRate,
	})
	if err != nil {
		logger.Error("submission create request failed",
			"event", "submission_create_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"creator_id", userID,
			"campaign_id", req.CampaignID,
			"error", err.Error(),
		)
		return httptransport.CreateSubmissionResponse{}, err
	}
	logger.Info("submission create request completed",
		"event", "submission_create_request_completed",
		"module", "campaign-editorial/submission-service",
		"layer", "adapter",
		"submission_id", result.Submission.SubmissionID,
		"replayed", result.Replayed,
	)
	mapped := mapSubmission(result.Submission)
	return httptransport.CreateSubmissionResponse{
		Submission:        mapped,
		Replayed:          result.Replayed,
		SubmissionID:      mapped.SubmissionID,
		CampaignID:        mapped.CampaignID,
		CreatorID:         mapped.CreatorID,
		Platform:          mapped.Platform,
		PostURL:           mapped.PostURL,
		Status:            mapped.Status,
		CreatedAt:         mapped.CreatedAt,
		CpvRate:           result.Submission.CpvRate,
		EstimatedEarnings: "Pending (tracking views)",
	}, nil
}

// GetSubmissionHandler godoc
// @Summary Get submission
// @Description Returns submission details by id.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param submission_id path string true "Submission id"
// @Success 200 {object} httptransport.GetSubmissionResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions/{submission_id} [get]
func (h Handler) GetSubmissionHandler(ctx context.Context, actorID string, submissionID string) (httptransport.GetSubmissionResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	item, err := h.Queries.GetSubmission(ctx, submissionID)
	if err != nil {
		logger.Error("submission get request failed",
			"event", "submission_get_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"error", err.Error(),
		)
		return httptransport.GetSubmissionResponse{}, err
	}
	if actorID != "" && item.CreatorID != actorID {
		logger.Error("submission get request forbidden",
			"event", "submission_get_request_forbidden",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"actor_id", actorID,
		)
		return httptransport.GetSubmissionResponse{}, domainerrors.ErrUnauthorizedActor
	}
	return httptransport.GetSubmissionResponse{
		Submission: mapSubmission(item),
	}, nil
}

// ListSubmissionsHandler godoc
// @Summary List submissions
// @Description Returns submissions filtered by creator, campaign, and status.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string false "Creator user id filter"
// @Param campaign_id query string false "Campaign id filter"
// @Param status query string false "Submission status filter"
// @Success 200 {object} httptransport.ListSubmissionsResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions [get]
func (h Handler) ListSubmissionsHandler(
	ctx context.Context,
	creatorID string,
	campaignID string,
	status string,
) (httptransport.ListSubmissionsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	items, err := h.Queries.ListSubmissions(ctx, queries.ListSubmissionsQuery{
		CreatorID:  creatorID,
		CampaignID: campaignID,
		Status:     status,
	})
	if err != nil {
		logger.Error("submission list request failed",
			"event", "submission_list_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"creator_id", creatorID,
			"campaign_id", campaignID,
			"status", status,
			"error", err.Error(),
		)
		return httptransport.ListSubmissionsResponse{}, err
	}
	result := make([]httptransport.SubmissionDTO, 0, len(items))
	for _, item := range items {
		result = append(result, mapSubmission(item))
	}
	return httptransport.ListSubmissionsResponse{Items: result}, nil
}

// ApproveSubmissionHandler godoc
// @Summary Approve submission
// @Description Approves a pending or flagged submission.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string true "Approver user id"
// @Param submission_id path string true "Submission id"
// @Param request body httptransport.ApproveSubmissionRequest true "Approve payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 403 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 409 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions/{submission_id}/approve [post]
func (h Handler) ApproveSubmissionHandler(
	ctx context.Context,
	actorID string,
	submissionID string,
	req httptransport.ApproveSubmissionRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.ReviewSubmission.Approve(ctx, commands.ApproveSubmissionCommand{
		IdempotencyKey: req.IdempotencyKey,
		SubmissionID:   submissionID,
		ActorID:        actorID,
		Reason:         req.Reason,
	}); err != nil {
		logger.Error("submission approve request failed",
			"event", "submission_approve_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"actor_id", actorID,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("submission approve request completed",
		"event", "submission_approve_request_completed",
		"module", "campaign-editorial/submission-service",
		"layer", "adapter",
		"submission_id", submissionID,
		"actor_id", actorID,
	)
	return nil
}

// RejectSubmissionHandler godoc
// @Summary Reject submission
// @Description Rejects a pending or flagged submission with reason and notes.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string true "Reviewer user id"
// @Param submission_id path string true "Submission id"
// @Param request body httptransport.RejectSubmissionRequest true "Reject payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 403 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 409 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions/{submission_id}/reject [post]
func (h Handler) RejectSubmissionHandler(
	ctx context.Context,
	actorID string,
	submissionID string,
	req httptransport.RejectSubmissionRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.ReviewSubmission.Reject(ctx, commands.RejectSubmissionCommand{
		IdempotencyKey: req.IdempotencyKey,
		SubmissionID:   submissionID,
		ActorID:        actorID,
		Reason:         req.Reason,
		Notes:          req.Notes,
	}); err != nil {
		logger.Error("submission reject request failed",
			"event", "submission_reject_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"actor_id", actorID,
			"error", err.Error(),
		)
		return err
	}
	logger.Info("submission reject request completed",
		"event", "submission_reject_request_completed",
		"module", "campaign-editorial/submission-service",
		"layer", "adapter",
		"submission_id", submissionID,
		"actor_id", actorID,
	)
	return nil
}

// ReportSubmissionHandler godoc
// @Summary Report submission
// @Description Files a user report against a submission and applies flag workflow.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string true "Reporter user id"
// @Param submission_id path string true "Submission id"
// @Param request body httptransport.ReportSubmissionRequest true "Report payload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httptransport.ErrorResponse
// @Failure 403 {object} httptransport.ErrorResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 409 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions/{submission_id}/report [post]
func (h Handler) ReportSubmissionHandler(
	ctx context.Context,
	reporterID string,
	submissionID string,
	req httptransport.ReportSubmissionRequest,
) error {
	logger := application.ResolveLogger(h.Logger)
	if err := h.ReportSubmission.Execute(ctx, commands.ReportSubmissionCommand{
		IdempotencyKey: req.IdempotencyKey,
		SubmissionID:   submissionID,
		ReporterID:     reporterID,
		Reason:         req.Reason,
		Description:    req.Description,
	}); err != nil {
		logger.Error("submission report request failed",
			"event", "submission_report_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"reporter_id", reporterID,
			"error", err.Error(),
		)
		return err
	}
	logger.Warn("submission report request completed",
		"event", "submission_report_request_completed",
		"module", "campaign-editorial/submission-service",
		"layer", "adapter",
		"submission_id", submissionID,
		"reporter_id", reporterID,
	)
	return nil
}

func (h Handler) BulkOperationHandler(
	ctx context.Context,
	actorID string,
	req httptransport.BulkOperationRequest,
) (httptransport.BulkOperationResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	result, err := h.BulkOperation.Execute(ctx, commands.BulkOperationCommand{
		IdempotencyKey: req.IdempotencyKey,
		ActorID:        actorID,
		OperationType:  req.OperationType,
		SubmissionIDs:  req.SubmissionIDs,
		ReasonCode:     req.ReasonCode,
		Reason:         req.Reason,
	})
	if err != nil {
		logger.Error("submission bulk operation failed",
			"event", "submission_bulk_operation_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"actor_id", actorID,
			"operation_type", req.OperationType,
			"error", err.Error(),
		)
		return httptransport.BulkOperationResponse{}, err
	}
	return httptransport.BulkOperationResponse{
		Processed:      result.Processed,
		SucceededCount: result.SucceededCount,
		FailedCount:    result.FailedCount,
	}, nil
}

// CreatorDashboardHandler godoc
// @Summary Get creator submission dashboard
// @Description Returns submission summary counts for a creator.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string false "Creator user id"
// @Success 200 {object} httptransport.DashboardResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /dashboard/creator [get]
func (h Handler) CreatorDashboardHandler(ctx context.Context, creatorID string) (httptransport.DashboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	summary, err := h.Queries.CreatorDashboard(ctx, creatorID)
	if err != nil {
		logger.Error("creator dashboard request failed",
			"event", "submission_creator_dashboard_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"creator_id", creatorID,
			"error", err.Error(),
		)
		return httptransport.DashboardResponse{}, err
	}
	return mapDashboard(summary), nil
}

// BrandDashboardHandler godoc
// @Summary Get brand submission dashboard
// @Description Returns submission summary counts for a campaign.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param campaign_id query string false "Campaign id"
// @Success 200 {object} httptransport.DashboardResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /dashboard/brand [get]
func (h Handler) BrandDashboardHandler(ctx context.Context, campaignID string) (httptransport.DashboardResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	summary, err := h.Queries.BrandDashboard(ctx, campaignID)
	if err != nil {
		logger.Error("brand dashboard request failed",
			"event", "submission_brand_dashboard_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"campaign_id", campaignID,
			"error", err.Error(),
		)
		return httptransport.DashboardResponse{}, err
	}
	return mapDashboard(summary), nil
}

// AnalyticsHandler godoc
// @Summary Get submission analytics
// @Description Returns basic analytics for one submission.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param submission_id path string true "Submission id"
// @Success 200 {object} httptransport.AnalyticsResponse
// @Failure 404 {object} httptransport.ErrorResponse
// @Failure 500 {object} httptransport.ErrorResponse
// @Router /submissions/{submission_id}/analytics [get]
func (h Handler) AnalyticsHandler(ctx context.Context, actorID string, submissionID string) (httptransport.AnalyticsResponse, error) {
	logger := application.ResolveLogger(h.Logger)
	item, err := h.Queries.GetSubmission(ctx, submissionID)
	if err != nil {
		logger.Error("submission analytics request failed",
			"event", "submission_analytics_request_failed",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"error", err.Error(),
		)
		return httptransport.AnalyticsResponse{}, err
	}
	if actorID != "" && item.CreatorID != actorID {
		logger.Error("submission analytics request forbidden",
			"event", "submission_analytics_request_forbidden",
			"module", "campaign-editorial/submission-service",
			"layer", "adapter",
			"submission_id", submissionID,
			"actor_id", actorID,
		)
		return httptransport.AnalyticsResponse{}, domainerrors.ErrUnauthorizedActor
	}
	return httptransport.AnalyticsResponse{
		SubmissionID: item.SubmissionID,
		ViewCount:    int64(item.ViewsCount),
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
		PostID:        item.PostID,
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
	if item.VerificationStart != nil {
		dto.VerificationStart = item.VerificationStart.Format(time.RFC3339)
	}
	if item.VerificationWindowEnd != nil {
		dto.VerificationEnd = item.VerificationWindowEnd.Format(time.RFC3339)
	}
	dto.ViewsCount = item.ViewsCount
	if item.LockedViews != nil {
		dto.LockedViews = *item.LockedViews
	}
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
