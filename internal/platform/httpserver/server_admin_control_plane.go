package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	clippingtoolservice "solomon/contexts/campaign-editorial/clipping-tool-service"
	clippingerrors "solomon/contexts/campaign-editorial/clipping-tool-service/domain/errors"
	clippinghttp "solomon/contexts/campaign-editorial/clipping-tool-service/transport/http"
	editordashboardservice "solomon/contexts/campaign-editorial/editor-dashboard-service"
	editordashboarderrors "solomon/contexts/campaign-editorial/editor-dashboard-service/domain/errors"
	authorization "solomon/contexts/identity-access/authorization-service"
	authzerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	authzhttp "solomon/contexts/identity-access/authorization-service/transport/http"
	admindashboardservice "solomon/contexts/internal-ops/admin-dashboard-service"
	admindashboardmemory "solomon/contexts/internal-ops/admin-dashboard-service/adapters/memory"
	admindashboarderrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	admindashboardports "solomon/contexts/internal-ops/admin-dashboard-service/ports"
	admindashboardhttp "solomon/contexts/internal-ops/admin-dashboard-service/transport/http"
	abusepreventionservice "solomon/contexts/moderation-safety/abuse-prevention-service"
	abuseerrors "solomon/contexts/moderation-safety/abuse-prevention-service/domain/errors"
	abusehttp "solomon/contexts/moderation-safety/abuse-prevention-service/transport/http"
	moderationservice "solomon/contexts/moderation-safety/moderation-service"
	moderationerrors "solomon/contexts/moderation-safety/moderation-service/domain/errors"
	moderationhttp "solomon/contexts/moderation-safety/moderation-service/transport/http"
)

type meshSuccessEnvelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type meshErrorEnvelope struct {
	Status string `json:"status"`
	Error  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type meshTopLevelErrorEnvelope struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	adminRuntimeModeEnv  = "ADMIN_RUNTIME_MODE"
	adminM39BaseURLEnv   = "ADMIN_M39_BASE_URL"
	adminM14BaseURLEnv   = "ADMIN_M14_BASE_URL"
	adminM44BaseURLEnv   = "ADMIN_M44_BASE_URL"
	adminM05BaseURLEnv   = "ADMIN_M05_BASE_URL"
	adminM41BaseURLEnv   = "ADMIN_M41_BASE_URL"
	adminM89BaseURLEnv   = "ADMIN_M89_BASE_URL"
	adminM50BaseURLEnv   = "ADMIN_M50_BASE_URL"
	adminM51BaseURLEnv   = "ADMIN_M51_BASE_URL"
	adminM68BaseURLEnv   = "ADMIN_M68_BASE_URL"
	adminM69BaseURLEnv   = "ADMIN_M69_BASE_URL"
	adminM73BaseURLEnv   = "ADMIN_M73_BASE_URL"
	adminM25BaseURLEnv   = "ADMIN_M25_BASE_URL"
	adminM70BaseURLEnv   = "ADMIN_M70_BASE_URL"
	adminM71BaseURLEnv   = "ADMIN_M71_BASE_URL"
	adminM72BaseURLEnv   = "ADMIN_M72_BASE_URL"
	adminM84BaseURLEnv   = "ADMIN_M84_BASE_URL"
	ownerRequestTimeout  = 2 * time.Second
	ownerRetryMaxAttempt = 3
	ownerRetryBackoff    = 100 * time.Millisecond
)

type adminControlPlaneRuntime struct {
	mode          string
	allowFallback bool
}

type adminOwnerClientConfig struct {
	runtime    adminControlPlaneRuntime
	m39BaseURL string
	m14BaseURL string
	m44BaseURL string
	m05BaseURL string
	m41BaseURL string
	m89BaseURL string
	m50BaseURL string
	m51BaseURL string
	m68BaseURL string
	m69BaseURL string
	m73BaseURL string
	m25BaseURL string
	m70BaseURL string
	m71BaseURL string
	m72BaseURL string
	m84BaseURL string
}

type controlPlaneAuthorizationClient struct {
	module authorization.Module
}

func (c controlPlaneAuthorizationClient) GrantRole(
	ctx context.Context,
	adminID string,
	userID string,
	roleID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.RoleGrantResult, error) {
	resp, err := c.module.Handler.GrantRoleHandler(ctx, userID, adminID, idempotencyKey, authzhttp.GrantRoleRequest{
		RoleID: roleID,
		Reason: reason,
	})
	if err != nil {
		return admindashboardports.RoleGrantResult{}, err
	}
	return admindashboardports.RoleGrantResult{
		AssignmentID: resp.AssignmentID,
		UserID:       resp.UserID,
		RoleID:       resp.RoleID,
		AuditLogID:   resp.AuditLogID,
	}, nil
}

type controlPlaneModerationClient struct {
	module moderationservice.Module
}

func (c controlPlaneModerationClient) ApproveSubmission(
	ctx context.Context,
	moderatorID string,
	submissionID string,
	campaignID string,
	reason string,
	notes string,
	idempotencyKey string,
) (admindashboardports.ModerationDecisionResult, error) {
	resp, err := c.module.Handler.ApproveHandler(ctx, idempotencyKey, moderatorID, moderationhttp.ApproveRequest{
		SubmissionID: submissionID,
		CampaignID:   campaignID,
		Reason:       reason,
		Notes:        notes,
	})
	if err != nil {
		return admindashboardports.ModerationDecisionResult{}, err
	}
	createdAt, _ := time.Parse(time.RFC3339, resp.Data.CreatedAt)
	return admindashboardports.ModerationDecisionResult{
		DecisionID:   resp.Data.DecisionID,
		SubmissionID: resp.Data.SubmissionID,
		CampaignID:   resp.Data.CampaignID,
		ModeratorID:  resp.Data.ModeratorID,
		Action:       resp.Data.Action,
		Reason:       resp.Data.Reason,
		Notes:        resp.Data.Notes,
		QueueStatus:  resp.Data.QueueStatus,
		CreatedAt:    createdAt,
	}, nil
}

func (c controlPlaneModerationClient) RejectSubmission(
	ctx context.Context,
	moderatorID string,
	submissionID string,
	campaignID string,
	reason string,
	notes string,
	idempotencyKey string,
) (admindashboardports.ModerationDecisionResult, error) {
	resp, err := c.module.Handler.RejectHandler(ctx, idempotencyKey, moderatorID, moderationhttp.RejectRequest{
		SubmissionID:    submissionID,
		CampaignID:      campaignID,
		RejectionReason: reason,
		RejectionNotes:  notes,
	})
	if err != nil {
		return admindashboardports.ModerationDecisionResult{}, err
	}
	createdAt, _ := time.Parse(time.RFC3339, resp.Data.CreatedAt)
	return admindashboardports.ModerationDecisionResult{
		DecisionID:   resp.Data.DecisionID,
		SubmissionID: resp.Data.SubmissionID,
		CampaignID:   resp.Data.CampaignID,
		ModeratorID:  resp.Data.ModeratorID,
		Action:       resp.Data.Action,
		Reason:       resp.Data.Reason,
		Notes:        resp.Data.Notes,
		QueueStatus:  resp.Data.QueueStatus,
		CreatedAt:    createdAt,
	}, nil
}

type controlPlaneAbusePreventionClient struct {
	module abusepreventionservice.Module
}

func (c controlPlaneAbusePreventionClient) ReleaseLockout(
	ctx context.Context,
	adminID string,
	userID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.AbuseLockoutResult, error) {
	resp, err := c.module.Handler.ReleaseLockoutHandler(
		ctx,
		adminID,
		idempotencyKey,
		userID,
		abusehttp.ReleaseLockoutRequest{
			Reason: reason,
		},
	)
	if err != nil {
		return admindashboardports.AbuseLockoutResult{}, err
	}
	releasedAt, _ := time.Parse(time.RFC3339, resp.ReleasedAt)
	return admindashboardports.AbuseLockoutResult{
		ThreatID:        resp.ThreatID,
		UserID:          resp.UserID,
		Status:          resp.Status,
		ReleasedAt:      releasedAt,
		OwnerAuditLogID: resp.OwnerAuditLogID,
	}, nil
}

type controlPlaneEditorWorkflowClient struct {
	module editordashboardservice.Module
}

func (c controlPlaneEditorWorkflowClient) SaveCampaign(
	ctx context.Context,
	_ string,
	editorID string,
	campaignID string,
	idempotencyKey string,
) (admindashboardports.EditorCampaignSaveResult, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
	defer cancel()
	resp, err := c.module.Handler.SaveCampaignHandler(attemptCtx, idempotencyKey, editorID, campaignID)
	if err != nil {
		return admindashboardports.EditorCampaignSaveResult{}, mapEditorWorkflowError(err)
	}
	var savedAt *time.Time
	if strings.TrimSpace(resp.Data.SavedAt) != "" {
		parsed := parseRFC3339OrZero(resp.Data.SavedAt)
		if !parsed.IsZero() {
			savedAt = &parsed
		}
	}
	return admindashboardports.EditorCampaignSaveResult{
		CampaignID: resp.Data.CampaignID,
		Saved:      resp.Data.Saved,
		SavedAt:    savedAt,
	}, nil
}

type controlPlaneClippingWorkflowClient struct {
	module clippingtoolservice.Module
}

func (c controlPlaneClippingWorkflowClient) RequestExport(
	ctx context.Context,
	_ string,
	userID string,
	projectID string,
	format string,
	resolution string,
	fps int,
	bitrate string,
	campaignID string,
	idempotencyKey string,
) (admindashboardports.ClippingExportResult, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
	defer cancel()
	resp, err := c.module.Handler.RequestExportHandler(attemptCtx, idempotencyKey, userID, projectID, clippinghttp.ExportRequest{
		Format:     format,
		Resolution: resolution,
		FPS:        fps,
		Bitrate:    bitrate,
		CampaignID: campaignID,
	})
	if err != nil {
		return admindashboardports.ClippingExportResult{}, mapClippingWorkflowError(err)
	}
	var completedAt *time.Time
	if strings.TrimSpace(resp.Data.CompletedAt) != "" {
		parsed := parseRFC3339OrZero(resp.Data.CompletedAt)
		if !parsed.IsZero() {
			completedAt = &parsed
		}
	}
	return admindashboardports.ClippingExportResult{
		ExportID:        resp.Data.ExportID,
		ProjectID:       resp.Data.ProjectID,
		Status:          resp.Data.Status,
		ProgressPercent: resp.Data.ProgressPercent,
		OutputURL:       resp.Data.OutputURL,
		ProviderJobID:   resp.Data.ProviderJobID,
		CreatedAt:       parseRFC3339OrZero(resp.Data.CreatedAt),
		CompletedAt:     completedAt,
	}, nil
}

type controlPlaneAutoClippingClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.AutoClippingClient
}

func (c controlPlaneAutoClippingClient) DeployModel(
	ctx context.Context,
	adminID string,
	input admindashboardports.AutoClippingModelDeployInput,
	idempotencyKey string,
) (admindashboardports.AutoClippingModelDeployResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.AutoClippingModelDeployResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.DeployModel(ctx, adminID, input, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/models/deploy",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"model_name":         input.ModelName,
			"version_tag":        input.VersionTag,
			"model_artifact_key": input.ModelArtifactKey,
			"canary_percentage":  input.CanaryPercentage,
			"description":        input.Description,
			"reason":             input.Reason,
		},
	)
	if err != nil {
		return admindashboardports.AutoClippingModelDeployResult{}, err
	}

	var payload struct {
		ModelVersionID   string `json:"model_version_id"`
		DeploymentStatus string `json:"deployment_status"`
		DeployedAt       string `json:"deployed_at"`
		Message          string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.AutoClippingModelDeployResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	deployedAt := parseRFC3339OrZero(payload.DeployedAt)
	return admindashboardports.AutoClippingModelDeployResult{
		ModelVersionID:   payload.ModelVersionID,
		DeploymentStatus: payload.DeploymentStatus,
		DeployedAt:       deployedAt,
		Message:          payload.Message,
	}, nil
}

func mapEditorWorkflowError(err error) error {
	switch {
	case errors.Is(err, editordashboarderrors.ErrInvalidRequest),
		errors.Is(err, editordashboarderrors.ErrIdempotencyKeyRequired):
		return admindashboarderrors.ErrInvalidInput
	case errors.Is(err, editordashboarderrors.ErrNotFound):
		return admindashboarderrors.ErrNotFound
	case errors.Is(err, editordashboarderrors.ErrForbidden):
		return admindashboarderrors.ErrUnauthorized
	case errors.Is(err, editordashboarderrors.ErrIdempotencyConflict):
		return admindashboarderrors.ErrIdempotencyConflict
	case errors.Is(err, editordashboarderrors.ErrDependencyUnavailable):
		return admindashboarderrors.ErrDependencyUnavailable
	default:
		return admindashboarderrors.ErrDependencyUnavailable
	}
}

func mapClippingWorkflowError(err error) error {
	switch {
	case errors.Is(err, clippingerrors.ErrInvalidRequest),
		errors.Is(err, clippingerrors.ErrIdempotencyKeyRequired):
		return admindashboarderrors.ErrInvalidInput
	case errors.Is(err, clippingerrors.ErrNotFound):
		return admindashboarderrors.ErrNotFound
	case errors.Is(err, clippingerrors.ErrForbidden):
		return admindashboarderrors.ErrUnauthorized
	case errors.Is(err, clippingerrors.ErrConflict),
		errors.Is(err, clippingerrors.ErrProjectExporting):
		return admindashboarderrors.ErrConflict
	case errors.Is(err, clippingerrors.ErrIdempotencyConflict):
		return admindashboarderrors.ErrIdempotencyConflict
	case errors.Is(err, clippingerrors.ErrDependencyUnavailable):
		return admindashboarderrors.ErrDependencyUnavailable
	default:
		return admindashboarderrors.ErrDependencyUnavailable
	}
}

type controlPlaneFinanceClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.FinanceClient
}

func (c controlPlaneFinanceClient) CreateRefund(
	ctx context.Context,
	adminID string,
	transactionID string,
	userID string,
	amount float64,
	reason string,
	idempotencyKey string,
) (admindashboardports.FinanceRefundResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.FinanceRefundResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateRefund(ctx, adminID, transactionID, userID, amount, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/transactions/"+url.PathEscape(transactionID)+"/refund",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
			"reason":  reason,
		},
	)
	if err != nil {
		return admindashboardports.FinanceRefundResult{}, err
	}

	var payload struct {
		RefundID      string  `json:"refund_id"`
		TransactionID string  `json:"transaction_id"`
		UserID        string  `json:"user_id"`
		Amount        float64 `json:"amount"`
		Reason        string  `json:"reason"`
		CreatedAt     string  `json:"created_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.FinanceRefundResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	createdAt, _ := time.Parse(time.RFC3339, payload.CreatedAt)
	return admindashboardports.FinanceRefundResult{
		RefundID:      payload.RefundID,
		TransactionID: payload.TransactionID,
		UserID:        payload.UserID,
		Amount:        payload.Amount,
		Reason:        payload.Reason,
		CreatedAt:     createdAt,
	}, nil
}

type controlPlaneBillingClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.BillingClient
}

func (c controlPlaneBillingClient) CreateInvoiceRefund(
	ctx context.Context,
	adminID string,
	invoiceID string,
	lineItemID string,
	amount float64,
	reason string,
	idempotencyKey string,
) (admindashboardports.BillingRefundResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.BillingRefundResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateInvoiceRefund(ctx, adminID, invoiceID, lineItemID, amount, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/invoices/"+url.PathEscape(invoiceID)+"/refund",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"line_item_id": lineItemID,
			"amount":       amount,
			"reason":       reason,
		},
	)
	if err != nil {
		return admindashboardports.BillingRefundResult{}, err
	}

	var payload struct {
		RefundID    string  `json:"refund_id"`
		InvoiceID   string  `json:"invoice_id"`
		LineItemID  string  `json:"line_item_id"`
		Amount      float64 `json:"amount"`
		Reason      string  `json:"reason"`
		ProcessedAt string  `json:"processed_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.BillingRefundResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	processedAt, _ := time.Parse(time.RFC3339, payload.ProcessedAt)
	return admindashboardports.BillingRefundResult{
		RefundID:    payload.RefundID,
		InvoiceID:   payload.InvoiceID,
		LineItemID:  payload.LineItemID,
		Amount:      payload.Amount,
		Reason:      payload.Reason,
		ProcessedAt: processedAt,
	}, nil
}

type controlPlaneRewardClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.RewardClient
}

func (c controlPlaneRewardClient) RecalculateReward(
	ctx context.Context,
	adminID string,
	userID string,
	submissionID string,
	campaignID string,
	lockedViews int64,
	ratePer1K float64,
	fraudScore float64,
	verificationCompletedAt time.Time,
	reason string,
	idempotencyKey string,
) (admindashboardports.RewardRecalculationResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.RewardRecalculationResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.RecalculateReward(ctx, adminID, userID, submissionID, campaignID, lockedViews, ratePer1K, fraudScore, verificationCompletedAt, reason, idempotencyKey)
	}

	body := map[string]interface{}{
		"user_id":       userID,
		"submission_id": submissionID,
		"campaign_id":   campaignID,
		"locked_views":  lockedViews,
		"rate_per_1k":   ratePer1K,
		"fraud_score":   fraudScore,
		"reason":        reason,
	}
	if !verificationCompletedAt.IsZero() {
		body["verification_completed_at"] = verificationCompletedAt.UTC().Format(time.RFC3339)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/rewards/recalculate",
		adminID,
		idempotencyKey,
		body,
	)
	if err != nil {
		return admindashboardports.RewardRecalculationResult{}, err
	}

	var payload struct {
		SubmissionID string  `json:"submission_id"`
		UserID       string  `json:"user_id"`
		CampaignID   string  `json:"campaign_id"`
		Status       string  `json:"status"`
		NetAmount    float64 `json:"net_amount"`
		Rollover     float64 `json:"rollover_total"`
		CalculatedAt string  `json:"calculated_at"`
		EligibleAt   *string `json:"eligible_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.RewardRecalculationResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	calculatedAt, _ := time.Parse(time.RFC3339, payload.CalculatedAt)
	var eligibleAt *time.Time
	if payload.EligibleAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *payload.EligibleAt); err == nil {
			eligibleAt = &parsed
		}
	}
	return admindashboardports.RewardRecalculationResult{
		SubmissionID:  payload.SubmissionID,
		UserID:        payload.UserID,
		CampaignID:    payload.CampaignID,
		Status:        payload.Status,
		NetAmount:     payload.NetAmount,
		RolloverTotal: payload.Rollover,
		CalculatedAt:  calculatedAt,
		EligibleAt:    eligibleAt,
	}, nil
}

type controlPlaneAffiliateClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.AffiliateClient
}

func (c controlPlaneAffiliateClient) SuspendAffiliate(
	ctx context.Context,
	adminID string,
	affiliateID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.AffiliateSuspensionResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.AffiliateSuspensionResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.SuspendAffiliate(ctx, adminID, affiliateID, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/admin/affiliates/"+url.PathEscape(affiliateID)+"/suspend",
		adminID,
		idempotencyKey,
		map[string]interface{}{"reason": reason},
	)
	if err != nil {
		return admindashboardports.AffiliateSuspensionResult{}, err
	}

	var payload struct {
		AffiliateID string `json:"affiliate_id"`
		Status      string `json:"status"`
		UpdatedAt   string `json:"updated_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.AffiliateSuspensionResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	updatedAt, _ := time.Parse(time.RFC3339, payload.UpdatedAt)
	return admindashboardports.AffiliateSuspensionResult{
		AffiliateID: payload.AffiliateID,
		Status:      payload.Status,
		UpdatedAt:   updatedAt,
	}, nil
}

