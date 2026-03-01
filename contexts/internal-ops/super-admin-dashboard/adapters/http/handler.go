package httpadapter

import (
	"context"
	"log/slog"
	"time"

	"solomon/contexts/internal-ops/super-admin-dashboard/application"
	httptransport "solomon/contexts/internal-ops/super-admin-dashboard/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) StartImpersonationHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.StartImpersonationRequest,
) (httptransport.StartImpersonationResponse, error) {
	result, err := h.Service.StartImpersonation(ctx, adminID, idempotencyKey, req.ImpersonatedUserID, req.Reason)
	if err != nil {
		return httptransport.StartImpersonationResponse{}, err
	}
	return httptransport.StartImpersonationResponse{
		ImpersonationID: result.ImpersonationID,
		UserID:          result.UserID,
		AccessToken:     result.AccessToken,
		TokenExpiresAt:  result.TokenExpiresAt.UTC().Format(time.RFC3339),
		StartedAt:       result.StartedAt.UTC().Format(time.RFC3339),
		Status:          result.Status,
	}, nil
}

func (h Handler) EndImpersonationHandler(
	ctx context.Context,
	idempotencyKey string,
	req httptransport.EndImpersonationRequest,
) (httptransport.EndImpersonationResponse, error) {
	result, err := h.Service.EndImpersonation(ctx, idempotencyKey, req.ImpersonationID)
	if err != nil {
		return httptransport.EndImpersonationResponse{}, err
	}
	resp := httptransport.EndImpersonationResponse{Status: result.Status}
	if result.EndedAt != nil {
		resp.EndedAt = result.EndedAt.UTC().Format(time.RFC3339)
		resp.ActivitySummary.DurationMinutes = int(result.EndedAt.Sub(result.StartedAt).Minutes())
	}
	return resp, nil
}

func (h Handler) AdjustWalletHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	req httptransport.WalletAdjustRequest,
) (httptransport.WalletAdjustResponse, error) {
	result, err := h.Service.AdjustWallet(ctx, adminID, idempotencyKey, userID, req.Amount, req.AdjustmentType, req.Reason)
	if err != nil {
		return httptransport.WalletAdjustResponse{}, err
	}
	return httptransport.WalletAdjustResponse{
		AdjustmentID:  result.AdjustmentID,
		UserID:        result.UserID,
		Amount:        result.Amount,
		BalanceBefore: result.BalanceBefore,
		BalanceAfter:  result.BalanceAfter,
		AdjustedAt:    result.AdjustedAt.UTC().Format(time.RFC3339),
		AuditLogID:    result.AuditLogID,
	}, nil
}

func (h Handler) WalletHistoryHandler(
	ctx context.Context,
	userID string,
	cursor string,
	limit int,
) (httptransport.WalletHistoryResponse, error) {
	items, next, err := h.Service.ListWalletHistory(ctx, userID, cursor, limit)
	if err != nil {
		return httptransport.WalletHistoryResponse{}, err
	}
	resp := httptransport.WalletHistoryResponse{}
	resp.Adjustments = make([]httptransport.WalletHistoryEntry, 0, len(items))
	for _, item := range items {
		resp.Adjustments = append(resp.Adjustments, httptransport.WalletHistoryEntry{
			AdjustmentID: item.AdjustmentID,
			Amount:       item.Amount,
			Type:         item.Type,
			Reason:       item.Reason,
			AdminID:      item.AdminID,
			AdjustedAt:   item.AdjustedAt.UTC().Format(time.RFC3339),
		})
	}
	resp.Pagination.Cursor = next
	resp.Pagination.HasMore = next != ""
	resp.Pagination.PageSize = limit
	return resp, nil
}

func (h Handler) BanUserHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	req httptransport.BanUserRequest,
) (httptransport.BanUserResponse, error) {
	result, err := h.Service.BanUser(ctx, adminID, idempotencyKey, userID, req.BanType, req.DurationDays, req.Reason)
	if err != nil {
		return httptransport.BanUserResponse{}, err
	}
	resp := httptransport.BanUserResponse{
		BanID:              result.BanID,
		UserID:             result.UserID,
		BanType:            result.BanType,
		BannedAt:           result.BannedAt.UTC().Format(time.RFC3339),
		AllSessionsRevoked: result.AllSessionsRevoked,
		AuditLogID:         result.AuditLogID,
	}
	if result.ExpiresAt != nil {
		resp.ExpiresAt = result.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) UnbanUserHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	req httptransport.UnbanUserRequest,
) (httptransport.UnbanUserResponse, error) {
	result, err := h.Service.UnbanUser(ctx, adminID, idempotencyKey, userID, req.Reason)
	if err != nil {
		return httptransport.UnbanUserResponse{}, err
	}
	unbannedAt := time.Now().UTC()
	if result.ExpiresAt != nil {
		unbannedAt = result.ExpiresAt.UTC()
	}
	return httptransport.UnbanUserResponse{
		UserID:     userID,
		UnbannedAt: unbannedAt.Format(time.RFC3339),
		Status:     "active",
	}, nil
}

