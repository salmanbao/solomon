package http

import (
	"context"
	"time"

	"solomon/contexts/internal-ops/admin-dashboard-service/application"
	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	httptransport "solomon/contexts/internal-ops/admin-dashboard-service/transport/http"
)

type Handler struct {
	Service application.Service
}

func (h Handler) RecordAdminActionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RecordAdminActionRequest,
) (httptransport.RecordAdminActionResponse, error) {
	row, err := h.Service.RecordAdminAction(ctx, idempotencyKey, application.RecordActionInput{
		ActorID:       adminID,
		Action:        req.Action,
		TargetID:      req.TargetID,
		Justification: req.Justification,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RecordAdminActionResponse{}, err
	}
	return httptransport.RecordAdminActionResponse{
		AuditID:    row.AuditID,
		OccurredAt: row.OccurredAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) GrantIdentityRoleHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.GrantIdentityRoleRequest,
) (httptransport.GrantIdentityRoleResponse, error) {
	result, err := h.Service.GrantIdentityRole(ctx, idempotencyKey, application.GrantIdentityRoleInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		RoleID:        req.RoleID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.GrantIdentityRoleResponse{}, err
	}
	return httptransport.GrantIdentityRoleResponse{
		AssignmentID:        result.AssignmentID,
		UserID:              result.UserID,
		RoleID:              result.RoleID,
		OwnerAuditLogID:     result.OwnerAuditLogID,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
		OccurredAt:          result.OccurredAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) ModerateSubmissionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.ModerateSubmissionRequest,
) (httptransport.ModerateSubmissionResponse, error) {
	result, err := h.Service.ModerateSubmission(ctx, idempotencyKey, application.ModerateSubmissionInput{
		ActorID:       adminID,
		SubmissionID:  req.SubmissionID,
		CampaignID:    req.CampaignID,
		Action:        req.Action,
		Reason:        req.Reason,
		Notes:         req.Notes,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ModerateSubmissionResponse{}, err
	}
	return httptransport.ModerateSubmissionResponse{
		DecisionID:          result.DecisionID,
		SubmissionID:        result.SubmissionID,
		CampaignID:          result.CampaignID,
		ModeratorID:         result.ModeratorID,
		Action:              result.Action,
		Reason:              result.Reason,
		Notes:               result.Notes,
		QueueStatus:         result.QueueStatus,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) ReleaseAbuseLockoutHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.ReleaseAbuseLockoutRequest,
) (httptransport.ReleaseAbuseLockoutResponse, error) {
	result, err := h.Service.ReleaseAbuseLockout(ctx, idempotencyKey, application.ReleaseAbuseLockoutInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ReleaseAbuseLockoutResponse{}, err
	}
	return httptransport.ReleaseAbuseLockoutResponse{
		ThreatID:            result.ThreatID,
		UserID:              result.UserID,
		Status:              result.Status,
		ReleasedAt:          result.ReleasedAt.UTC().Format(time.RFC3339),
		OwnerAuditLogID:     result.OwnerAuditLogID,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) CreateFinanceRefundHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.CreateFinanceRefundRequest,
) (httptransport.CreateFinanceRefundResponse, error) {
	result, err := h.Service.CreateFinanceRefund(ctx, idempotencyKey, application.CreateFinanceRefundInput{
		ActorID:       adminID,
		TransactionID: req.TransactionID,
		UserID:        req.UserID,
		Amount:        req.Amount,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CreateFinanceRefundResponse{}, err
	}
	return httptransport.CreateFinanceRefundResponse{
		RefundID:            result.RefundID,
		TransactionID:       result.TransactionID,
		UserID:              result.UserID,
		Amount:              result.Amount,
		Reason:              result.Reason,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) CreateBillingRefundHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.CreateBillingRefundRequest,
) (httptransport.CreateBillingRefundResponse, error) {
	result, err := h.Service.CreateBillingRefund(ctx, idempotencyKey, application.CreateBillingRefundInput{
		ActorID:       adminID,
		InvoiceID:     req.InvoiceID,
		LineItemID:    req.LineItemID,
		Amount:        req.Amount,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CreateBillingRefundResponse{}, err
	}
	return httptransport.CreateBillingRefundResponse{
		RefundID:            result.RefundID,
		InvoiceID:           result.InvoiceID,
		LineItemID:          result.LineItemID,
		Amount:              result.Amount,
		Reason:              result.Reason,
		ProcessedAt:         result.ProcessedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) RecalculateRewardHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RecalculateRewardRequest,
) (httptransport.RecalculateRewardResponse, error) {
	verificationCompletedAt := time.Time{}
	if req.VerificationCompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.VerificationCompletedAt)
		if err != nil {
			return httptransport.RecalculateRewardResponse{}, domainerrors.ErrInvalidInput
		}
		verificationCompletedAt = parsed
	}
	result, err := h.Service.RecalculateReward(ctx, idempotencyKey, application.RecalculateRewardInput{
		ActorID:                 adminID,
		UserID:                  req.UserID,
		SubmissionID:            req.SubmissionID,
		CampaignID:              req.CampaignID,
		LockedViews:             req.LockedViews,
		RatePer1K:               req.RatePer1K,
		FraudScore:              req.FraudScore,
		VerificationCompletedAt: verificationCompletedAt,
		Reason:                  req.Reason,
		SourceIP:                req.SourceIP,
		CorrelationID:           req.CorrelationID,
	})
	if err != nil {
		return httptransport.RecalculateRewardResponse{}, err
	}
	var eligibleAt *string
	if result.EligibleAt != nil {
		value := result.EligibleAt.UTC().Format(time.RFC3339)
		eligibleAt = &value
	}
	return httptransport.RecalculateRewardResponse{
		SubmissionID:        result.SubmissionID,
		UserID:              result.UserID,
		CampaignID:          result.CampaignID,
		Status:              result.Status,
		NetAmount:           result.NetAmount,
		RolloverTotal:       result.RolloverTotal,
		CalculatedAt:        result.CalculatedAt.UTC().Format(time.RFC3339),
		EligibleAt:          eligibleAt,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) SuspendAffiliateHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.SuspendAffiliateRequest,
) (httptransport.SuspendAffiliateResponse, error) {
	result, err := h.Service.SuspendAffiliate(ctx, idempotencyKey, application.SuspendAffiliateInput{
		ActorID:       adminID,
		AffiliateID:   req.AffiliateID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.SuspendAffiliateResponse{}, err
	}
	return httptransport.SuspendAffiliateResponse{
		AffiliateID:         result.AffiliateID,
		Status:              result.Status,
		UpdatedAt:           result.UpdatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) CreateAffiliateAttributionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.CreateAffiliateAttributionRequest,
) (httptransport.CreateAffiliateAttributionResponse, error) {
	result, err := h.Service.CreateAffiliateAttribution(ctx, idempotencyKey, application.CreateAffiliateAttributionInput{
		ActorID:       adminID,
		AffiliateID:   req.AffiliateID,
		ClickID:       req.ClickID,
		OrderID:       req.OrderID,
		ConversionID:  req.ConversionID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CreateAffiliateAttributionResponse{}, err
	}
	return httptransport.CreateAffiliateAttributionResponse{
		AttributionID:       result.AttributionID,
		AffiliateID:         result.AffiliateID,
		OrderID:             result.OrderID,
		Amount:              result.Amount,
		AttributedAt:        result.AttributedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) RetryFailedPayoutHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RetryFailedPayoutRequest,
) (httptransport.RetryFailedPayoutResponse, error) {
	result, err := h.Service.RetryPayout(ctx, idempotencyKey, application.RetryPayoutInput{
		ActorID:       adminID,
		PayoutID:      req.PayoutID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RetryFailedPayoutResponse{}, err
	}
	return httptransport.RetryFailedPayoutResponse{
		PayoutID:            result.PayoutID,
		UserID:              result.UserID,
		Status:              result.Status,
		FailureReason:       result.FailureReason,
		ProcessedAt:         result.ProcessedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) ResolveDisputeHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.ResolveDisputeRequest,
) (httptransport.ResolveDisputeResponse, error) {
	result, err := h.Service.ResolveDispute(ctx, idempotencyKey, application.ResolveDisputeInput{
		ActorID:       adminID,
		DisputeID:     req.DisputeID,
		Action:        req.Action,
		Reason:        req.Reason,
		Notes:         req.Notes,
		RefundAmount:  req.RefundAmount,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ResolveDisputeResponse{}, err
	}
	return httptransport.ResolveDisputeResponse{
		DisputeID:           result.DisputeID,
		Status:              result.Status,
		ResolutionType:      result.ResolutionType,
		RefundAmount:        result.RefundAmount,
		ProcessedAt:         result.ProcessedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) GetConsentHandler(
	ctx context.Context,
	adminID string,
	req httptransport.GetConsentRequest,
) (httptransport.GetConsentResponse, error) {
	result, err := h.Service.GetConsent(ctx, application.GetConsentInput{
		ActorID: adminID,
		UserID:  req.UserID,
	})
	if err != nil {
		return httptransport.GetConsentResponse{}, err
	}
	return httptransport.GetConsentResponse{
		UserID:        result.UserID,
		Status:        result.Status,
		Preferences:   result.Preferences,
		LastUpdated:   result.LastUpdated.UTC().Format(time.RFC3339),
		LastUpdatedBy: result.LastUpdatedBy,
	}, nil
}

func (h Handler) UpdateConsentHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.UpdateConsentRequest,
) (httptransport.UpdateConsentResponse, error) {
	result, err := h.Service.UpdateConsent(ctx, idempotencyKey, application.UpdateConsentInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		Preferences:   req.Preferences,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.UpdateConsentResponse{}, err
	}
	return httptransport.UpdateConsentResponse{
		UserID:              result.UserID,
		Status:              result.Status,
		UpdatedAt:           result.UpdatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) WithdrawConsentHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.WithdrawConsentRequest,
) (httptransport.WithdrawConsentResponse, error) {
	result, err := h.Service.WithdrawConsent(ctx, idempotencyKey, application.WithdrawConsentInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		Category:      req.Category,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.WithdrawConsentResponse{}, err
	}
	return httptransport.WithdrawConsentResponse{
		UserID:              result.UserID,
		Status:              result.Status,
		UpdatedAt:           result.UpdatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) StartDataExportHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.StartDataExportRequest,
) (httptransport.StartDataExportResponse, error) {
	result, err := h.Service.StartDataExport(ctx, idempotencyKey, application.StartDataExportInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		Format:        req.Format,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.StartDataExportResponse{}, err
	}
	var completedAt *string
	if result.CompletedAt != nil {
		value := result.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &value
	}
	return httptransport.StartDataExportResponse{
		RequestID:           result.RequestID,
		UserID:              result.UserID,
		RequestType:         result.RequestType,
		Format:              result.Format,
		Status:              result.Status,
		RequestedAt:         result.RequestedAt.UTC().Format(time.RFC3339),
		CompletedAt:         completedAt,
		DownloadURL:         result.DownloadURL,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) GetDataExportHandler(
	ctx context.Context,
	adminID string,
	requestID string,
	sourceIP string,
	correlationID string,
) (httptransport.GetDataExportResponse, error) {
	result, err := h.Service.GetDataExport(ctx, application.GetDataExportInput{
		ActorID:       adminID,
		RequestID:     requestID,
		SourceIP:      sourceIP,
		CorrelationID: correlationID,
	})
	if err != nil {
		return httptransport.GetDataExportResponse{}, err
	}
	var completedAt *string
	if result.CompletedAt != nil {
		value := result.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &value
	}
	return httptransport.GetDataExportResponse{
		RequestID:   result.RequestID,
		UserID:      result.UserID,
		RequestType: result.RequestType,
		Format:      result.Format,
		Status:      result.Status,
		Reason:      result.Reason,
		RequestedAt: result.RequestedAt.UTC().Format(time.RFC3339),
		CompletedAt: completedAt,
		DownloadURL: result.DownloadURL,
	}, nil
}

func (h Handler) RequestDeletionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RequestDeletionRequest,
) (httptransport.RequestDeletionResponse, error) {
	result, err := h.Service.RequestDeletion(ctx, idempotencyKey, application.RequestDeletionInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RequestDeletionResponse{}, err
	}
	var completedAt *string
	if result.CompletedAt != nil {
		value := result.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &value
	}
	return httptransport.RequestDeletionResponse{
		RequestID:           result.RequestID,
		UserID:              result.UserID,
		Status:              result.Status,
		Reason:              result.Reason,
		RequestedAt:         result.RequestedAt.UTC().Format(time.RFC3339),
		CompletedAt:         completedAt,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) CreateRetentionLegalHoldHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.CreateRetentionLegalHoldRequest,
) (httptransport.CreateRetentionLegalHoldResponse, error) {
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return httptransport.CreateRetentionLegalHoldResponse{}, domainerrors.ErrInvalidInput
		}
		utc := parsed.UTC()
		expiresAt = &utc
	}
	result, err := h.Service.CreateRetentionLegalHold(ctx, idempotencyKey, application.CreateRetentionLegalHoldInput{
		ActorID:       adminID,
		EntityID:      req.EntityID,
		DataType:      req.DataType,
		Reason:        req.Reason,
		ExpiresAt:     expiresAt,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CreateRetentionLegalHoldResponse{}, err
	}
	var expiresAtOut *string
	if result.ExpiresAt != nil {
		value := result.ExpiresAt.UTC().Format(time.RFC3339)
		expiresAtOut = &value
	}
	return httptransport.CreateRetentionLegalHoldResponse{
		HoldID:              result.HoldID,
		EntityID:            result.EntityID,
		DataType:            result.DataType,
		Status:              result.Status,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ExpiresAt:           expiresAtOut,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) CheckLegalHoldHandler(
	ctx context.Context,
	adminID string,
	req httptransport.CheckLegalHoldRequest,
) (httptransport.CheckLegalHoldResponse, error) {
	result, err := h.Service.CheckLegalHold(ctx, application.CheckLegalHoldInput{
		ActorID:       adminID,
		EntityType:    req.EntityType,
		EntityID:      req.EntityID,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CheckLegalHoldResponse{}, err
	}
	return httptransport.CheckLegalHoldResponse{
		EntityType:          result.EntityType,
		EntityID:            result.EntityID,
		Held:                result.Held,
		HoldID:              result.HoldID,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) ReleaseLegalHoldHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.ReleaseLegalHoldRequest,
) (httptransport.ReleaseLegalHoldResponse, error) {
	result, err := h.Service.ReleaseLegalHold(ctx, idempotencyKey, application.ReleaseLegalHoldInput{
		ActorID:       adminID,
		HoldID:        req.HoldID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ReleaseLegalHoldResponse{}, err
	}
	var releasedAt *string
	if result.ReleasedAt != nil {
		value := result.ReleasedAt.UTC().Format(time.RFC3339)
		releasedAt = &value
	}
	return httptransport.ReleaseLegalHoldResponse{
		HoldID:              result.HoldID,
		EntityType:          result.EntityType,
		EntityID:            result.EntityID,
		Status:              result.Status,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ReleasedAt:          releasedAt,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) RunComplianceScanHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RunComplianceScanRequest,
) (httptransport.RunComplianceScanResponse, error) {
	result, err := h.Service.RunComplianceScan(ctx, idempotencyKey, application.RunComplianceScanInput{
		ActorID:       adminID,
		ReportType:    req.ReportType,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RunComplianceScanResponse{}, err
	}
	return httptransport.RunComplianceScanResponse{
		ReportID:            result.ReportID,
		ReportType:          result.ReportType,
		Status:              result.Status,
		FindingsCount:       result.FindingsCount,
		DownloadURL:         result.DownloadURL,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) GetSupportTicketHandler(
	ctx context.Context,
	adminID string,
	req httptransport.GetSupportTicketRequest,
) (httptransport.SupportTicketResponse, error) {
	result, err := h.Service.GetSupportTicket(ctx, application.GetSupportTicketInput{
		ActorID:  adminID,
		TicketID: req.TicketID,
	})
	if err != nil {
		return httptransport.SupportTicketResponse{}, err
	}
	return toSupportTicketResponse(result), nil
}

func (h Handler) SearchSupportTicketsHandler(
	ctx context.Context,
	adminID string,
	req httptransport.SearchSupportTicketsRequest,
) (httptransport.SearchSupportTicketsResponse, error) {
	results, err := h.Service.SearchSupportTickets(ctx, application.SearchSupportTicketsInput{
		ActorID:    adminID,
		Query:      req.Query,
		Status:     req.Status,
		Category:   req.Category,
		AssignedTo: req.AssignedTo,
		Limit:      req.Limit,
	})
	if err != nil {
		return httptransport.SearchSupportTicketsResponse{}, err
	}
	resp := httptransport.SearchSupportTicketsResponse{
		Tickets: make([]httptransport.SupportTicketResponse, 0, len(results)),
	}
	for _, result := range results {
		resp.Tickets = append(resp.Tickets, toSupportTicketResponse(result))
	}
	return resp, nil
}

func (h Handler) AssignSupportTicketHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.AssignSupportTicketRequest,
) (httptransport.AssignSupportTicketResponse, error) {
	result, err := h.Service.AssignSupportTicket(ctx, idempotencyKey, application.AssignSupportTicketInput{
		ActorID:       adminID,
		TicketID:      req.TicketID,
		AgentID:       req.AgentID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.AssignSupportTicketResponse{}, err
	}
	return httptransport.AssignSupportTicketResponse{
		SupportTicketResponse: toSupportTicketResponse(result.SupportTicketResult),
		ControlPlaneAuditID:   result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) UpdateSupportTicketHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.UpdateSupportTicketRequest,
) (httptransport.UpdateSupportTicketResponse, error) {
	result, err := h.Service.UpdateSupportTicket(ctx, idempotencyKey, application.UpdateSupportTicketInput{
		ActorID:       adminID,
		TicketID:      req.TicketID,
		Status:        req.Status,
		SubStatus:     req.SubStatus,
		Priority:      req.Priority,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.UpdateSupportTicketResponse{}, err
	}
	return httptransport.UpdateSupportTicketResponse{
		SupportTicketResponse: toSupportTicketResponse(result.SupportTicketResult),
		ControlPlaneAuditID:   result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) SaveEditorCampaignHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.SaveEditorCampaignRequest,
) (httptransport.SaveEditorCampaignResponse, error) {
	result, err := h.Service.SaveEditorCampaign(ctx, idempotencyKey, application.SaveEditorCampaignInput{
		ActorID:       adminID,
		EditorID:      req.EditorID,
		CampaignID:    req.CampaignID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.SaveEditorCampaignResponse{}, err
	}
	var savedAt *string
	if result.SavedAt != nil {
		value := result.SavedAt.UTC().Format(time.RFC3339)
		savedAt = &value
	}
	return httptransport.SaveEditorCampaignResponse{
		CampaignID:          result.CampaignID,
		Saved:               result.Saved,
		SavedAt:             savedAt,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) RequestClippingExportHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RequestClippingExportRequest,
) (httptransport.RequestClippingExportResponse, error) {
	result, err := h.Service.RequestClippingExport(ctx, idempotencyKey, application.RequestClippingExportInput{
		ActorID:       adminID,
		UserID:        req.UserID,
		ProjectID:     req.ProjectID,
		Format:        req.Format,
		Resolution:    req.Resolution,
		FPS:           req.FPS,
		Bitrate:       req.Bitrate,
		CampaignID:    req.CampaignID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RequestClippingExportResponse{}, err
	}
	var completedAt *string
	if result.CompletedAt != nil {
		value := result.CompletedAt.UTC().Format(time.RFC3339)
		completedAt = &value
	}
	return httptransport.RequestClippingExportResponse{
		ExportID:            result.ExportID,
		ProjectID:           result.ProjectID,
		Status:              result.Status,
		ProgressPercent:     result.ProgressPercent,
		OutputURL:           result.OutputURL,
		ProviderJobID:       result.ProviderJobID,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		CompletedAt:         completedAt,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) DeployAutoClippingModelHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.DeployAutoClippingModelRequest,
) (httptransport.DeployAutoClippingModelResponse, error) {
	result, err := h.Service.DeployAutoClippingModel(ctx, idempotencyKey, application.DeployAutoClippingModelInput{
		ActorID:          adminID,
		ModelName:        req.ModelName,
		VersionTag:       req.VersionTag,
		ModelArtifactKey: req.ModelArtifactKey,
		CanaryPercentage: req.CanaryPercentage,
		Description:      req.Description,
		Reason:           req.Reason,
		SourceIP:         req.SourceIP,
		CorrelationID:    req.CorrelationID,
	})
	if err != nil {
		return httptransport.DeployAutoClippingModelResponse{}, err
	}
	return httptransport.DeployAutoClippingModelResponse{
		ModelVersionID:      result.ModelVersionID,
		DeploymentStatus:    result.DeploymentStatus,
		DeployedAt:          result.DeployedAt.UTC().Format(time.RFC3339),
		Message:             result.Message,
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) RotateIntegrationKeyHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RotateIntegrationKeyRequest,
) (httptransport.RotateIntegrationKeyResponse, error) {
	result, err := h.Service.RotateIntegrationKey(ctx, idempotencyKey, application.RotateIntegrationKeyInput{
		ActorID:       adminID,
		KeyID:         req.KeyID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RotateIntegrationKeyResponse{}, err
	}
	return httptransport.RotateIntegrationKeyResponse{
		RotationID:          result.RotationID,
		DeveloperID:         result.DeveloperID,
		OldKeyID:            result.OldKeyID,
		NewKeyID:            result.NewKeyID,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) TestIntegrationWorkflowHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.TestIntegrationWorkflowRequest,
) (httptransport.TestIntegrationWorkflowResponse, error) {
	result, err := h.Service.TestIntegrationWorkflow(ctx, idempotencyKey, application.TestIntegrationWorkflowInput{
		ActorID:       adminID,
		WorkflowID:    req.WorkflowID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.TestIntegrationWorkflowResponse{}, err
	}
	return httptransport.TestIntegrationWorkflowResponse{
		ExecutionID:         result.ExecutionID,
		WorkflowID:          result.WorkflowID,
		Status:              result.Status,
		TestRun:             result.TestRun,
		StartedAt:           result.StartedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) ReplayWebhookHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.ReplayWebhookRequest,
) (httptransport.ReplayWebhookResponse, error) {
	result, err := h.Service.ReplayWebhook(ctx, idempotencyKey, application.ReplayWebhookInput{
		ActorID:       adminID,
		WebhookID:     req.WebhookID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ReplayWebhookResponse{}, err
	}
	return httptransport.ReplayWebhookResponse{
		DeliveryID:          result.DeliveryID,
		WebhookID:           result.WebhookID,
		Status:              result.Status,
		HTTPStatus:          result.HTTPStatus,
		LatencyMS:           result.LatencyMS,
		Timestamp:           result.Timestamp.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) DisableWebhookHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.DisableWebhookRequest,
) (httptransport.DisableWebhookResponse, error) {
	result, err := h.Service.DisableWebhook(ctx, idempotencyKey, application.DisableWebhookInput{
		ActorID:       adminID,
		WebhookID:     req.WebhookID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.DisableWebhookResponse{}, err
	}
	return httptransport.DisableWebhookResponse{
		WebhookID:           result.WebhookID,
		Status:              result.Status,
		UpdatedAt:           result.UpdatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) GetWebhookDeliveriesHandler(
	ctx context.Context,
	adminID string,
	req httptransport.GetWebhookDeliveriesRequest,
) (httptransport.GetWebhookDeliveriesResponse, error) {
	rows, err := h.Service.GetWebhookDeliveries(ctx, application.GetWebhookDeliveriesInput{
		ActorID:   adminID,
		WebhookID: req.WebhookID,
		Limit:     req.Limit,
	})
	if err != nil {
		return httptransport.GetWebhookDeliveriesResponse{}, err
	}
	out := httptransport.GetWebhookDeliveriesResponse{
		Deliveries: make([]httptransport.WebhookDeliveryResponse, 0, len(rows)),
	}
	for _, row := range rows {
		out.Deliveries = append(out.Deliveries, httptransport.WebhookDeliveryResponse{
			DeliveryID:      row.DeliveryID,
			WebhookID:       row.WebhookID,
			OriginalEventID: row.OriginalEventID,
			OriginalType:    row.OriginalType,
			HTTPStatus:      row.HTTPStatus,
			LatencyMS:       row.LatencyMS,
			RetryCount:      row.RetryCount,
			DeliveredAt:     row.DeliveredAt.UTC().Format(time.RFC3339),
			IsTest:          row.IsTest,
			Success:         row.Success,
		})
	}
	return out, nil
}

func (h Handler) GetWebhookAnalyticsHandler(
	ctx context.Context,
	adminID string,
	req httptransport.GetWebhookAnalyticsRequest,
) (httptransport.GetWebhookAnalyticsResponse, error) {
	row, err := h.Service.GetWebhookAnalytics(ctx, application.GetWebhookAnalyticsInput{
		ActorID:   adminID,
		WebhookID: req.WebhookID,
	})
	if err != nil {
		return httptransport.GetWebhookAnalyticsResponse{}, err
	}
	byType := make(map[string]httptransport.WebhookAnalyticsMetrics, len(row.ByEventType))
	for key, value := range row.ByEventType {
		byType[key] = httptransport.WebhookAnalyticsMetrics{
			Total:      value.Total,
			Success:    value.Success,
			Failed:     value.Failed,
			AvgLatency: value.AvgLatency,
		}
	}
	return httptransport.GetWebhookAnalyticsResponse{
		TotalDeliveries:      row.TotalDeliveries,
		SuccessfulDeliveries: row.SuccessfulDeliveries,
		FailedDeliveries:     row.FailedDeliveries,
		SuccessRate:          row.SuccessRate,
		AvgLatencyMS:         row.AvgLatencyMS,
		P95LatencyMS:         row.P95LatencyMS,
		P99LatencyMS:         row.P99LatencyMS,
		ByEventType:          byType,
	}, nil
}

func (h Handler) CreateMigrationPlanHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.CreateMigrationPlanRequest,
) (httptransport.CreateMigrationPlanResponse, error) {
	result, err := h.Service.CreateMigrationPlan(ctx, idempotencyKey, application.CreateMigrationPlanInput{
		ActorID:       adminID,
		ServiceName:   req.ServiceName,
		Environment:   req.Environment,
		Version:       req.Version,
		Plan:          req.Plan,
		DryRun:        req.DryRun,
		RiskLevel:     req.RiskLevel,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.CreateMigrationPlanResponse{}, err
	}
	return httptransport.CreateMigrationPlanResponse{
		PlanID:              result.PlanID,
		ServiceName:         result.ServiceName,
		Environment:         result.Environment,
		Version:             result.Version,
		Plan:                result.Plan,
		Status:              result.Status,
		DryRun:              result.DryRun,
		RiskLevel:           result.RiskLevel,
		StagingValidated:    result.StagingValidated,
		BackupRequired:      result.BackupRequired,
		CreatedBy:           result.CreatedBy,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:           result.UpdatedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func (h Handler) ListMigrationPlansHandler(
	ctx context.Context,
	adminID string,
) (httptransport.ListMigrationPlansResponse, error) {
	rows, err := h.Service.ListMigrationPlans(ctx, application.ListMigrationPlansInput{ActorID: adminID})
	if err != nil {
		return httptransport.ListMigrationPlansResponse{}, err
	}
	out := httptransport.ListMigrationPlansResponse{
		Plans: make([]httptransport.MigrationPlanResponse, 0, len(rows)),
	}
	for _, row := range rows {
		out.Plans = append(out.Plans, httptransport.MigrationPlanResponse{
			PlanID:           row.PlanID,
			ServiceName:      row.ServiceName,
			Environment:      row.Environment,
			Version:          row.Version,
			Plan:             row.Plan,
			Status:           row.Status,
			DryRun:           row.DryRun,
			RiskLevel:        row.RiskLevel,
			StagingValidated: row.StagingValidated,
			BackupRequired:   row.BackupRequired,
			CreatedBy:        row.CreatedBy,
			CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:        row.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, nil
}

func (h Handler) StartMigrationRunHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.StartMigrationRunRequest,
) (httptransport.StartMigrationRunResponse, error) {
	result, err := h.Service.StartMigrationRun(ctx, idempotencyKey, application.StartMigrationRunInput{
		ActorID:       adminID,
		PlanID:        req.PlanID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.StartMigrationRunResponse{}, err
	}
	return httptransport.StartMigrationRunResponse{
		RunID:               result.RunID,
		PlanID:              result.PlanID,
		Status:              result.Status,
		OperatorID:          result.OperatorID,
		SnapshotCreated:     result.SnapshotCreated,
		RollbackAvailable:   result.RollbackAvailable,
		ValidationStatus:    result.ValidationStatus,
		BackfillJobID:       result.BackfillJobID,
		StartedAt:           result.StartedAt.UTC().Format(time.RFC3339),
		CompletedAt:         result.CompletedAt.UTC().Format(time.RFC3339),
		ControlPlaneAuditID: result.ControlPlaneAuditID,
	}, nil
}

func toSupportTicketResponse(result application.SupportTicketResult) httptransport.SupportTicketResponse {
	return httptransport.SupportTicketResponse{
		TicketID:         result.TicketID,
		UserID:           result.UserID,
		Subject:          result.Subject,
		Description:      result.Description,
		Category:         result.Category,
		Priority:         result.Priority,
		Status:           result.Status,
		SubStatus:        result.SubStatus,
		AssignedAgentID:  result.AssignedAgentID,
		SLAResponseDueAt: result.SLAResponseDueAt.UTC().Format(time.RFC3339),
		LastActivityAt:   result.LastActivityAt.UTC().Format(time.RFC3339),
		UpdatedAt:        result.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