func (c controlPlaneAffiliateClient) CreateAttribution(
	ctx context.Context,
	adminID string,
	affiliateID string,
	clickID string,
	orderID string,
	conversionID string,
	amount float64,
	currency string,
	reason string,
	idempotencyKey string,
) (admindashboardports.AffiliateAttributionResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.AffiliateAttributionResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateAttribution(ctx, adminID, affiliateID, clickID, orderID, conversionID, amount, currency, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/admin/affiliates/"+url.PathEscape(affiliateID)+"/attributions",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"click_id":      clickID,
			"order_id":      orderID,
			"conversion_id": conversionID,
			"amount":        amount,
			"currency":      currency,
			"reason":        reason,
		},
	)
	if err != nil {
		return admindashboardports.AffiliateAttributionResult{}, err
	}

	var payload struct {
		AttributionID string  `json:"attribution_id"`
		AffiliateID   string  `json:"affiliate_id"`
		OrderID       string  `json:"order_id"`
		Amount        float64 `json:"amount"`
		AttributedAt  string  `json:"attributed_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.AffiliateAttributionResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	attributedAt, _ := time.Parse(time.RFC3339, payload.AttributedAt)
	return admindashboardports.AffiliateAttributionResult{
		AttributionID: payload.AttributionID,
		AffiliateID:   payload.AffiliateID,
		OrderID:       payload.OrderID,
		Amount:        payload.Amount,
		AttributedAt:  attributedAt,
	}, nil
}

type controlPlanePayoutClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.PayoutClient
}

func (c controlPlanePayoutClient) RetryFailedPayout(
	ctx context.Context,
	adminID string,
	payoutID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.PayoutRetryResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.PayoutRetryResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.RetryFailedPayout(ctx, adminID, payoutID, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/payouts/"+url.PathEscape(payoutID)+"/retry",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"reason": reason,
		},
	)
	if err != nil {
		return admindashboardports.PayoutRetryResult{}, err
	}

	var payload struct {
		PayoutID      string `json:"payout_id"`
		UserID        string `json:"user_id"`
		Status        string `json:"status"`
		FailureReason string `json:"failure_reason"`
		ProcessedAt   string `json:"processed_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.PayoutRetryResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	processedAt, _ := time.Parse(time.RFC3339, payload.ProcessedAt)
	return admindashboardports.PayoutRetryResult{
		PayoutID:      payload.PayoutID,
		UserID:        payload.UserID,
		Status:        payload.Status,
		FailureReason: payload.FailureReason,
		ProcessedAt:   processedAt,
	}, nil
}

type controlPlaneResolutionClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.ResolutionClient
}

func (c controlPlaneResolutionClient) ResolveDispute(
	ctx context.Context,
	adminID string,
	disputeID string,
	action string,
	reason string,
	notes string,
	refundAmount float64,
	idempotencyKey string,
) (admindashboardports.DisputeResolutionResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.DisputeResolutionResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.ResolveDispute(ctx, adminID, disputeID, action, reason, notes, refundAmount, idempotencyKey)
	}

	path := "/api/v1/admin/disputes/" + url.PathEscape(disputeID) + "/resolve"
	body := map[string]interface{}{
		"reason":        reason,
		"notes":         notes,
		"refund_amount": refundAmount,
	}
	if action == "reopen" {
		path = "/api/v1/admin/disputes/" + url.PathEscape(disputeID) + "/reopen"
		body = map[string]interface{}{
			"reason": reason,
			"notes":  notes,
		}
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		path,
		adminID,
		idempotencyKey,
		body,
	)
	if err != nil {
		return admindashboardports.DisputeResolutionResult{}, err
	}

	var payload struct {
		DisputeID      string  `json:"dispute_id"`
		Status         string  `json:"status"`
		ResolutionType string  `json:"resolution_type"`
		RefundAmount   float64 `json:"refund_amount"`
		ProcessedAt    string  `json:"processed_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.DisputeResolutionResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	processedAt, _ := time.Parse(time.RFC3339, payload.ProcessedAt)
	return admindashboardports.DisputeResolutionResult{
		DisputeID:      payload.DisputeID,
		Status:         payload.Status,
		ResolutionType: payload.ResolutionType,
		RefundAmount:   payload.RefundAmount,
		ProcessedAt:    processedAt,
	}, nil
}

type controlPlaneConsentClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.ConsentClient
}

func (c controlPlaneConsentClient) GetConsent(
	ctx context.Context,
	adminID string,
	userID string,
) (admindashboardports.ConsentRecordResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.ConsentRecordResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.GetConsent(ctx, adminID, userID)
	}

	respBody, err := getMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/consent/"+url.PathEscape(userID),
		adminID,
	)
	if err != nil {
		return admindashboardports.ConsentRecordResult{}, err
	}

	var payload struct {
		UserID      string          `json:"user_id"`
		Status      string          `json:"status"`
		Preferences map[string]bool `json:"preferences"`
		UpdatedAt   string          `json:"updated_at"`
		LastUpdated string          `json:"last_updated"`
		UpdatedBy   string          `json:"updated_by"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.ConsentRecordResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	lastUpdated := parseRFC3339OrZero(payload.UpdatedAt)
	if lastUpdated.IsZero() {
		lastUpdated = parseRFC3339OrZero(payload.LastUpdated)
	}
	return admindashboardports.ConsentRecordResult{
		UserID:        payload.UserID,
		Status:        payload.Status,
		Preferences:   payload.Preferences,
		LastUpdated:   lastUpdated,
		LastUpdatedBy: payload.UpdatedBy,
	}, nil
}

func (c controlPlaneConsentClient) UpdateConsent(
	ctx context.Context,
	adminID string,
	userID string,
	preferences map[string]bool,
	reason string,
	idempotencyKey string,
) (admindashboardports.ConsentChangeResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.ConsentChangeResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.UpdateConsent(ctx, adminID, userID, preferences, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/consent/"+url.PathEscape(userID)+"/update",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"preferences": preferences,
			"reason":      reason,
		},
	)
	if err != nil {
		return admindashboardports.ConsentChangeResult{}, err
	}

	var payload struct {
		UserID    string `json:"user_id"`
		Status    string `json:"status"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.ConsentChangeResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.ConsentChangeResult{
		UserID:    payload.UserID,
		Status:    payload.Status,
		UpdatedAt: parseRFC3339OrZero(payload.UpdatedAt),
	}, nil
}

func (c controlPlaneConsentClient) WithdrawConsent(
	ctx context.Context,
	adminID string,
	userID string,
	category string,
	reason string,
	idempotencyKey string,
) (admindashboardports.ConsentChangeResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.ConsentChangeResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.WithdrawConsent(ctx, adminID, userID, category, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/consent/"+url.PathEscape(userID)+"/withdraw",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"category": category,
			"reason":   reason,
		},
	)
	if err != nil {
		return admindashboardports.ConsentChangeResult{}, err
	}

	var payload struct {
		UserID    string `json:"user_id"`
		Status    string `json:"status"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.ConsentChangeResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.ConsentChangeResult{
		UserID:    payload.UserID,
		Status:    payload.Status,
		UpdatedAt: parseRFC3339OrZero(payload.UpdatedAt),
	}, nil
}

type controlPlanePortabilityClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.PortabilityClient
}

func (c controlPlanePortabilityClient) CreateExport(
	ctx context.Context,
	adminID string,
	userID string,
	format string,
	reason string,
	idempotencyKey string,
) (admindashboardports.PortabilityRequestResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.PortabilityRequestResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateExport(ctx, adminID, userID, format, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/exports",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"user_id": userID,
			"format":  format,
			"reason":  reason,
		},
	)
	if err != nil {
		return admindashboardports.PortabilityRequestResult{}, err
	}
	return parsePortabilityResult(respBody)
}

func (c controlPlanePortabilityClient) GetExport(
	ctx context.Context,
	adminID string,
	requestID string,
) (admindashboardports.PortabilityRequestResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.PortabilityRequestResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.GetExport(ctx, adminID, requestID)
	}

	respBody, err := getMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/exports/"+url.PathEscape(requestID),
		adminID,
	)
	if err != nil {
		return admindashboardports.PortabilityRequestResult{}, err
	}
	return parsePortabilityResult(respBody)
}

func (c controlPlanePortabilityClient) CreateEraseRequest(
	ctx context.Context,
	adminID string,
	userID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.PortabilityRequestResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.PortabilityRequestResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateEraseRequest(ctx, adminID, userID, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/v1/admin/exports/erase",
		adminID,
		idempotencyKey,
		map[string]interface{}{
			"user_id": userID,
			"reason":  reason,
		},
	)
	if err != nil {
		return admindashboardports.PortabilityRequestResult{}, err
	}
	return parsePortabilityResult(respBody)
}

type controlPlaneRetentionClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.RetentionClient
}

func (c controlPlaneRetentionClient) CreateLegalHold(
	ctx context.Context,
	adminID string,
	entityID string,
	dataType string,
	reason string,
	expiresAt *time.Time,
	idempotencyKey string,
) (admindashboardports.RetentionHoldResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.RetentionHoldResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateLegalHold(ctx, adminID, entityID, dataType, reason, expiresAt, idempotencyKey)
	}

	body := map[string]interface{}{
		"entity_id": entityID,
		"data_type": dataType,
		"reason":    reason,
	}
	if expiresAt != nil && !expiresAt.IsZero() {
		body["expires_at"] = expiresAt.UTC().Format(time.RFC3339)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/admin/retention/legal-holds",
		adminID,
		idempotencyKey,
		body,
	)
	if err != nil {
		return admindashboardports.RetentionHoldResult{}, err
	}

	var payload struct {
		HoldID    string  `json:"hold_id"`
		EntityID  string  `json:"entity_id"`
		DataType  string  `json:"data_type"`
		Reason    string  `json:"reason"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.RetentionHoldResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	var parsedExpiresAt *time.Time
	if payload.ExpiresAt != nil {
		parsed := parseRFC3339OrZero(*payload.ExpiresAt)
		if !parsed.IsZero() {
			parsedExpiresAt = &parsed
		}
	}
	return admindashboardports.RetentionHoldResult{
		HoldID:    payload.HoldID,
		EntityID:  payload.EntityID,
		DataType:  payload.DataType,
		Reason:    payload.Reason,
		Status:    payload.Status,
		CreatedAt: parseRFC3339OrZero(payload.CreatedAt),
		ExpiresAt: parsedExpiresAt,
	}, nil
}

type controlPlaneLegalClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.LegalClient
}

func (c controlPlaneLegalClient) CheckHold(
	ctx context.Context,
	adminID string,
	entityType string,
	entityID string,
) (admindashboardports.LegalHoldCheckResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.LegalHoldCheckResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CheckHold(ctx, adminID, entityType, entityID)
	}

	path := "/api/v1/admin/legal/holds/check?entity_type=" + url.QueryEscape(entityType) + "&entity_id=" + url.QueryEscape(entityID)
	respBody, err := getMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		path,
		adminID,
	)
	if err != nil {
		return admindashboardports.LegalHoldCheckResult{}, err
	}

	var payload struct {
		EntityType string `json:"entity_type"`
		EntityID   string `json:"entity_id"`
		Held       bool   `json:"held"`
		HoldID     string `json:"hold_id"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.LegalHoldCheckResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.LegalHoldCheckResult{
		EntityType: payload.EntityType,
		EntityID:   payload.EntityID,
		Held:       payload.Held,
		HoldID:     payload.HoldID,
	}, nil
}