func (h Handler) SearchUsersHandler(
	ctx context.Context,
	query string,
	status string,
	cursor string,
	pageSize int,
) (httptransport.UserSearchResponse, error) {
	items, next, total, err := h.Service.SearchUsers(ctx, query, status, cursor, pageSize)
	if err != nil {
		return httptransport.UserSearchResponse{}, err
	}
	resp := httptransport.UserSearchResponse{}
	resp.Users = make([]struct {
		UserID        string  `json:"user_id"`
		Email         string  `json:"email"`
		Username      string  `json:"username"`
		Role          string  `json:"role"`
		CreatedAt     string  `json:"created_at"`
		TotalEarnings float64 `json:"total_earnings"`
		Status        string  `json:"status"`
		KYCStatus     string  `json:"kyc_status"`
		LastLoginAt   string  `json:"last_login_at,omitempty"`
	}, 0, len(items))
	for _, item := range items {
		entry := struct {
			UserID        string  `json:"user_id"`
			Email         string  `json:"email"`
			Username      string  `json:"username"`
			Role          string  `json:"role"`
			CreatedAt     string  `json:"created_at"`
			TotalEarnings float64 `json:"total_earnings"`
			Status        string  `json:"status"`
			KYCStatus     string  `json:"kyc_status"`
			LastLoginAt   string  `json:"last_login_at,omitempty"`
		}{
			UserID:        item.UserID,
			Email:         item.Email,
			Username:      item.Username,
			Role:          item.Role,
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
			TotalEarnings: item.TotalEarnings,
			Status:        item.Status,
			KYCStatus:     item.KYCStatus,
		}
		if item.LastLoginAt != nil {
			entry.LastLoginAt = item.LastLoginAt.UTC().Format(time.RFC3339)
		}
		resp.Users = append(resp.Users, entry)
	}
	resp.Pagination.Cursor = next
	resp.Pagination.HasMore = next != ""
	resp.Pagination.TotalCount = total
	return resp, nil
}

func (h Handler) BulkActionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.BulkActionRequest,
) (httptransport.BulkActionResponse, error) {
	result, err := h.Service.BulkAction(ctx, adminID, idempotencyKey, req.UserIDs, req.Action)
	if err != nil {
		return httptransport.BulkActionResponse{}, err
	}
	return httptransport.BulkActionResponse{
		JobID:                   result.JobID,
		Action:                  result.Action,
		UserCount:               result.UserCount,
		Status:                  result.Status,
		CreatedAt:               result.CreatedAt.UTC().Format(time.RFC3339),
		EstimatedCompletionTime: result.EstimatedCompletionTime.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) PauseCampaignHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	campaignID string,
	req httptransport.PauseCampaignRequest,
) (httptransport.PauseCampaignResponse, error) {
	result, err := h.Service.PauseCampaign(ctx, adminID, idempotencyKey, campaignID, req.Reason)
	if err != nil {
		return httptransport.PauseCampaignResponse{}, err
	}
	return httptransport.PauseCampaignResponse{
		CampaignID: result.CampaignID,
		Status:     result.Status,
		PausedAt:   result.PausedAt.UTC().Format(time.RFC3339),
		AuditLogID: result.AuditLogID,
	}, nil
}

func (h Handler) AdjustCampaignHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	campaignID string,
	req httptransport.AdjustCampaignRequest,
) (httptransport.AdjustCampaignResponse, error) {
	result, err := h.Service.AdjustCampaign(ctx, adminID, idempotencyKey, campaignID, req.NewBudget, req.NewRatePer1kViews, req.Reason)
	if err != nil {
		return httptransport.AdjustCampaignResponse{}, err
	}
	return httptransport.AdjustCampaignResponse{
		CampaignID:        result.CampaignID,
		OldBudget:         result.OldBudget,
		NewBudget:         result.NewBudget,
		OldRatePer1kViews: result.OldRatePer1kViews,
		NewRatePer1kViews: result.NewRatePer1kViews,
		AdjustedAt:        result.AdjustedAt.UTC().Format(time.RFC3339),
		AuditLogID:        result.AuditLogID,
	}, nil
}

func (h Handler) OverrideSubmissionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	submissionID string,
	req httptransport.OverrideSubmissionRequest,
) (httptransport.OverrideSubmissionResponse, error) {
	result, err := h.Service.OverrideSubmission(ctx, adminID, idempotencyKey, submissionID, req.NewStatus, req.OverrideReason)
	if err != nil {
		return httptransport.OverrideSubmissionResponse{}, err
	}
	return httptransport.OverrideSubmissionResponse{
		SubmissionID: result.SubmissionID,
		OldStatus:    result.OldStatus,
		NewStatus:    result.NewStatus,
		OverriddenAt: result.OverriddenAt.UTC().Format(time.RFC3339),
		AuditLogID:   result.AuditLogID,
	}, nil
}

