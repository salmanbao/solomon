package memory

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type Store struct {
	mu                sync.Mutex
	logs              []ports.AuditLog
	idempotency       map[string]ports.IdempotencyRecord
	consents          map[string]ports.ConsentRecordResult
	portability       map[string]ports.PortabilityRequestResult
	retentionHolds    map[string]ports.RetentionHoldResult
	legalHolds        map[string]ports.LegalHoldResult
	legalReports      map[string]ports.LegalComplianceReportResult
	supportTickets    map[string]ports.SupportTicketResult
	migrationPlans    map[string]ports.MigrationPlanResult
	migrationRuns     map[string]ports.MigrationRunResult
	webhookStatus     map[string]ports.WebhookEndpointResult
	webhookDeliveries map[string][]ports.WebhookDeliveryResult
	sequence          int64
}

func NewStore() *Store {
	return &Store{
		logs:              make([]ports.AuditLog, 0, 128),
		idempotency:       map[string]ports.IdempotencyRecord{},
		consents:          map[string]ports.ConsentRecordResult{},
		portability:       map[string]ports.PortabilityRequestResult{},
		retentionHolds:    map[string]ports.RetentionHoldResult{},
		legalHolds:        map[string]ports.LegalHoldResult{},
		legalReports:      map[string]ports.LegalComplianceReportResult{},
		supportTickets:    map[string]ports.SupportTicketResult{},
		migrationPlans:    map[string]ports.MigrationPlanResult{},
		migrationRuns:     map[string]ports.MigrationRunResult{},
		webhookStatus:     map[string]ports.WebhookEndpointResult{},
		webhookDeliveries: map[string][]ports.WebhookDeliveryResult{},
	}
}

func (s *Store) AppendAuditLog(_ context.Context, row ports.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, row)
	return nil
}