func (c controlPlaneLegalClient) ReleaseHold(
	ctx context.Context,
	adminID string,
	holdID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.LegalHoldResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.LegalHoldResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.ReleaseHold(ctx, adminID, holdID, reason, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/admin/legal/holds/"+url.PathEscape(holdID)+"/release",
		adminID,
		idempotencyKey,
		map[string]interface{}{"reason": reason},
	)
	if err != nil {
		return admindashboardports.LegalHoldResult{}, err
	}

	var payload struct {
		HoldID     string  `json:"hold_id"`
		EntityType string  `json:"entity_type"`
		EntityID   string  `json:"entity_id"`
		Reason     string  `json:"reason"`
		Status     string  `json:"status"`
		CreatedAt  string  `json:"created_at"`
		ReleasedAt *string `json:"released_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.LegalHoldResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	var releasedAt *time.Time
	if payload.ReleasedAt != nil {
		parsed := parseRFC3339OrZero(*payload.ReleasedAt)
		if !parsed.IsZero() {
			releasedAt = &parsed
		}
	}
	return admindashboardports.LegalHoldResult{
		HoldID:     payload.HoldID,
		EntityType: payload.EntityType,
		EntityID:   payload.EntityID,
		Reason:     payload.Reason,
		Status:     payload.Status,
		CreatedAt:  parseRFC3339OrZero(payload.CreatedAt),
		ReleasedAt: releasedAt,
	}, nil
}

func (c controlPlaneLegalClient) RunComplianceScan(
	ctx context.Context,
	adminID string,
	reportType string,
	idempotencyKey string,
) (admindashboardports.LegalComplianceReportResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.LegalComplianceReportResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.RunComplianceScan(ctx, adminID, reportType, idempotencyKey)
	}

	respBody, err := postMeshOwner(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/admin/legal/compliance/scan",
		adminID,
		idempotencyKey,
		map[string]interface{}{"report_type": reportType},
	)
	if err != nil {
		return admindashboardports.LegalComplianceReportResult{}, err
	}

	var payload struct {
		ReportID      string `json:"report_id"`
		ReportType    string `json:"report_type"`
		Status        string `json:"status"`
		FindingsCount int    `json:"findings_count"`
		DownloadURL   string `json:"download_url"`
		CreatedAt     string `json:"created_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.LegalComplianceReportResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.LegalComplianceReportResult{
		ReportID:      payload.ReportID,
		ReportType:    payload.ReportType,
		Status:        payload.Status,
		FindingsCount: payload.FindingsCount,
		DownloadURL:   payload.DownloadURL,
		CreatedAt:     parseRFC3339OrZero(payload.CreatedAt),
	}, nil
}

type controlPlaneSupportClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.SupportClient
}

func (c controlPlaneSupportClient) GetTicket(
	ctx context.Context,
	adminID string,
	ticketID string,
) (admindashboardports.SupportTicketResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.SupportTicketResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.GetTicket(ctx, adminID, ticketID)
	}
	respBody, err := getMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/support/admin/tickets/"+url.PathEscape(ticketID),
		adminID,
		"admin",
	)
	if err != nil {
		return admindashboardports.SupportTicketResult{}, err
	}
	return parseSupportTicketResult(respBody)
}

func (c controlPlaneSupportClient) SearchTickets(
	ctx context.Context,
	adminID string,
	filter admindashboardports.SupportTicketSearchFilter,
) ([]admindashboardports.SupportTicketResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.SearchTickets(ctx, adminID, filter)
	}

	query := url.Values{}
	if v := strings.TrimSpace(filter.Query); v != "" {
		query.Set("q", v)
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		query.Set("status", v)
	}
	if v := strings.TrimSpace(filter.Category); v != "" {
		query.Set("category", v)
	}
	if v := strings.TrimSpace(filter.AssignedTo); v != "" {
		query.Set("assigned_to", v)
	}
	if filter.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", filter.Limit))
	}
	path := "/api/v1/support/admin/tickets/search"
	if encoded := query.Encode(); encoded != "" {
		path = path + "?" + encoded
	}
	respBody, err := getMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		path,
		adminID,
		"admin",
	)
	if err != nil {
		return nil, err
	}
	var payload []json.RawMessage
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	results := make([]admindashboardports.SupportTicketResult, 0, len(payload))
	for _, item := range payload {
		row, err := parseSupportTicketResult(item)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, nil
}

func (c controlPlaneSupportClient) AssignTicket(
	ctx context.Context,
	adminID string,
	ticketID string,
	agentID string,
	reason string,
	idempotencyKey string,
) (admindashboardports.SupportTicketResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.SupportTicketResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.AssignTicket(ctx, adminID, ticketID, agentID, reason, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/support/admin/tickets/"+url.PathEscape(ticketID)+"/assign",
		adminID,
		"admin",
		idempotencyKey,
		map[string]interface{}{
			"agent_id": agentID,
			"reason":   reason,
		},
	)
	if err != nil {
		return admindashboardports.SupportTicketResult{}, err
	}
	return parseSupportTicketResult(respBody)
}

func (c controlPlaneSupportClient) UpdateTicket(
	ctx context.Context,
	adminID string,
	ticketID string,
	status string,
	subStatus string,
	priority string,
	reason string,
	idempotencyKey string,
) (admindashboardports.SupportTicketResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.SupportTicketResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.UpdateTicket(ctx, adminID, ticketID, status, subStatus, priority, reason, idempotencyKey)
	}
	body := map[string]interface{}{
		"reason": reason,
	}
	if strings.TrimSpace(status) != "" {
		body["status"] = status
	}
	if strings.TrimSpace(subStatus) != "" {
		body["sub_status"] = subStatus
	}
	if strings.TrimSpace(priority) != "" {
		body["priority"] = priority
	}
	respBody, err := patchMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/support/admin/tickets/"+url.PathEscape(ticketID),
		adminID,
		"admin",
		idempotencyKey,
		body,
	)
	if err != nil {
		return admindashboardports.SupportTicketResult{}, err
	}
	return parseSupportTicketResult(respBody)
}

type controlPlaneDeveloperPortalClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.DeveloperPortalClient
}

func (c controlPlaneDeveloperPortalClient) RotateAPIKey(
	ctx context.Context,
	adminID string,
	keyID string,
	idempotencyKey string,
) (admindashboardports.IntegrationKeyRotationResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.IntegrationKeyRotationResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.RotateAPIKey(ctx, adminID, keyID, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/developers/api-keys/"+url.PathEscape(keyID)+"/rotate",
		adminID,
		"admin",
		idempotencyKey,
		map[string]interface{}{},
	)
	if err != nil {
		return admindashboardports.IntegrationKeyRotationResult{}, err
	}
	var payload struct {
		RotationID  string `json:"rotation_id"`
		DeveloperID string `json:"developer_id"`
		OldKey      struct {
			KeyID string `json:"key_id"`
		} `json:"old_key"`
		NewKey struct {
			KeyID string `json:"key_id"`
		} `json:"new_key"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.IntegrationKeyRotationResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.IntegrationKeyRotationResult{
		RotationID:  payload.RotationID,
		DeveloperID: payload.DeveloperID,
		OldKeyID:    payload.OldKey.KeyID,
		NewKeyID:    payload.NewKey.KeyID,
		CreatedAt:   parseRFC3339OrZero(payload.CreatedAt),
	}, nil
}

type controlPlaneIntegrationHubClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.IntegrationHubClient
}

func (c controlPlaneIntegrationHubClient) TestWorkflow(
	ctx context.Context,
	adminID string,
	workflowID string,
	idempotencyKey string,
) (admindashboardports.IntegrationWorkflowTestResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.IntegrationWorkflowTestResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.TestWorkflow(ctx, adminID, workflowID, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/workflows/"+url.PathEscape(workflowID)+"/test",
		adminID,
		"admin",
		idempotencyKey,
		map[string]interface{}{},
	)
	if err != nil {
		return admindashboardports.IntegrationWorkflowTestResult{}, err
	}
	var payload struct {
		ExecutionID string `json:"execution_id"`
		WorkflowID  string `json:"workflow_id"`
		Status      string `json:"status"`
		TestRun     bool   `json:"test_run"`
		StartedAt   string `json:"started_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.IntegrationWorkflowTestResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.IntegrationWorkflowTestResult{
		ExecutionID: payload.ExecutionID,
		WorkflowID:  payload.WorkflowID,
		Status:      payload.Status,
		TestRun:     payload.TestRun,
		StartedAt:   parseRFC3339OrZero(payload.StartedAt),
	}, nil
}

type controlPlaneWebhookManagerClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.WebhookManagerClient
}