func (h Handler) ListFeatureFlagsHandler(ctx context.Context) (httptransport.FeatureFlagsResponse, error) {
	items, err := h.Service.ListFeatureFlags(ctx)
	if err != nil {
		return httptransport.FeatureFlagsResponse{}, err
	}
	resp := httptransport.FeatureFlagsResponse{Flags: make([]httptransport.FeatureFlagDTO, 0, len(items))}
	for _, item := range items {
		resp.Flags = append(resp.Flags, httptransport.FeatureFlagDTO{
			FlagKey:   item.FlagKey,
			Enabled:   item.Enabled,
			Config:    item.Config,
			UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
			UpdatedBy: item.UpdatedBy,
		})
	}
	return resp, nil
}

func (h Handler) ToggleFeatureFlagHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	flagKey string,
	req httptransport.ToggleFeatureFlagRequest,
) (httptransport.ToggleFeatureFlagResponse, error) {
	flag, oldEnabled, err := h.Service.ToggleFeatureFlag(ctx, adminID, idempotencyKey, flagKey, req.Enabled, req.Reason, req.Config)
	if err != nil {
		return httptransport.ToggleFeatureFlagResponse{}, err
	}
	return httptransport.ToggleFeatureFlagResponse{
		FlagKey:              flag.FlagKey,
		OldEnabled:           oldEnabled,
		NewEnabled:           flag.Enabled,
		UpdatedAt:            flag.UpdatedAt.UTC().Format(time.RFC3339),
		PropagatedToServices: []string{"api-gateway", "web-app"},
	}, nil
}

func (h Handler) AnalyticsDashboardHandler(ctx context.Context, start time.Time, end time.Time) (httptransport.AnalyticsDashboardResponse, error) {
	result, err := h.Service.GetAnalyticsDashboard(ctx, start, end)
	if err != nil {
		return httptransport.AnalyticsDashboardResponse{}, err
	}
	resp := httptransport.AnalyticsDashboardResponse{}
	resp.DateRange.Start = result.DateRangeStart.UTC().Format(time.RFC3339)
	resp.DateRange.End = result.DateRangeEnd.UTC().Format(time.RFC3339)
	resp.Metrics.TotalRevenue = result.TotalRevenue
	resp.Metrics.UserGrowth = result.UserGrowth
	resp.Metrics.CampaignCount = result.CampaignCount
	resp.Metrics.FraudMetrics = result.FraudAlerts
	return resp, nil
}

func (h Handler) AuditLogsHandler(
	ctx context.Context,
	adminID string,
	actionType string,
	cursor string,
	pageSize int,
) (httptransport.AuditLogsResponse, error) {
	items, next, err := h.Service.ListAuditLogs(ctx, adminID, actionType, cursor, pageSize)
	if err != nil {
		return httptransport.AuditLogsResponse{}, err
	}
	resp := httptransport.AuditLogsResponse{AuditLogs: make([]httptransport.AuditLogDTO, 0, len(items))}
	for _, item := range items {
		resp.AuditLogs = append(resp.AuditLogs, httptransport.AuditLogDTO{
			AuditID:            item.AuditID,
			AdminID:            item.AdminID,
			ActionType:         item.ActionType,
			TargetResourceID:   item.TargetResourceID,
			TargetResourceType: item.TargetResourceType,
			OldValue:           item.OldValue,
			NewValue:           item.NewValue,
			Reason:             item.Reason,
			PerformedAt:        item.PerformedAt.UTC().Format(time.RFC3339),
			IPAddress:          item.IPAddress,
			SignatureHash:      item.SignatureHash,
			IsVerified:         item.IsVerified,
		})
	}
	resp.Pagination.Cursor = next
	resp.Pagination.HasMore = next != ""
	return resp, nil
}

func (h Handler) ExportAuditLogsHandler(
	ctx context.Context,
	format string,
	start time.Time,
	end time.Time,
	includeSignatures bool,
) (httptransport.AuditLogExportResponse, error) {
	result, err := h.Service.ExportAuditLogs(ctx, format, start, end, includeSignatures)
	if err != nil {
		return httptransport.AuditLogExportResponse{}, err
	}
	return httptransport.AuditLogExportResponse{
		ExportJobID:         result.ExportJobID,
		Status:              result.Status,
		FileURL:             result.FileURL,
		CreatedAt:           result.CreatedAt.UTC().Format(time.RFC3339),
		EstimatedCompletion: result.EstimatedCompletion.UTC().Format(time.RFC3339),
	}, nil
}