func (s *Store) ListRecentAuditLogs(_ context.Context, limit int) ([]ports.AuditLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	out := make([]ports.AuditLog, 0, limit)
	for i := len(s.logs) - 1; i >= 0; i-- {
		out = append(out, s.logs[i])
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) Get(_ context.Context, key string, now time.Time) (*ports.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.idempotency[key]
	if !ok {
		return nil, nil
	}
	if now.After(row.ExpiresAt) {
		delete(s.idempotency, key)
		return nil, nil
	}
	clone := row
	clone.ResponseBody = slices.Clone(row.ResponseBody)
	return &clone, nil
}

func (s *Store) Reserve(_ context.Context, key, requestHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if row, ok := s.idempotency[key]; ok && time.Now().UTC().Before(row.ExpiresAt) {
		if row.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[key] = ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		ExpiresAt:   expiresAt,
	}
	return nil
}

func (s *Store) Complete(_ context.Context, key string, responseBody []byte, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.idempotency[key]
	if !ok {
		return nil
	}
	row.ResponseBody = slices.Clone(responseBody)
	if at.After(row.ExpiresAt) {
		row.ExpiresAt = at.Add(7 * 24 * time.Hour)
	}
	s.idempotency[key] = row
	return nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) GrantRole(
	_ context.Context,
	_ string,
	userID string,
	roleID string,
	_ string,
	_ string,
) (ports.RoleGrantResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	return ports.RoleGrantResult{
		AssignmentID: "asg_" + formatSeq(s.sequence),
		UserID:       userID,
		RoleID:       roleID,
		AuditLogID:   "authz_audit_" + formatSeq(s.sequence),
	}, nil
}

func (s *Store) ApproveSubmission(
	_ context.Context,
	moderatorID string,
	submissionID string,
	campaignID string,
	reason string,
	notes string,
	_ string,
) (ports.ModerationDecisionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.ModerationDecisionResult{
		DecisionID:   "mod_" + formatSeq(s.sequence),
		SubmissionID: submissionID,
		CampaignID:   campaignID,
		ModeratorID:  moderatorID,
		Action:       "approved",
		Reason:       reason,
		Notes:        notes,
		QueueStatus:  "approved",
		CreatedAt:    now,
	}, nil
}

func (s *Store) RejectSubmission(
	_ context.Context,
	moderatorID string,
	submissionID string,
	campaignID string,
	reason string,
	notes string,
	_ string,
) (ports.ModerationDecisionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.ModerationDecisionResult{
		DecisionID:   "mod_" + formatSeq(s.sequence),
		SubmissionID: submissionID,
		CampaignID:   campaignID,
		ModeratorID:  moderatorID,
		Action:       "rejected",
		Reason:       reason,
		Notes:        notes,
		QueueStatus:  "rejected",
		CreatedAt:    now,
	}, nil
}

func (s *Store) ReleaseLockout(
	_ context.Context,
	_ string,
	userID string,
	_ string,
	_ string,
) (ports.AbuseLockoutResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.AbuseLockoutResult{
		ThreatID:        "abuse_" + formatSeq(s.sequence),
		UserID:          userID,
		Status:          "released",
		ReleasedAt:      now,
		OwnerAuditLogID: "abuse_audit_" + formatSeq(s.sequence),
	}, nil
}

func (s *Store) CreateRefund(
	_ context.Context,
	_ string,
	transactionID string,
	userID string,
	amount float64,
	reason string,
	_ string,
) (ports.FinanceRefundResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.FinanceRefundResult{
		RefundID:      "refund_" + formatSeq(s.sequence),
		TransactionID: transactionID,
		UserID:        userID,
		Amount:        amount,
		Reason:        reason,
		CreatedAt:     now,
	}, nil
}

func (s *Store) CreateInvoiceRefund(
	_ context.Context,
	_ string,
	invoiceID string,
	lineItemID string,
	amount float64,
	reason string,
	_ string,
) (ports.BillingRefundResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.BillingRefundResult{
		RefundID:    "billing_refund_" + formatSeq(s.sequence),
		InvoiceID:   invoiceID,
		LineItemID:  lineItemID,
		Amount:      amount,
		Reason:      reason,
		ProcessedAt: now,
	}, nil
}

func (s *Store) RecalculateReward(
	_ context.Context,
	_ string,
	userID string,
	submissionID string,
	campaignID string,
	_ int64,
	_ float64,
	_ float64,
	_ time.Time,
	_ string,
	_ string,
) (ports.RewardRecalculationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	eligibleAt := now
	return ports.RewardRecalculationResult{
		SubmissionID:  submissionID,
		UserID:        userID,
		CampaignID:    campaignID,
		Status:        "reward_eligible",
		NetAmount:     12.5,
		RolloverTotal: 0,
		CalculatedAt:  now,
		EligibleAt:    &eligibleAt,
	}, nil
}

func (s *Store) SuspendAffiliate(
	_ context.Context,
	_ string,
	affiliateID string,
	_ string,
	_ string,
) (ports.AffiliateSuspensionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	return ports.AffiliateSuspensionResult{
		AffiliateID: affiliateID,
		Status:      "suspended",
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

func (s *Store) CreateAttribution(
	_ context.Context,
	_ string,
	affiliateID string,
	_ string,
	orderID string,
	_ string,
	amount float64,
	_ string,
	_ string,
	_ string,
) (ports.AffiliateAttributionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	return ports.AffiliateAttributionResult{
		AttributionID: "attr_" + formatSeq(s.sequence),
		AffiliateID:   affiliateID,
		OrderID:       orderID,
		Amount:        amount,
		AttributedAt:  time.Now().UTC(),
	}, nil
}

func (s *Store) RetryFailedPayout(
	_ context.Context,
	_ string,
	payoutID string,
	_ string,
	_ string,
) (ports.PayoutRetryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.PayoutRetryResult{
		PayoutID:    payoutID,
		UserID:      "user_" + formatSeq(s.sequence),
		Status:      "paid",
		ProcessedAt: now,
	}, nil
}

func (s *Store) ResolveDispute(
	_ context.Context,
	_ string,
	disputeID string,
	action string,
	_ string,
	_ string,
	refundAmount float64,
	_ string,
) (ports.DisputeResolutionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	status := "resolved"
	resolutionType := "refund_issued"
	if action == "reopen" {
		status = "reopened"
		resolutionType = ""
		refundAmount = 0
	}
	return ports.DisputeResolutionResult{
		DisputeID:      disputeID,
		Status:         status,
		ResolutionType: resolutionType,
		RefundAmount:   refundAmount,
		ProcessedAt:    now,
	}, nil
}

func (s *Store) GetConsent(
	_ context.Context,
	_ string,
	userID string,
) (ports.ConsentRecordResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.consents[userID]
	if !ok {
		return ports.ConsentRecordResult{}, domainerrors.ErrNotFound
	}
	return row, nil
}

func (s *Store) UpdateConsent(
	_ context.Context,
	adminID string,
	userID string,
	preferences map[string]bool,
	_ string,
	_ string,
) (ports.ConsentChangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	row := ports.ConsentRecordResult{
		UserID:        userID,
		Status:        "active",
		Preferences:   cloneBoolMap(preferences),
		LastUpdated:   now,
		LastUpdatedBy: adminID,
	}
	s.consents[userID] = row
	return ports.ConsentChangeResult{
		UserID:    userID,
		Status:    row.Status,
		UpdatedAt: now,
	}, nil
}

func (s *Store) WithdrawConsent(
	_ context.Context,
	adminID string,
	userID string,
	category string,
	_ string,
	_ string,
) (ports.ConsentChangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	row, ok := s.consents[userID]
	if !ok {
		row = ports.ConsentRecordResult{
			UserID:        userID,
			Preferences:   map[string]bool{},
			LastUpdatedBy: adminID,
		}
	}
	if strings.TrimSpace(category) == "" || strings.EqualFold(strings.TrimSpace(category), "all") {
		for key := range row.Preferences {
			row.Preferences[key] = false
		}
	} else {
		row.Preferences[strings.TrimSpace(category)] = false
	}
	row.Status = "withdrawn"
	row.LastUpdated = now
	row.LastUpdatedBy = adminID
	s.consents[userID] = row
	return ports.ConsentChangeResult{
		UserID:    userID,
		Status:    row.Status,
		UpdatedAt: now,
	}, nil
}

func (s *Store) CreateExport(
	_ context.Context,
	_ string,
	userID string,
	format string,
	_ string,
	_ string,
) (ports.PortabilityRequestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	if format == "" {
		format = "json"
	}
	row := ports.PortabilityRequestResult{
		RequestID:   "exp_" + formatSeq(s.sequence),
		UserID:      userID,
		RequestType: "export",
		Format:      format,
		Status:      "completed",
		RequestedAt: now,
		CompletedAt: &now,
		DownloadURL: "https://downloads.example.com/v1/exports/exp_" + formatSeq(s.sequence),
	}
	s.portability[row.RequestID] = row
	return row, nil
}

func (s *Store) GetExport(
	_ context.Context,
	_ string,
	requestID string,
) (ports.PortabilityRequestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.portability[requestID]
	if !ok {
		return ports.PortabilityRequestResult{}, domainerrors.ErrNotFound
	}
	return row, nil
}

func (s *Store) CreateEraseRequest(
	_ context.Context,
	_ string,
	userID string,
	reason string,
	_ string,
) (ports.PortabilityRequestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	row := ports.PortabilityRequestResult{
		RequestID:   "erase_" + formatSeq(s.sequence),
		UserID:      userID,
		RequestType: "erase",
		Status:      "completed",
		Reason:      reason,
		RequestedAt: now,
		CompletedAt: &now,
	}
	s.portability[row.RequestID] = row
	return row, nil
}

func (s *Store) CreateLegalHold(
	_ context.Context,
	_ string,
	entityID string,
	dataType string,
	reason string,
	expiresAt *time.Time,
	_ string,
) (ports.RetentionHoldResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	row := ports.RetentionHoldResult{
		HoldID:    "ret_hold_" + formatSeq(s.sequence),
		EntityID:  entityID,
		DataType:  dataType,
		Reason:    reason,
		Status:    "active",
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}
	s.retentionHolds[row.HoldID] = row
	return row, nil
}

func (s *Store) CheckHold(
	_ context.Context,
	_ string,
	entityType string,
	entityID string,
) (ports.LegalHoldCheckResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range s.legalHolds {
		if row.EntityType == entityType && row.EntityID == entityID && row.Status == "active" {
			return ports.LegalHoldCheckResult{
				EntityType: entityType,
				EntityID:   entityID,
				Held:       true,
				HoldID:     row.HoldID,
			}, nil
		}
	}
	return ports.LegalHoldCheckResult{
		EntityType: entityType,
		EntityID:   entityID,
		Held:       false,
	}, nil
}

func (s *Store) ReleaseHold(
	_ context.Context,
	_ string,
	holdID string,
	_ string,
	_ string,
) (ports.LegalHoldResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.legalHolds[holdID]
	if !ok {
		return ports.LegalHoldResult{}, domainerrors.ErrNotFound
	}
	now := time.Now().UTC()
	row.Status = "released"
	row.ReleasedAt = &now
	s.legalHolds[holdID] = row
	return row, nil
}

func (s *Store) RunComplianceScan(
	_ context.Context,
	_ string,
	reportType string,
	_ string,
) (ports.LegalComplianceReportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	if strings.TrimSpace(reportType) == "" {
		reportType = "manual"
	}
	row := ports.LegalComplianceReportResult{
		ReportID:      "scan_" + formatSeq(s.sequence),
		ReportType:    reportType,
		Status:        "completed",
		FindingsCount: 0,
		DownloadURL:   "https://downloads.example.com/legal/compliance/" + reportType + ".pdf",
		CreatedAt:     now,
	}
	s.legalReports[row.ReportID] = row
	return row, nil
}

func (s *Store) GetTicket(
	_ context.Context,
	_ string,
	ticketID string,
) (ports.SupportTicketResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, ok := s.supportTickets[strings.TrimSpace(ticketID)]
	if !ok {
		return ports.SupportTicketResult{}, domainerrors.ErrNotFound
	}
	return row, nil
}

func (s *Store) SearchTickets(
	_ context.Context,
	_ string,
	filter ports.SupportTicketSearchFilter,
) ([]ports.SupportTicketResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	category := strings.ToLower(strings.TrimSpace(filter.Category))
	assigned := strings.TrimSpace(filter.AssignedTo)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	out := make([]ports.SupportTicketResult, 0, limit)
	for _, row := range s.supportTickets {
		if query != "" && !strings.Contains(strings.ToLower(row.Subject+" "+row.Description), query) {
			continue
		}
		if status != "" && strings.ToLower(row.Status) != status {
			continue
		}
		if category != "" && strings.ToLower(row.Category) != category {
			continue
		}
		if assigned != "" && row.AssignedAgentID != assigned {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) AssignTicket(
	_ context.Context,
	_ string,
	ticketID string,
	agentID string,
	_ string,
	_ string,
) (ports.SupportTicketResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(ticketID)
	if id == "" {
		return ports.SupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	now := time.Now().UTC()
	row, ok := s.supportTickets[id]
	if !ok {
		row = ports.SupportTicketResult{
			TicketID:         id,
			UserID:           "user_" + formatSeq(s.sequence+1),
			Subject:          "support ticket",
			Description:      "autogenerated fallback ticket",
			Category:         "Other",
			Priority:         "normal",
			Status:           "open",
			SubStatus:        "new",
			SLAResponseDueAt: now.Add(24 * time.Hour),
			LastActivityAt:   now,
			UpdatedAt:        now,
		}
	}
	row.AssignedAgentID = strings.TrimSpace(agentID)
	row.UpdatedAt = now
	row.LastActivityAt = now
	s.supportTickets[id] = row
	return row, nil
}

func (s *Store) UpdateTicket(
	_ context.Context,
	_ string,
	ticketID string,
	status string,
	subStatus string,
	priority string,
	_ string,
	_ string,
) (ports.SupportTicketResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(ticketID)
	if id == "" {
		return ports.SupportTicketResult{}, domainerrors.ErrInvalidInput
	}
	now := time.Now().UTC()
	row, ok := s.supportTickets[id]
	if !ok {
		row = ports.SupportTicketResult{
			TicketID:         id,
			UserID:           "user_" + formatSeq(s.sequence+1),
			Subject:          "support ticket",
			Description:      "autogenerated fallback ticket",
			Category:         "Other",
			Priority:         "normal",
			Status:           "open",
			SubStatus:        "new",
			SLAResponseDueAt: now.Add(24 * time.Hour),
			LastActivityAt:   now,
			UpdatedAt:        now,
		}
	}
	if v := strings.TrimSpace(status); v != "" {
		row.Status = v
	}
	if v := strings.TrimSpace(subStatus); v != "" {
		row.SubStatus = v
	}
	if v := strings.TrimSpace(priority); v != "" {
		row.Priority = v
	}
	row.UpdatedAt = now
	row.LastActivityAt = now
	s.supportTickets[id] = row
	return row, nil
}

func (s *Store) SaveCampaign(
	_ context.Context,
	_ string,
	_ string,
	campaignID string,
	_ string,
) (ports.EditorCampaignSaveResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	return ports.EditorCampaignSaveResult{
		CampaignID: strings.TrimSpace(campaignID),
		Saved:      true,
		SavedAt:    &now,
	}, nil
}

func (s *Store) RequestExport(
	_ context.Context,
	_ string,
	_ string,
	projectID string,
	_ string,
	_ string,
	_ int,
	_ string,
	_ string,
	_ string,
) (ports.ClippingExportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.ClippingExportResult{
		ExportID:        "export_" + formatSeq(s.sequence),
		ProjectID:       strings.TrimSpace(projectID),
		Status:          "queued",
		ProgressPercent: 0,
		ProviderJobID:   "provider_job_" + formatSeq(s.sequence),
		CreatedAt:       now,
	}, nil
}

func (s *Store) DeployModel(
	_ context.Context,
	_ string,
	input ports.AutoClippingModelDeployInput,
	_ string,
) (ports.AutoClippingModelDeployResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	return ports.AutoClippingModelDeployResult{
		ModelVersionID:   "model_" + formatSeq(s.sequence),
		DeploymentStatus: "canary_" + strconv.Itoa(input.CanaryPercentage) + "pct",
		DeployedAt:       now,
		Message:          "model deployment accepted",
	}, nil
}

func (s *Store) RotateAPIKey(
	_ context.Context,
	_ string,
	keyID string,
	_ string,
) (ports.IntegrationKeyRotationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	trimmed := strings.TrimSpace(keyID)
	if trimmed == "" {
		trimmed = "key_" + formatSeq(s.sequence)
	}
	return ports.IntegrationKeyRotationResult{
		RotationID:  "rot_" + formatSeq(s.sequence),
		DeveloperID: "dev_" + formatSeq(s.sequence),
		OldKeyID:    trimmed,
		NewKeyID:    "key_new_" + formatSeq(s.sequence),
		CreatedAt:   now,
	}, nil
}

func (s *Store) TestWorkflow(
	_ context.Context,
	_ string,
	workflowID string,
	_ string,
) (ports.IntegrationWorkflowTestResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	return ports.IntegrationWorkflowTestResult{
		ExecutionID: "exec_" + formatSeq(s.sequence),
		WorkflowID:  strings.TrimSpace(workflowID),
		Status:      "success",
		TestRun:     true,
		StartedAt:   time.Now().UTC(),
	}, nil
}

func (s *Store) ReplayWebhook(
	_ context.Context,
	_ string,
	webhookID string,
	_ string,
) (ports.WebhookReplayResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	id := strings.TrimSpace(webhookID)
	if id == "" {
		id = "wh_" + formatSeq(s.sequence)
	}
	now := time.Now().UTC()
	delivery := ports.WebhookDeliveryResult{
		DeliveryID:      "del_" + formatSeq(s.sequence),
		WebhookID:       id,
		OriginalEventID: "event_" + formatSeq(s.sequence),
		OriginalType:    "webhook.replay",
		HTTPStatus:      http.StatusOK,
		LatencyMS:       42,
		RetryCount:      1,
		DeliveredAt:     now,
		IsTest:          false,
		Success:         true,
	}
	s.webhookDeliveries[id] = append(s.webhookDeliveries[id], delivery)
	if _, ok := s.webhookStatus[id]; !ok {
		s.webhookStatus[id] = ports.WebhookEndpointResult{
			WebhookID: id,
			Status:    "active",
			UpdatedAt: now,
		}
	}
	return ports.WebhookReplayResult{
		DeliveryID: delivery.DeliveryID,
		WebhookID:  delivery.WebhookID,
		Status:     "success",
		HTTPStatus: delivery.HTTPStatus,
		LatencyMS:  delivery.LatencyMS,
		Timestamp:  now,
	}, nil
}

func (s *Store) DisableWebhook(
	_ context.Context,
	_ string,
	webhookID string,
	_ string,
) (ports.WebhookEndpointResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	id := strings.TrimSpace(webhookID)
	if id == "" {
		id = "wh_" + formatSeq(s.sequence+1)
	}
	row := ports.WebhookEndpointResult{
		WebhookID: id,
		Status:    "disabled",
		UpdatedAt: now,
	}
	s.webhookStatus[id] = row
	return row, nil
}

func (s *Store) ListDeliveries(
	_ context.Context,
	_ string,
	webhookID string,
	limit int,
) ([]ports.WebhookDeliveryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(webhookID)
	rows := s.webhookDeliveries[id]
	if limit <= 0 {
		limit = 50
	}
	if len(rows) > limit {
		rows = rows[len(rows)-limit:]
	}
	out := make([]ports.WebhookDeliveryResult, len(rows))
	copy(out, rows)
	return out, nil
}

func (s *Store) GetAnalytics(
	_ context.Context,
	_ string,
	webhookID string,
) (ports.WebhookAnalyticsResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := strings.TrimSpace(webhookID)
	rows := s.webhookDeliveries[id]
	total := int64(len(rows))
	success := int64(0)
	failed := int64(0)
	var latencySum int64
	for _, row := range rows {
		if row.Success {
			success++
		} else {
			failed++
		}
		latencySum += row.LatencyMS
	}
	rate := 0.0
	avg := 0.0
	if total > 0 {
		rate = float64(success) / float64(total)
		avg = float64(latencySum) / float64(total)
	}
	return ports.WebhookAnalyticsResult{
		TotalDeliveries:      total,
		SuccessfulDeliveries: success,
		FailedDeliveries:     failed,
		SuccessRate:          rate,
		AvgLatencyMS:         avg,
		P95LatencyMS:         avg,
		P99LatencyMS:         avg,
		ByEventType: map[string]ports.WebhookAnalyticsMetrics{
			"webhook.replay": {
				Total:      total,
				Success:    success,
				Failed:     failed,
				AvgLatency: avg,
			},
		},
	}, nil
}

func (s *Store) CreatePlan(
	_ context.Context,
	adminID string,
	serviceName string,
	environment string,
	version string,
	plan map[string]interface{},
	dryRun bool,
	riskLevel string,
	_ string,
) (ports.MigrationPlanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	row := ports.MigrationPlanResult{
		PlanID:           "plan_" + formatSeq(s.sequence),
		ServiceName:      strings.TrimSpace(serviceName),
		Environment:      strings.TrimSpace(environment),
		Version:          strings.TrimSpace(version),
		Plan:             plan,
		Status:           "validated",
		DryRun:           dryRun,
		RiskLevel:        strings.TrimSpace(riskLevel),
		StagingValidated: true,
		BackupRequired:   true,
		CreatedBy:        strings.TrimSpace(adminID),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if row.RiskLevel == "" {
		row.RiskLevel = "medium"
	}
	s.migrationPlans[row.PlanID] = row
	return row, nil
}

func (s *Store) ListPlans(
	_ context.Context,
	_ string,
) ([]ports.MigrationPlanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ports.MigrationPlanResult, 0, len(s.migrationPlans))
	for _, row := range s.migrationPlans {
		out = append(out, row)
	}
	return out, nil
}

func (s *Store) CreateRun(
	_ context.Context,
	adminID string,
	planID string,
	_ string,
) (ports.MigrationRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	now := time.Now().UTC()
	row := ports.MigrationRunResult{
		RunID:             "run_" + formatSeq(s.sequence),
		PlanID:            strings.TrimSpace(planID),
		Status:            "completed",
		OperatorID:        strings.TrimSpace(adminID),
		SnapshotCreated:   true,
		RollbackAvailable: true,
		ValidationStatus:  "passed",
		BackfillJobID:     "bf_" + formatSeq(s.sequence),
		StartedAt:         now,
		CompletedAt:       now.Add(2 * time.Minute),
	}
	s.migrationRuns[row.RunID] = row
	return row, nil
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	if len(in) == 0 {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func formatSeq(v int64) string {
	return strconv.FormatInt(v, 10)
}