func (c controlPlaneWebhookManagerClient) ReplayWebhook(
	ctx context.Context,
	adminID string,
	webhookID string,
	idempotencyKey string,
) (admindashboardports.WebhookReplayResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.WebhookReplayResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.ReplayWebhook(ctx, adminID, webhookID, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/webhooks/"+url.PathEscape(webhookID)+"/test",
		adminID,
		"admin",
		idempotencyKey,
		map[string]interface{}{},
	)
	if err != nil {
		return admindashboardports.WebhookReplayResult{}, err
	}
	var payload struct {
		DeliveryID string `json:"delivery_id"`
		WebhookID  string `json:"webhook_id"`
		Status     string `json:"status"`
		HTTPStatus int    `json:"http_status"`
		LatencyMS  int64  `json:"latency_ms"`
		Timestamp  string `json:"timestamp"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.WebhookReplayResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.WebhookReplayResult{
		DeliveryID: payload.DeliveryID,
		WebhookID:  payload.WebhookID,
		Status:     payload.Status,
		HTTPStatus: payload.HTTPStatus,
		LatencyMS:  payload.LatencyMS,
		Timestamp:  parseRFC3339OrZero(payload.Timestamp),
	}, nil
}

func (c controlPlaneWebhookManagerClient) DisableWebhook(
	ctx context.Context,
	adminID string,
	webhookID string,
	idempotencyKey string,
) (admindashboardports.WebhookEndpointResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.WebhookEndpointResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.DisableWebhook(ctx, adminID, webhookID, idempotencyKey)
	}
	respBody, err := patchMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/webhooks/"+url.PathEscape(webhookID),
		adminID,
		"admin",
		idempotencyKey,
		map[string]interface{}{
			"status": "disabled",
		},
	)
	if err != nil {
		return admindashboardports.WebhookEndpointResult{}, err
	}
	var payload struct {
		WebhookID   string `json:"webhook_id"`
		Status      string `json:"status"`
		UpdatedAt   string `json:"updated_at"`
		EndpointURL string `json:"endpoint_url"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.WebhookEndpointResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.WebhookEndpointResult{
		WebhookID:   payload.WebhookID,
		Status:      payload.Status,
		UpdatedAt:   parseRFC3339OrZero(payload.UpdatedAt),
		EndpointURL: payload.EndpointURL,
	}, nil
}

func (c controlPlaneWebhookManagerClient) ListDeliveries(
	ctx context.Context,
	adminID string,
	webhookID string,
	limit int,
) ([]admindashboardports.WebhookDeliveryResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.ListDeliveries(ctx, adminID, webhookID, limit)
	}
	path := "/api/v1/webhooks/" + url.PathEscape(webhookID) + "/deliveries"
	if limit > 0 {
		path = path + "?limit=" + strconv.Itoa(limit)
	}
	respBody, err := getMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		path,
		adminID,
		"admin",
	)
	if err != nil {
		return nil, err
	}
	var payload []struct {
		DeliveryID      string `json:"delivery_id"`
		WebhookID       string `json:"webhook_id"`
		OriginalEventID string `json:"original_event_id"`
		OriginalType    string `json:"original_event_type"`
		HTTPStatus      int    `json:"http_status"`
		LatencyMS       int64  `json:"latency_ms"`
		RetryCount      int    `json:"retry_count"`
		DeliveredAt     string `json:"delivered_at"`
		IsTest          bool   `json:"is_test"`
		Success         bool   `json:"success"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	rows := make([]admindashboardports.WebhookDeliveryResult, 0, len(payload))
	for _, item := range payload {
		rows = append(rows, admindashboardports.WebhookDeliveryResult{
			DeliveryID:      item.DeliveryID,
			WebhookID:       item.WebhookID,
			OriginalEventID: item.OriginalEventID,
			OriginalType:    item.OriginalType,
			HTTPStatus:      item.HTTPStatus,
			LatencyMS:       item.LatencyMS,
			RetryCount:      item.RetryCount,
			DeliveredAt:     parseRFC3339OrZero(item.DeliveredAt),
			IsTest:          item.IsTest,
			Success:         item.Success,
		})
	}
	return rows, nil
}

func (c controlPlaneWebhookManagerClient) GetAnalytics(
	ctx context.Context,
	adminID string,
	webhookID string,
) (admindashboardports.WebhookAnalyticsResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.WebhookAnalyticsResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.GetAnalytics(ctx, adminID, webhookID)
	}
	respBody, err := getMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/api/v1/webhooks/"+url.PathEscape(webhookID)+"/analytics",
		adminID,
		"admin",
	)
	if err != nil {
		return admindashboardports.WebhookAnalyticsResult{}, err
	}
	var payload struct {
		TotalDeliveries      int64   `json:"total_deliveries"`
		SuccessfulDeliveries int64   `json:"successful_deliveries"`
		FailedDeliveries     int64   `json:"failed_deliveries"`
		SuccessRate          float64 `json:"success_rate"`
		AvgLatencyMS         float64 `json:"avg_latency_ms"`
		P95LatencyMS         float64 `json:"p95_latency_ms"`
		P99LatencyMS         float64 `json:"p99_latency_ms"`
		ByEventType          map[string]struct {
			Total      int64   `json:"total"`
			Success    int64   `json:"success"`
			Failed     int64   `json:"failed"`
			AvgLatency float64 `json:"avg_latency"`
		} `json:"by_event_type"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.WebhookAnalyticsResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	byType := make(map[string]admindashboardports.WebhookAnalyticsMetrics, len(payload.ByEventType))
	for key, value := range payload.ByEventType {
		byType[key] = admindashboardports.WebhookAnalyticsMetrics{
			Total:      value.Total,
			Success:    value.Success,
			Failed:     value.Failed,
			AvgLatency: value.AvgLatency,
		}
	}
	return admindashboardports.WebhookAnalyticsResult{
		TotalDeliveries:      payload.TotalDeliveries,
		SuccessfulDeliveries: payload.SuccessfulDeliveries,
		FailedDeliveries:     payload.FailedDeliveries,
		SuccessRate:          payload.SuccessRate,
		AvgLatencyMS:         payload.AvgLatencyMS,
		P95LatencyMS:         payload.P95LatencyMS,
		P99LatencyMS:         payload.P99LatencyMS,
		ByEventType:          byType,
	}, nil
}

type controlPlaneDataMigrationClient struct {
	baseURL  string
	client   *http.Client
	fallback admindashboardports.DataMigrationClient
}

func (c controlPlaneDataMigrationClient) CreatePlan(
	ctx context.Context,
	adminID string,
	serviceName string,
	environment string,
	version string,
	plan map[string]interface{},
	dryRun bool,
	riskLevel string,
	idempotencyKey string,
) (admindashboardports.MigrationPlanResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.MigrationPlanResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreatePlan(ctx, adminID, serviceName, environment, version, plan, dryRun, riskLevel, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/plans",
		adminID,
		"ops_admin",
		idempotencyKey,
		map[string]interface{}{
			"service_name": serviceName,
			"environment":  environment,
			"version":      version,
			"plan":         plan,
			"dry_run":      dryRun,
			"risk_level":   riskLevel,
		},
	)
	if err != nil {
		return admindashboardports.MigrationPlanResult{}, err
	}
	return parseMigrationPlanResult(respBody)
}

func (c controlPlaneDataMigrationClient) ListPlans(
	ctx context.Context,
	adminID string,
) ([]admindashboardports.MigrationPlanResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.ListPlans(ctx, adminID)
	}
	respBody, err := getMeshOwnerAsRole(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/plans",
		adminID,
		"ops_admin",
	)
	if err != nil {
		return nil, err
	}
	var payload []json.RawMessage
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	rows := make([]admindashboardports.MigrationPlanResult, 0, len(payload))
	for _, item := range payload {
		row, err := parseMigrationPlanResult(item)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (c controlPlaneDataMigrationClient) CreateRun(
	ctx context.Context,
	adminID string,
	planID string,
	idempotencyKey string,
) (admindashboardports.MigrationRunResult, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		if c.fallback == nil {
			return admindashboardports.MigrationRunResult{}, admindashboarderrors.ErrDependencyUnavailable
		}
		return c.fallback.CreateRun(ctx, adminID, planID, idempotencyKey)
	}
	respBody, err := postMeshOwnerAsRoleWithHeaders(
		ctx,
		httpClientOrDefault(c.client),
		c.baseURL,
		"/runs",
		adminID,
		"ops_admin",
		idempotencyKey,
		map[string]interface{}{
			"plan_id": planID,
		},
		map[string]string{
			"X-MFA-Verified": "true",
		},
	)
	if err != nil {
		return admindashboardports.MigrationRunResult{}, err
	}
	var payload struct {
		RunID             string `json:"run_id"`
		PlanID            string `json:"plan_id"`
		Status            string `json:"status"`
		OperatorID        string `json:"operator_id"`
		SnapshotCreated   bool   `json:"snapshot_created"`
		RollbackAvailable bool   `json:"rollback_available"`
		ValidationStatus  string `json:"validation_status"`
		BackfillJobID     string `json:"backfill_job_id"`
		StartedAt         string `json:"started_at"`
		CompletedAt       string `json:"completed_at"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return admindashboardports.MigrationRunResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.MigrationRunResult{
		RunID:             payload.RunID,
		PlanID:            payload.PlanID,
		Status:            payload.Status,
		OperatorID:        payload.OperatorID,
		SnapshotCreated:   payload.SnapshotCreated,
		RollbackAvailable: payload.RollbackAvailable,
		ValidationStatus:  payload.ValidationStatus,
		BackfillJobID:     payload.BackfillJobID,
		StartedAt:         parseRFC3339OrZero(payload.StartedAt),
		CompletedAt:       parseRFC3339OrZero(payload.CompletedAt),
	}, nil
}

func newAdminDashboardModule(
	authorizationModule authorization.Module,
	moderationModule moderationservice.Module,
	abuseModule abusepreventionservice.Module,
	editorModule editordashboardservice.Module,
	clippingModule clippingtoolservice.Module,
) (admindashboardservice.Module, error) {
	cfg, err := loadAdminOwnerClientConfigFromEnv()
	if err != nil {
		return admindashboardservice.Module{}, err
	}

	store := admindashboardmemory.NewStore()
	httpClient := &http.Client{Timeout: ownerRequestTimeout}

	var financeFallback admindashboardports.FinanceClient
	var billingFallback admindashboardports.BillingClient
	var rewardFallback admindashboardports.RewardClient
	var affiliateFallback admindashboardports.AffiliateClient
	var payoutFallback admindashboardports.PayoutClient
	var resolutionFallback admindashboardports.ResolutionClient
	var consentFallback admindashboardports.ConsentClient
	var portabilityFallback admindashboardports.PortabilityClient
	var retentionFallback admindashboardports.RetentionClient
	var legalFallback admindashboardports.LegalClient
	var supportFallback admindashboardports.SupportClient
	var autoClippingFallback admindashboardports.AutoClippingClient
	var developerPortalFallback admindashboardports.DeveloperPortalClient
	var integrationHubFallback admindashboardports.IntegrationHubClient
	var webhookManagerFallback admindashboardports.WebhookManagerClient
	var dataMigrationFallback admindashboardports.DataMigrationClient
	if cfg.runtime.allowFallback {
		financeFallback = store
		billingFallback = store
		rewardFallback = store
		affiliateFallback = store
		payoutFallback = store
		resolutionFallback = store
		consentFallback = store
		portabilityFallback = store
		retentionFallback = store
		legalFallback = store
		supportFallback = store
		autoClippingFallback = store
		developerPortalFallback = store
		integrationHubFallback = store
		webhookManagerFallback = store
		dataMigrationFallback = store
	}

	module := admindashboardservice.NewModule(admindashboardservice.Dependencies{
		Repository:            store,
		Idempotency:           store,
		AuthorizationClient:   controlPlaneAuthorizationClient{module: authorizationModule},
		ModerationClient:      controlPlaneModerationClient{module: moderationModule},
		AbusePreventionClient: controlPlaneAbusePreventionClient{module: abuseModule},
		FinanceClient: controlPlaneFinanceClient{
			baseURL:  cfg.m39BaseURL,
			client:   httpClient,
			fallback: financeFallback,
		},
		BillingClient: controlPlaneBillingClient{
			baseURL:  cfg.m05BaseURL,
			client:   httpClient,
			fallback: billingFallback,
		},
		RewardClient: controlPlaneRewardClient{
			baseURL:  cfg.m41BaseURL,
			client:   httpClient,
			fallback: rewardFallback,
		},
		AffiliateClient: controlPlaneAffiliateClient{
			baseURL:  cfg.m89BaseURL,
			client:   httpClient,
			fallback: affiliateFallback,
		},
		PayoutClient: controlPlanePayoutClient{
			baseURL:  cfg.m14BaseURL,
			client:   httpClient,
			fallback: payoutFallback,
		},
		ResolutionClient: controlPlaneResolutionClient{
			baseURL:  cfg.m44BaseURL,
			client:   httpClient,
			fallback: resolutionFallback,
		},
		ConsentClient: controlPlaneConsentClient{
			baseURL:  cfg.m50BaseURL,
			client:   httpClient,
			fallback: consentFallback,
		},
		PortabilityClient: controlPlanePortabilityClient{
			baseURL:  cfg.m51BaseURL,
			client:   httpClient,
			fallback: portabilityFallback,
		},
		RetentionClient: controlPlaneRetentionClient{
			baseURL:  cfg.m68BaseURL,
			client:   httpClient,
			fallback: retentionFallback,
		},
		LegalClient: controlPlaneLegalClient{
			baseURL:  cfg.m69BaseURL,
			client:   httpClient,
			fallback: legalFallback,
		},
		SupportClient: controlPlaneSupportClient{
			baseURL:  cfg.m73BaseURL,
			client:   httpClient,
			fallback: supportFallback,
		},
		EditorWorkflowClient: controlPlaneEditorWorkflowClient{
			module: editorModule,
		},
		ClippingWorkflowClient: controlPlaneClippingWorkflowClient{
			module: clippingModule,
		},
		AutoClippingClient: controlPlaneAutoClippingClient{
			baseURL:  cfg.m25BaseURL,
			client:   httpClient,
			fallback: autoClippingFallback,
		},
		DeveloperPortalClient: controlPlaneDeveloperPortalClient{
			baseURL:  cfg.m70BaseURL,
			client:   httpClient,
			fallback: developerPortalFallback,
		},
		IntegrationHubClient: controlPlaneIntegrationHubClient{
			baseURL:  cfg.m71BaseURL,
			client:   httpClient,
			fallback: integrationHubFallback,
		},
		WebhookManagerClient: controlPlaneWebhookManagerClient{
			baseURL:  cfg.m72BaseURL,
			client:   httpClient,
			fallback: webhookManagerFallback,
		},
		DataMigrationClient: controlPlaneDataMigrationClient{
			baseURL:  cfg.m84BaseURL,
			client:   httpClient,
			fallback: dataMigrationFallback,
		},
		Clock:          store,
		IdempotencyTTL: 7 * 24 * time.Hour,
	})
	module.Store = store
	return module, nil
}

func loadAdminOwnerClientConfigFromEnv() (adminOwnerClientConfig, error) {
	runtime := adminRuntimeFromEnv()
	cfg := adminOwnerClientConfig{
		runtime:    runtime,
		m39BaseURL: strings.TrimSpace(os.Getenv(adminM39BaseURLEnv)),
		m14BaseURL: strings.TrimSpace(os.Getenv(adminM14BaseURLEnv)),
		m44BaseURL: strings.TrimSpace(os.Getenv(adminM44BaseURLEnv)),
		m05BaseURL: strings.TrimSpace(os.Getenv(adminM05BaseURLEnv)),
		m41BaseURL: strings.TrimSpace(os.Getenv(adminM41BaseURLEnv)),
		m89BaseURL: strings.TrimSpace(os.Getenv(adminM89BaseURLEnv)),
		m50BaseURL: strings.TrimSpace(os.Getenv(adminM50BaseURLEnv)),
		m51BaseURL: strings.TrimSpace(os.Getenv(adminM51BaseURLEnv)),
		m68BaseURL: strings.TrimSpace(os.Getenv(adminM68BaseURLEnv)),
		m69BaseURL: strings.TrimSpace(os.Getenv(adminM69BaseURLEnv)),
		m73BaseURL: strings.TrimSpace(os.Getenv(adminM73BaseURLEnv)),
		m25BaseURL: strings.TrimSpace(os.Getenv(adminM25BaseURLEnv)),
		m70BaseURL: strings.TrimSpace(os.Getenv(adminM70BaseURLEnv)),
		m71BaseURL: strings.TrimSpace(os.Getenv(adminM71BaseURLEnv)),
		m72BaseURL: strings.TrimSpace(os.Getenv(adminM72BaseURLEnv)),
		m84BaseURL: strings.TrimSpace(os.Getenv(adminM84BaseURLEnv)),
	}
	if runtime.allowFallback {
		return cfg, nil
	}

	var invalid []string
	for _, item := range []struct {
		envName string
		value   string
	}{
		{envName: adminM39BaseURLEnv, value: cfg.m39BaseURL},
		{envName: adminM14BaseURLEnv, value: cfg.m14BaseURL},
		{envName: adminM44BaseURLEnv, value: cfg.m44BaseURL},
		{envName: adminM05BaseURLEnv, value: cfg.m05BaseURL},
		{envName: adminM41BaseURLEnv, value: cfg.m41BaseURL},
		{envName: adminM89BaseURLEnv, value: cfg.m89BaseURL},
		{envName: adminM50BaseURLEnv, value: cfg.m50BaseURL},
		{envName: adminM51BaseURLEnv, value: cfg.m51BaseURL},
		{envName: adminM68BaseURLEnv, value: cfg.m68BaseURL},
		{envName: adminM69BaseURLEnv, value: cfg.m69BaseURL},
		{envName: adminM73BaseURLEnv, value: cfg.m73BaseURL},
		{envName: adminM25BaseURLEnv, value: cfg.m25BaseURL},
		{envName: adminM70BaseURLEnv, value: cfg.m70BaseURL},
		{envName: adminM71BaseURLEnv, value: cfg.m71BaseURL},
		{envName: adminM72BaseURLEnv, value: cfg.m72BaseURL},
		{envName: adminM84BaseURLEnv, value: cfg.m84BaseURL},
	} {
		if err := validateOwnerBaseURL(item.envName, item.value); err != nil {
			invalid = append(invalid, err.Error())
		}
	}
	if len(invalid) > 0 {
		return adminOwnerClientConfig{}, fmt.Errorf(
			"admin control-plane owner client config invalid for production runtime: %s (set %s to dev/local/test to allow local fallbacks)",
			strings.Join(invalid, "; "),
			adminRuntimeModeEnv,
		)
	}
	return cfg, nil
}

func adminRuntimeFromEnv() adminControlPlaneRuntime {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(adminRuntimeModeEnv)))
	switch mode {
	case "dev", "development", "local", "test", "testing":
		return adminControlPlaneRuntime{mode: mode, allowFallback: true}
	default:
		return adminControlPlaneRuntime{mode: mode, allowFallback: false}
	}
}

func validateOwnerBaseURL(envName, rawValue string) error {
	value := strings.TrimSpace(rawValue)
	if value == "" {
		return fmt.Errorf("%s is required", envName)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", envName)
	}
	return nil
}

func postMeshOwner(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
	idempotencyKey string,
	body interface{},
) ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, admindashboarderrors.ErrInvalidInput
	}
	httpClient := httpClientOrDefault(client)
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	idem := strings.TrimSpace(idempotencyKey)
	requestID := fmt.Sprintf("m86-%s", idem)
	lastErr := admindashboarderrors.ErrDependencyUnavailable

	for attempt := 1; attempt <= ownerRetryMaxAttempt; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			cancel()
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer admin:"+strings.TrimSpace(adminID))
		req.Header.Set("X-Actor-Role", "admin")
		req.Header.Set("X-Request-Id", requestID)
		req.Header.Set("Idempotency-Key", idem)

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && shouldRetryTransportError(err) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			lastErr = mapMeshError(resp.StatusCode, raw)
			if attempt < ownerRetryMaxAttempt && shouldRetryStatusCode(resp.StatusCode) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		var envelope meshSuccessEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return envelope.Data, nil
	}

	return nil, lastErr
}

func getMeshOwner(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
) ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	httpClient := httpClientOrDefault(client)
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	requestID := fmt.Sprintf("m86-read-%d", time.Now().UTC().UnixNano())
	lastErr := admindashboarderrors.ErrDependencyUnavailable

	for attempt := 1; attempt <= ownerRetryMaxAttempt; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			cancel()
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		req.Header.Set("Authorization", "Bearer admin:"+strings.TrimSpace(adminID))
		req.Header.Set("X-Actor-Role", "admin")
		req.Header.Set("X-Request-Id", requestID)

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && shouldRetryTransportError(err) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			lastErr = mapMeshError(resp.StatusCode, raw)
			if attempt < ownerRetryMaxAttempt && shouldRetryStatusCode(resp.StatusCode) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		var envelope meshSuccessEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return envelope.Data, nil
	}

	return nil, lastErr
}

func getMeshOwnerAsRole(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
	actorRole string,
) ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	httpClient := httpClientOrDefault(client)
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	requestID := fmt.Sprintf("m86-read-%d", time.Now().UTC().UnixNano())
	lastErr := admindashboarderrors.ErrDependencyUnavailable
	role := strings.TrimSpace(actorRole)
	if role == "" {
		role = "admin"
	}

	for attempt := 1; attempt <= ownerRetryMaxAttempt; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			cancel()
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		req.Header.Set("Authorization", "Bearer admin:"+strings.TrimSpace(adminID))
		req.Header.Set("X-Actor-Role", role)
		req.Header.Set("X-Request-Id", requestID)

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && shouldRetryTransportError(err) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			lastErr = mapMeshError(resp.StatusCode, raw)
			if attempt < ownerRetryMaxAttempt && shouldRetryStatusCode(resp.StatusCode) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		var envelope meshSuccessEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return envelope.Data, nil
	}

	return nil, lastErr
}

func postMeshOwnerAsRole(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
	actorRole string,
	idempotencyKey string,
	body interface{},
) ([]byte, error) {
	return postMeshOwnerAsRoleWithHeaders(
		ctx,
		client,
		baseURL,
		path,
		adminID,
		actorRole,
		idempotencyKey,
		body,
		nil,
	)
}

func postMeshOwnerAsRoleWithHeaders(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
	actorRole string,
	idempotencyKey string,
	body interface{},
	extraHeaders map[string]string,
) ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, admindashboarderrors.ErrInvalidInput
	}
	httpClient := httpClientOrDefault(client)
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	idem := strings.TrimSpace(idempotencyKey)
	requestID := fmt.Sprintf("m86-%s", idem)
	lastErr := admindashboarderrors.ErrDependencyUnavailable
	role := strings.TrimSpace(actorRole)
	if role == "" {
		role = "admin"
	}

	for attempt := 1; attempt <= ownerRetryMaxAttempt; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			cancel()
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer admin:"+strings.TrimSpace(adminID))
		req.Header.Set("X-Actor-Role", role)
		req.Header.Set("X-Request-Id", requestID)
		req.Header.Set("Idempotency-Key", idem)
		for key, value := range extraHeaders {
			if strings.TrimSpace(key) == "" {
				continue
			}
			req.Header.Set(key, value)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && shouldRetryTransportError(err) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			lastErr = mapMeshError(resp.StatusCode, raw)
			if attempt < ownerRetryMaxAttempt && shouldRetryStatusCode(resp.StatusCode) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		var envelope meshSuccessEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return envelope.Data, nil
	}

	return nil, lastErr
}

func patchMeshOwnerAsRole(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	path string,
	adminID string,
	actorRole string,
	idempotencyKey string,
	body interface{},
) ([]byte, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, admindashboarderrors.ErrDependencyUnavailable
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, admindashboarderrors.ErrInvalidInput
	}
	httpClient := httpClientOrDefault(client)
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	idem := strings.TrimSpace(idempotencyKey)
	requestID := fmt.Sprintf("m86-%s", idem)
	lastErr := admindashboarderrors.ErrDependencyUnavailable
	role := strings.TrimSpace(actorRole)
	if role == "" {
		role = "admin"
	}

	for attempt := 1; attempt <= ownerRetryMaxAttempt; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, ownerRequestTimeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPatch, endpoint, bytes.NewReader(payload))
		if err != nil {
			cancel()
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer admin:"+strings.TrimSpace(adminID))
		req.Header.Set("X-Actor-Role", role)
		req.Header.Set("X-Request-Id", requestID)
		req.Header.Set("Idempotency-Key", idem)

		resp, err := httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && shouldRetryTransportError(err) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		raw, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			lastErr = admindashboarderrors.ErrDependencyUnavailable
			if attempt < ownerRetryMaxAttempt && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			lastErr = mapMeshError(resp.StatusCode, raw)
			if attempt < ownerRetryMaxAttempt && shouldRetryStatusCode(resp.StatusCode) && sleepBeforeRetry(ctx, attempt) {
				continue
			}
			return nil, lastErr
		}

		var envelope meshSuccessEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, admindashboarderrors.ErrDependencyUnavailable
		}
		return envelope.Data, nil
	}

	return nil, lastErr
}

func parsePortabilityResult(raw []byte) (admindashboardports.PortabilityRequestResult, error) {
	var payload struct {
		RequestID   string  `json:"request_id"`
		UserID      string  `json:"user_id"`
		RequestType string  `json:"request_type"`
		Format      string  `json:"format"`
		Status      string  `json:"status"`
		Reason      string  `json:"reason"`
		RequestedAt string  `json:"requested_at"`
		CompletedAt *string `json:"completed_at"`
		DownloadURL string  `json:"download_url"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return admindashboardports.PortabilityRequestResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	var completedAt *time.Time
	if payload.CompletedAt != nil {
		parsed := parseRFC3339OrZero(*payload.CompletedAt)
		if !parsed.IsZero() {
			completedAt = &parsed
		}
	}
	return admindashboardports.PortabilityRequestResult{
		RequestID:   payload.RequestID,
		UserID:      payload.UserID,
		RequestType: payload.RequestType,
		Format:      payload.Format,
		Status:      payload.Status,
		Reason:      payload.Reason,
		RequestedAt: parseRFC3339OrZero(payload.RequestedAt),
		CompletedAt: completedAt,
		DownloadURL: payload.DownloadURL,
	}, nil
}

func parseSupportTicketResult(raw []byte) (admindashboardports.SupportTicketResult, error) {
	var payload struct {
		TicketID         string `json:"ticket_id"`
		UserID           string `json:"user_id"`
		Subject          string `json:"subject"`
		Description      string `json:"description"`
		Category         string `json:"category"`
		Priority         string `json:"priority"`
		Status           string `json:"status"`
		SubStatus        string `json:"sub_status"`
		AssignedAgentID  string `json:"assigned_agent_id"`
		SLAResponseDueAt string `json:"sla_response_due_at"`
		LastActivityAt   string `json:"last_activity_at"`
		UpdatedAt        string `json:"updated_at"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return admindashboardports.SupportTicketResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.SupportTicketResult{
		TicketID:         payload.TicketID,
		UserID:           payload.UserID,
		Subject:          payload.Subject,
		Description:      payload.Description,
		Category:         payload.Category,
		Priority:         payload.Priority,
		Status:           payload.Status,
		SubStatus:        payload.SubStatus,
		AssignedAgentID:  payload.AssignedAgentID,
		SLAResponseDueAt: parseRFC3339OrZero(payload.SLAResponseDueAt),
		LastActivityAt:   parseRFC3339OrZero(payload.LastActivityAt),
		UpdatedAt:        parseRFC3339OrZero(payload.UpdatedAt),
	}, nil
}

func parseMigrationPlanResult(raw []byte) (admindashboardports.MigrationPlanResult, error) {
	var payload struct {
		PlanID           string                 `json:"plan_id"`
		ServiceName      string                 `json:"service_name"`
		Environment      string                 `json:"environment"`
		Version          string                 `json:"version"`
		Plan             map[string]interface{} `json:"plan"`
		Status           string                 `json:"status"`
		DryRun           bool                   `json:"dry_run"`
		RiskLevel        string                 `json:"risk_level"`
		StagingValidated bool                   `json:"staging_validated"`
		BackupRequired   bool                   `json:"backup_required"`
		CreatedBy        string                 `json:"created_by"`
		CreatedAt        string                 `json:"created_at"`
		UpdatedAt        string                 `json:"updated_at"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return admindashboardports.MigrationPlanResult{}, admindashboarderrors.ErrDependencyUnavailable
	}
	return admindashboardports.MigrationPlanResult{
		PlanID:           payload.PlanID,
		ServiceName:      payload.ServiceName,
		Environment:      payload.Environment,
		Version:          payload.Version,
		Plan:             payload.Plan,
		Status:           payload.Status,
		DryRun:           payload.DryRun,
		RiskLevel:        payload.RiskLevel,
		StagingValidated: payload.StagingValidated,
		BackupRequired:   payload.BackupRequired,
		CreatedBy:        payload.CreatedBy,
		CreatedAt:        parseRFC3339OrZero(payload.CreatedAt),
		UpdatedAt:        parseRFC3339OrZero(payload.UpdatedAt),
	}, nil
}

func mapMeshError(statusCode int, raw []byte) error {
	if mapped, ok := mapMeshErrorCode(raw); ok {
		return mapped
	}

	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return admindashboarderrors.ErrInvalidInput
	case http.StatusUnauthorized, http.StatusForbidden:
		return admindashboarderrors.ErrUnauthorized
	case http.StatusNotFound:
		return admindashboarderrors.ErrNotFound
	case http.StatusConflict:
		return admindashboarderrors.ErrConflict
	case http.StatusLocked:
		return admindashboarderrors.ErrConflict
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return admindashboarderrors.ErrDependencyUnavailable
	default:
		return admindashboarderrors.ErrDependencyUnavailable
	}
}

func mapMeshErrorCode(raw []byte) (error, bool) {
	code := ""
	var payload meshErrorEnvelope
	if err := json.Unmarshal(raw, &payload); err == nil {
		code = strings.TrimSpace(strings.ToLower(payload.Error.Code))
	}
	if code == "" {
		var topLevel meshTopLevelErrorEnvelope
		if err := json.Unmarshal(raw, &topLevel); err == nil {
			code = strings.TrimSpace(strings.ToLower(topLevel.Code))
		}
	}
	if code == "" {
		return nil, false
	}
	switch code {
	case "not_found":
		return admindashboarderrors.ErrNotFound, true
	case "conflict", "idempotency_conflict", "webhook_already_processed", "legal_hold_active":
		return admindashboarderrors.ErrConflict, true
	case "forbidden", "unauthorized":
		return admindashboarderrors.ErrUnauthorized, true
	case "invalid_input", "invalid_request", "invalid_json", "missing_request_id", "method_not_allowed":
		return admindashboarderrors.ErrInvalidInput, true
	case "idempotency_key_required", "missing_idempotency_key":
		return admindashboarderrors.ErrIdempotencyRequired, true
	case "service_unavailable", "dependency_unavailable", "upstream_timeout":
		return admindashboarderrors.ErrDependencyUnavailable, true
	default:
		return nil, false
	}
}

func httpClientOrDefault(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: ownerRequestTimeout}
}

func shouldRetryStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func shouldRetryTransportError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}
	return true
}

func sleepBeforeRetry(ctx context.Context, attempt int) bool {
	delay := ownerRetryBackoff * time.Duration(attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func parseRFC3339OrZero(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func writeAdminDashboardDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, admindashboarderrors.ErrInvalidInput):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, admindashboarderrors.ErrUnauthorized):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, admindashboarderrors.ErrNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, admindashboarderrors.ErrConflict):
		writeSuperAdminError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, admindashboarderrors.ErrIdempotencyRequired):
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, admindashboarderrors.ErrIdempotencyConflict):
		writeSuperAdminError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, admindashboarderrors.ErrDependencyUnavailable):
		writeSuperAdminError(w, http.StatusServiceUnavailable, "dependency_unavailable", err.Error())
	case errors.Is(err, admindashboarderrors.ErrUnsupportedAction):
		writeSuperAdminError(w, http.StatusBadRequest, "unsupported_action", err.Error())
	case errors.Is(err, authzerrors.ErrInvalidPermission):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, authzerrors.ErrInvalidUserID):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, authzerrors.ErrInvalidRoleID):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, authzerrors.ErrInvalidAdminID):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, authzerrors.ErrRoleNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, authzerrors.ErrUserNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, authzerrors.ErrRoleAlreadyAssigned):
		writeSuperAdminError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, authzerrors.ErrRoleNotAssigned):
		writeSuperAdminError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, authzerrors.ErrForbidden):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, authzerrors.ErrIdempotencyKeyRequired):
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, authzerrors.ErrIdempotencyConflict):
		writeSuperAdminError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, moderationerrors.ErrInvalidRequest):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, moderationerrors.ErrNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, moderationerrors.ErrForbidden):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, moderationerrors.ErrDependencyUnavailable):
		writeSuperAdminError(w, http.StatusServiceUnavailable, "dependency_unavailable", err.Error())
	case errors.Is(err, moderationerrors.ErrIdempotencyKeyRequired):
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, moderationerrors.ErrIdempotencyConflict):
		writeSuperAdminError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, abuseerrors.ErrInvalidRequest):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, abuseerrors.ErrUnauthorized):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, abuseerrors.ErrForbidden):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, abuseerrors.ErrThreatNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, abuseerrors.ErrIdempotencyKeyRequired):
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, abuseerrors.ErrIdempotencyConflict):
		writeSuperAdminError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	default:
		writeSuperAdminError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (s *Server) requireAdminControlPlanePermission(
	w http.ResponseWriter,
	r *http.Request,
	adminID string,
	permission string,
) bool {
	decision, err := s.authorization.Handler.CheckPermissionHandler(
		r.Context(),
		adminID,
		authzhttp.CheckPermissionRequest{
			Permission: permission,
		},
	)
	if err != nil {
		writeSuperAdminError(w, http.StatusServiceUnavailable, "dependency_unavailable", "authorization scope check failed")
		return false
	}
	if !decision.Allowed {
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", "insufficient admin permission scope")
		return false
	}
	return true
}

func (s *Server) handleAdminRecordAction(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req admindashboardhttp.RecordAdminActionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RecordAdminActionHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminIdentityGrantRole(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req admindashboardhttp.GrantIdentityRoleRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.GrantIdentityRoleHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminModerationDecision(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req admindashboardhttp.ModerateSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.ModerateSubmissionHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminReleaseAbuseLockout(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.ReleaseAbuseLockoutRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.UserID = strings.TrimSpace(r.PathValue("user_id"))
	if strings.TrimSpace(req.UserID) == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.ReleaseAbuseLockoutHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreateFinanceRefund(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.CreateFinanceRefundRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.CreateFinanceRefundHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreateBillingRefund(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.CreateBillingRefundRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.InvoiceID = strings.TrimSpace(r.PathValue("invoice_id"))
	if req.InvoiceID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "invoice_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.CreateBillingRefundHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminRecalculateReward(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RecalculateRewardRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RecalculateRewardHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminSuspendAffiliate(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.SuspendAffiliateRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.AffiliateID = strings.TrimSpace(r.PathValue("affiliate_id"))
	if req.AffiliateID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "affiliate_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.SuspendAffiliateHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreateAffiliateAttribution(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.CreateAffiliateAttributionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.AffiliateID = strings.TrimSpace(r.PathValue("affiliate_id"))
	if req.AffiliateID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "affiliate_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.CreateAffiliateAttributionHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminRetryFailedPayout(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RetryFailedPayoutRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.PayoutID = strings.TrimSpace(r.PathValue("payout_id"))
	if strings.TrimSpace(req.PayoutID) == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "payout_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RetryFailedPayoutHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminResolveDispute(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.ResolveDisputeRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.DisputeID = strings.TrimSpace(r.PathValue("dispute_id"))
	if strings.TrimSpace(req.DisputeID) == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "dispute_id is required")
		return
	}
	if strings.TrimSpace(req.Action) == "" {
		req.Action = "resolve"
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.ResolveDisputeHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminGetConsent(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	req := admindashboardhttp.GetConsentRequest{
		UserID: strings.TrimSpace(r.PathValue("user_id")),
	}
	if req.UserID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}
	resp, err := s.adminDashboard.Handler.GetConsentHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminUpdateConsent(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.UpdateConsentRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.UserID = strings.TrimSpace(r.PathValue("user_id"))
	if req.UserID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.UpdateConsentHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminWithdrawConsent(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.WithdrawConsentRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.UserID = strings.TrimSpace(r.PathValue("user_id"))
	if req.UserID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.WithdrawConsentHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminStartDataExport(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.StartDataExportRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.StartDataExportHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminGetDataExport(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	requestID := strings.TrimSpace(r.PathValue("request_id"))
	if requestID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "request_id is required")
		return
	}
	resp, err := s.adminDashboard.Handler.GetDataExportHandler(
		r.Context(),
		adminID,
		requestID,
		resolveClientIP(r),
		getRequestID(r),
	)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminRequestDeletion(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RequestDeletionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RequestDeletionHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreateRetentionLegalHold(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.CreateRetentionLegalHoldRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.CreateRetentionLegalHoldHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCheckLegalHold(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	req := admindashboardhttp.CheckLegalHoldRequest{
		EntityType:    strings.TrimSpace(r.URL.Query().Get("entity_type")),
		EntityID:      strings.TrimSpace(r.URL.Query().Get("entity_id")),
		SourceIP:      resolveClientIP(r),
		CorrelationID: getRequestID(r),
	}
	resp, err := s.adminDashboard.Handler.CheckLegalHoldHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminReleaseLegalHold(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.ReleaseLegalHoldRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.HoldID = strings.TrimSpace(r.PathValue("hold_id"))
	if req.HoldID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "hold_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.ReleaseLegalHoldHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminRunComplianceScan(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RunComplianceScanRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RunComplianceScanHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminGetSupportTicket(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	req := admindashboardhttp.GetSupportTicketRequest{
		TicketID: strings.TrimSpace(r.PathValue("ticket_id")),
	}
	if req.TicketID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "ticket_id is required")
		return
	}
	resp, err := s.adminDashboard.Handler.GetSupportTicketHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminSearchSupportTickets(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	req := admindashboardhttp.SearchSupportTicketsRequest{
		Query:      query,
		Status:     strings.TrimSpace(r.URL.Query().Get("status")),
		Category:   strings.TrimSpace(r.URL.Query().Get("category")),
		AssignedTo: strings.TrimSpace(r.URL.Query().Get("assigned_to")),
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "limit must be a valid integer")
			return
		}
		req.Limit = parsed
	}
	resp, err := s.adminDashboard.Handler.SearchSupportTicketsHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAssignSupportTicket(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.AssignSupportTicketRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.TicketID = strings.TrimSpace(r.PathValue("ticket_id"))
	if req.TicketID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "ticket_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.AssignSupportTicketHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminUpdateSupportTicket(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.UpdateSupportTicketRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.TicketID = strings.TrimSpace(r.PathValue("ticket_id"))
	if req.TicketID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "ticket_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.UpdateSupportTicketHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreatorWorkflowEditorSaveCampaign(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.SaveEditorCampaignRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.CampaignID = strings.TrimSpace(r.PathValue("campaign_id"))
	if req.CampaignID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "campaign_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.SaveEditorCampaignHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreatorWorkflowClippingRequestExport(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RequestClippingExportRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.ProjectID = strings.TrimSpace(r.PathValue("project_id"))
	if req.ProjectID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "project_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RequestClippingExportHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreatorWorkflowAutoClippingDeployModel(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.DeployAutoClippingModelRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.DeployAutoClippingModelHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminRotateIntegrationKey(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.RotateIntegrationKeyRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.RotateIntegrationKeyHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminTestIntegrationWorkflow(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.TestIntegrationWorkflowRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.TestIntegrationWorkflowHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminReplayWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.ReplayWebhookRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.WebhookID = strings.TrimSpace(r.PathValue("webhook_id"))
	if req.WebhookID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "webhook_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.ReplayWebhookHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminDisableWebhook(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.DisableWebhookRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	req.WebhookID = strings.TrimSpace(r.PathValue("webhook_id"))
	if req.WebhookID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "webhook_id is required")
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.DisableWebhookHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminGetWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	req := admindashboardhttp.GetWebhookDeliveriesRequest{
		WebhookID: strings.TrimSpace(r.PathValue("webhook_id")),
	}
	if req.WebhookID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "webhook_id is required")
		return
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "limit must be a valid integer")
			return
		}
		req.Limit = limit
	}
	resp, err := s.adminDashboard.Handler.GetWebhookDeliveriesHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminGetWebhookAnalytics(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	req := admindashboardhttp.GetWebhookAnalyticsRequest{
		WebhookID: strings.TrimSpace(r.PathValue("webhook_id")),
	}
	if req.WebhookID == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "webhook_id is required")
		return
	}
	resp, err := s.adminDashboard.Handler.GetWebhookAnalyticsHandler(r.Context(), adminID, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminCreateMigrationPlan(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.CreateMigrationPlanRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.CreateMigrationPlanHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminListMigrationPlans(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}

	resp, err := s.adminDashboard.Handler.ListMigrationPlansHandler(r.Context(), adminID)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminStartMigrationRun(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	adminID, ok := requireAdminID(w, r)
	if !ok {
		return
	}
	if !s.requireAdminControlPlanePermission(w, r, adminID, "policy.manage") {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req admindashboardhttp.StartMigrationRunRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	if strings.TrimSpace(req.SourceIP) == "" {
		req.SourceIP = resolveClientIP(r)
	}
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = getRequestID(r)
	}
	resp, err := s.adminDashboard.Handler.StartMigrationRunHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeAdminDashboardDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
