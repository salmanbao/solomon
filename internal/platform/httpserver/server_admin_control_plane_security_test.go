package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type adminControlPlaneGrantRoleResponse struct {
	AssignmentID        string `json:"assignment_id"`
	UserID              string `json:"user_id"`
	RoleID              string `json:"role_id"`
	OwnerAuditLogID     string `json:"owner_audit_log_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
	OccurredAt          string `json:"occurred_at"`
}

type adminControlPlaneModerationResponse struct {
	DecisionID          string `json:"decision_id"`
	SubmissionID        string `json:"submission_id"`
	CampaignID          string `json:"campaign_id"`
	ModeratorID         string `json:"moderator_id"`
	Action              string `json:"action"`
	Reason              string `json:"reason"`
	Notes               string `json:"notes"`
	QueueStatus         string `json:"queue_status"`
	CreatedAt           string `json:"created_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneAbuseLockoutResponse struct {
	ThreatID            string `json:"threat_id"`
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	ReleasedAt          string `json:"released_at"`
	OwnerAuditLogID     string `json:"owner_audit_log_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneFinanceRefundResponse struct {
	RefundID            string  `json:"refund_id"`
	TransactionID       string  `json:"transaction_id"`
	UserID              string  `json:"user_id"`
	Amount              float64 `json:"amount"`
	Reason              string  `json:"reason"`
	CreatedAt           string  `json:"created_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type adminControlPlaneBillingRefundResponse struct {
	RefundID            string  `json:"refund_id"`
	InvoiceID           string  `json:"invoice_id"`
	LineItemID          string  `json:"line_item_id"`
	Amount              float64 `json:"amount"`
	Reason              string  `json:"reason"`
	ProcessedAt         string  `json:"processed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type adminControlPlaneRewardRecalculateResponse struct {
	SubmissionID        string  `json:"submission_id"`
	UserID              string  `json:"user_id"`
	CampaignID          string  `json:"campaign_id"`
	Status              string  `json:"status"`
	NetAmount           float64 `json:"net_amount"`
	RolloverTotal       float64 `json:"rollover_total"`
	CalculatedAt        string  `json:"calculated_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type adminControlPlaneAffiliateSuspendResponse struct {
	AffiliateID         string `json:"affiliate_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneAffiliateAttributionResponse struct {
	AttributionID       string  `json:"attribution_id"`
	AffiliateID         string  `json:"affiliate_id"`
	OrderID             string  `json:"order_id"`
	Amount              float64 `json:"amount"`
	AttributedAt        string  `json:"attributed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type adminControlPlanePayoutRetryResponse struct {
	PayoutID            string `json:"payout_id"`
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	FailureReason       string `json:"failure_reason"`
	ProcessedAt         string `json:"processed_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneDisputeResponse struct {
	DisputeID           string  `json:"dispute_id"`
	Status              string  `json:"status"`
	ResolutionType      string  `json:"resolution_type"`
	RefundAmount        float64 `json:"refund_amount"`
	ProcessedAt         string  `json:"processed_at"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
}

type adminControlPlaneConsentResponse struct {
	UserID              string `json:"user_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneDataExportResponse struct {
	RequestID           string  `json:"request_id"`
	UserID              string  `json:"user_id"`
	Status              string  `json:"status"`
	ControlPlaneAuditID string  `json:"control_plane_audit_id"`
	CompletedAt         *string `json:"completed_at"`
}

type adminControlPlaneLegalHoldCheckResponse struct {
	EntityType          string `json:"entity_type"`
	EntityID            string `json:"entity_id"`
	Held                bool   `json:"held"`
	HoldID              string `json:"hold_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneComplianceScanResponse struct {
	ReportID            string `json:"report_id"`
	ReportType          string `json:"report_type"`
	Status              string `json:"status"`
	FindingsCount       int    `json:"findings_count"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneSupportTicketResponse struct {
	TicketID            string `json:"ticket_id"`
	Status              string `json:"status"`
	SubStatus           string `json:"sub_status"`
	AssignedAgentID     string `json:"assigned_agent_id"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneRotateIntegrationKeyResponse struct {
	RotationID          string `json:"rotation_id"`
	DeveloperID         string `json:"developer_id"`
	OldKeyID            string `json:"old_key_id"`
	NewKeyID            string `json:"new_key_id"`
	CreatedAt           string `json:"created_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneReplayWebhookResponse struct {
	DeliveryID          string `json:"delivery_id"`
	WebhookID           string `json:"webhook_id"`
	Status              string `json:"status"`
	HTTPStatus          int    `json:"http_status"`
	LatencyMS           int64  `json:"latency_ms"`
	Timestamp           string `json:"timestamp"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneDisableWebhookResponse struct {
	WebhookID           string `json:"webhook_id"`
	Status              string `json:"status"`
	UpdatedAt           string `json:"updated_at"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneMigrationPlanResponse struct {
	PlanID              string `json:"plan_id"`
	ServiceName         string `json:"service_name"`
	Environment         string `json:"environment"`
	Version             string `json:"version"`
	Status              string `json:"status"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneMigrationRunResponse struct {
	RunID               string `json:"run_id"`
	PlanID              string `json:"plan_id"`
	Status              string `json:"status"`
	ControlPlaneAuditID string `json:"control_plane_audit_id"`
}

type adminControlPlaneWebhookDeliveriesResponse struct {
	Deliveries []struct {
		DeliveryID string `json:"delivery_id"`
	} `json:"deliveries"`
}

type adminControlPlaneWebhookAnalyticsResponse struct {
	TotalDeliveries int64 `json:"total_deliveries"`
}

type controlPlaneErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func adminHeaders(req *http.Request, adminID string, idempotencyKey string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-admin-cp-1")
	req.Header.Set("X-MFA-Code", "123456")
	req.Header.Set("X-Admin-Id", adminID)
	req.Header.Set("Idempotency-Key", idempotencyKey)
}

func TestAdminControlPlaneGrantRoleRequiresMFA(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity/roles/grant", bytes.NewReader([]byte(`{
		"user_id":"user-100",
		"role_id":"editor",
		"reason":"moderation staffing"
	}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Request-Id", "req-admin-cp-no-mfa")
	req.Header.Set("X-Admin-Id", "admin-1")
	req.Header.Set("Idempotency-Key", "idem-admin-cp-no-mfa")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneGrantRoleIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()

	bodyOne := []byte(`{
		"user_id":"user-100",
		"role_id":"editor",
		"reason":"moderation staffing"
	}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity/roles/grant", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-grant-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first grant 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneGrantRoleResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first grant response: %v", err)
	}
	if first.AssignmentID == "" || first.ControlPlaneAuditID == "" {
		t.Fatalf("expected assignment and control-plane audit IDs")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity/roles/grant", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-grant-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay grant 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneGrantRoleResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay grant response: %v", err)
	}
	if replay.AssignmentID != first.AssignmentID {
		t.Fatalf("expected replay assignment id %q, got %q", first.AssignmentID, replay.AssignmentID)
	}

	bodyConflict := []byte(`{
		"user_id":"user-100",
		"role_id":"brand",
		"reason":"changed payload with same key"
	}`)
	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity/roles/grant", bytes.NewReader(bodyConflict))
	adminHeaders(req3, "admin-1", "idem-admin-cp-grant-1")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestAdminControlPlaneGrantRoleOwnerForbiddenMapsToForbidden(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/identity/roles/grant", bytes.NewReader([]byte(`{
		"user_id":"user-200",
		"role_id":"editor",
		"reason":"should fail permission"
	}`)))
	adminHeaders(req, "viewer-1", "idem-admin-cp-forbidden")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneModerationOwnerNotFoundMapsToNotFound(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/moderation/decisions", bytes.NewReader([]byte(`{
		"submission_id":"missing-submission",
		"campaign_id":"camp-1",
		"action":"reject",
		"reason":"not found should map"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-mod-notfound")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var er controlPlaneErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Code != "not_found" {
		t.Fatalf("expected not_found code, got %q", er.Code)
	}
}

func TestAdminControlPlaneModerationEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/moderation/decisions", bytes.NewReader([]byte(`{
		"submission_id":"sub-1",
		"campaign_id":"camp-1",
		"action":"approve",
		"reason":"policy pass"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-mod-audit")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var response adminControlPlaneModerationResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ControlPlaneAuditID == "" {
		t.Fatalf("expected control-plane audit id")
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected audit logs")
	}
	if logs[0].Action != "admin.submission.moderated" {
		t.Fatalf("expected admin.submission.moderated action, got %q", logs[0].Action)
	}
}

func TestAdminControlPlaneRecordActionRouteAvailable(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/actions/log", bytes.NewReader([]byte(`{
		"action":"admin.manual.review",
		"target_id":"sub-1",
		"justification":"manual intervention"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-action-log")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneAbuseLockoutReleaseForbiddenByScope(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/locked-user-1/release", bytes.NewReader([]byte(`{
		"reason":"manual recovery"
	}`)))
	adminHeaders(req, "viewer-1", "idem-admin-cp-abuse-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneAbuseLockoutReleaseIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()
	bodyOne := []byte(`{
		"reason":"manual false-positive recovery"
	}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/locked-user-1/release", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-abuse-2")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first release 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneAbuseLockoutResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first release response: %v", err)
	}
	if first.ControlPlaneAuditID == "" || first.OwnerAuditLogID == "" || first.ThreatID == "" {
		t.Fatalf("expected threat/audit ids in first response")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/locked-user-1/release", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-abuse-2")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay release 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneAbuseLockoutResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay release response: %v", err)
	}
	if replay.ThreatID != first.ThreatID {
		t.Fatalf("expected replay threat id %q, got %q", first.ThreatID, replay.ThreatID)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/locked-user-1/release", bytes.NewReader([]byte(`{
		"reason":"changed reason with same key"
	}`)))
	adminHeaders(req3, "admin-1", "idem-admin-cp-abuse-2")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestAdminControlPlaneAbuseLockoutOwnerNotFoundMapsToNotFound(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/missing-user/release", bytes.NewReader([]byte(`{
		"reason":"should map owner not_found"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-abuse-3")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	var er controlPlaneErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &er); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if er.Code != "not_found" {
		t.Fatalf("expected not_found code, got %q", er.Code)
	}
}

func TestAdminControlPlaneAbuseLockoutEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/abuse-prevention/lockouts/user-200/release", bytes.NewReader([]byte(`{
		"reason":"manual unlock after review"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-abuse-4")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var response adminControlPlaneAbuseLockoutResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ControlPlaneAuditID == "" {
		t.Fatalf("expected control-plane audit id")
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected audit logs")
	}
	if logs[0].Action != "admin.abuse.lockout.released" {
		t.Fatalf("expected admin.abuse.lockout.released action, got %q", logs[0].Action)
	}
}

func TestAdminControlPlaneFinanceRefundForbiddenByScope(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader([]byte(`{
		"transaction_id":"txn-1",
		"user_id":"user-1",
		"amount":25,
		"reason":"duplicate charge"
	}`)))
	adminHeaders(req, "viewer-1", "idem-admin-cp-fin-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneFinanceRefundIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()
	bodyOne := []byte(`{
		"transaction_id":"txn-1",
		"user_id":"user-1",
		"amount":25,
		"reason":"duplicate charge"
	}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-fin-2")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first finance refund 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneFinanceRefundResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first refund response: %v", err)
	}
	if first.RefundID == "" || first.ControlPlaneAuditID == "" {
		t.Fatalf("expected refund and audit IDs")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-fin-2")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay refund 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneFinanceRefundResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay refund response: %v", err)
	}
	if replay.RefundID != first.RefundID {
		t.Fatalf("expected replay refund id %q, got %q", first.RefundID, replay.RefundID)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader([]byte(`{
		"transaction_id":"txn-1",
		"user_id":"user-1",
		"amount":30,
		"reason":"changed payload"
	}`)))
	adminHeaders(req3, "admin-1", "idem-admin-cp-fin-2")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestAdminControlPlanePayoutRetryEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/payouts/pay-1/retry", bytes.NewReader([]byte(`{
		"reason":"provider outage recovered"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-pay-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var response adminControlPlanePayoutRetryResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode payout retry response: %v", err)
	}
	if response.PayoutID == "" || response.ControlPlaneAuditID == "" {
		t.Fatalf("expected payout and audit IDs")
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.payout.retry.requested" {
		t.Fatalf("expected admin.payout.retry.requested action, got %+v", logs)
	}
}

func TestAdminControlPlaneDisputeResolveAndReopen(t *testing.T) {
	server := newTestServer()
	resolveReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/disputes/dispute-1/resolve", bytes.NewReader([]byte(`{
		"action":"resolve",
		"reason":"validated evidence",
		"refund_amount":12.5
	}`)))
	adminHeaders(resolveReq, "admin-1", "idem-admin-cp-dsp-1")
	resolveRR := httptest.NewRecorder()
	server.mux.ServeHTTP(resolveRR, resolveReq)
	if resolveRR.Code != http.StatusOK {
		t.Fatalf("expected resolve 200, got %d body=%s", resolveRR.Code, resolveRR.Body.String())
	}

	reopenReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/disputes/dispute-1/reopen", bytes.NewReader([]byte(`{
		"action":"reopen",
		"reason":"new evidence submitted"
	}`)))
	adminHeaders(reopenReq, "admin-1", "idem-admin-cp-dsp-2")
	reopenRR := httptest.NewRecorder()
	server.mux.ServeHTTP(reopenRR, reopenReq)
	if reopenRR.Code != http.StatusOK {
		t.Fatalf("expected reopen 200, got %d body=%s", reopenRR.Code, reopenRR.Body.String())
	}
	var reopenResponse adminControlPlaneDisputeResponse
	if err := json.Unmarshal(reopenRR.Body.Bytes(), &reopenResponse); err != nil {
		t.Fatalf("decode reopen response: %v", err)
	}
	if reopenResponse.Status != "reopened" {
		t.Fatalf("expected reopened status, got %q", reopenResponse.Status)
	}
}

func TestAdminControlPlaneBillingRefundIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()
	bodyOne := []byte(`{
		"line_item_id":"line-1",
		"amount":25,
		"reason":"duplicate charge"
	}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/billing/invoices/invoice-1/refund", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-billing-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first billing refund 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneBillingRefundResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first billing refund response: %v", err)
	}
	if first.RefundID == "" || first.ControlPlaneAuditID == "" {
		t.Fatalf("expected refund and audit IDs")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/billing/invoices/invoice-1/refund", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-billing-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay billing refund 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneBillingRefundResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay billing refund response: %v", err)
	}
	if replay.RefundID != first.RefundID {
		t.Fatalf("expected replay refund id %q, got %q", first.RefundID, replay.RefundID)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/billing/invoices/invoice-1/refund", bytes.NewReader([]byte(`{
		"line_item_id":"line-1",
		"amount":30,
		"reason":"changed payload"
	}`)))
	adminHeaders(req3, "admin-1", "idem-admin-cp-billing-1")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestAdminControlPlaneRecalculateRewardForbiddenByScope(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/rewards/recalculate", bytes.NewReader([]byte(`{
		"user_id":"user-1",
		"submission_id":"sub-1",
		"campaign_id":"camp-1",
		"locked_views":1200,
		"rate_per_1k":2.5,
		"reason":"manual correction"
	}`)))
	adminHeaders(req, "viewer-1", "idem-admin-cp-reward-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneRecalculateRewardEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/rewards/recalculate", bytes.NewReader([]byte(`{
		"user_id":"user-1",
		"submission_id":"sub-1",
		"campaign_id":"camp-1",
		"locked_views":1200,
		"rate_per_1k":2.5,
		"reason":"manual correction"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-reward-2")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var response adminControlPlaneRewardRecalculateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.SubmissionID != "sub-1" || response.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected response payload: %+v", response)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.reward.recalculated" {
		t.Fatalf("expected admin.reward.recalculated action, got %+v", logs)
	}
}

func TestAdminControlPlaneAffiliateActionsEmitAuditLog(t *testing.T) {
	server := newTestServer()

	suspendReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/affiliates/aff-1/suspend", bytes.NewReader([]byte(`{
		"reason":"fraud signal"
	}`)))
	adminHeaders(suspendReq, "admin-1", "idem-admin-cp-aff-1")
	suspendRR := httptest.NewRecorder()
	server.mux.ServeHTTP(suspendRR, suspendReq)
	if suspendRR.Code != http.StatusOK {
		t.Fatalf("expected suspend 200, got %d body=%s", suspendRR.Code, suspendRR.Body.String())
	}
	var suspendResp adminControlPlaneAffiliateSuspendResponse
	if err := json.Unmarshal(suspendRR.Body.Bytes(), &suspendResp); err != nil {
		t.Fatalf("decode suspend response: %v", err)
	}
	if suspendResp.AffiliateID != "aff-1" || suspendResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected suspend response: %+v", suspendResp)
	}

	attributeReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/affiliates/aff-1/attributions", bytes.NewReader([]byte(`{
		"order_id":"ord-1",
		"conversion_id":"conv-1",
		"amount":40,
		"currency":"USD",
		"reason":"manual attribution"
	}`)))
	adminHeaders(attributeReq, "admin-1", "idem-admin-cp-aff-2")
	attributeRR := httptest.NewRecorder()
	server.mux.ServeHTTP(attributeRR, attributeReq)
	if attributeRR.Code != http.StatusOK {
		t.Fatalf("expected attribution 200, got %d body=%s", attributeRR.Code, attributeRR.Body.String())
	}
	var attrResp adminControlPlaneAffiliateAttributionResponse
	if err := json.Unmarshal(attributeRR.Body.Bytes(), &attrResp); err != nil {
		t.Fatalf("decode attribution response: %v", err)
	}
	if attrResp.AffiliateID != "aff-1" || attrResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected attribution response: %+v", attrResp)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.affiliate.attribution.created" {
		t.Fatalf("expected latest audit action admin.affiliate.attribution.created, got %q", logs[0].Action)
	}
	if logs[1].Action != "admin.affiliate.suspended" {
		t.Fatalf("expected previous audit action admin.affiliate.suspended, got %q", logs[1].Action)
	}
}

func TestAdminControlPlaneConsentWithdrawIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()
	bodyOne := []byte(`{
		"reason":"user requested consent withdrawal",
		"category":"all"
	}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/consent/user-1/withdraw", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-consent-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first withdraw 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneConsentResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first withdraw response: %v", err)
	}
	if first.UserID != "user-1" || first.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected first withdraw response: %+v", first)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/consent/user-1/withdraw", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-consent-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay withdraw 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneConsentResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay withdraw response: %v", err)
	}
	if replay.ControlPlaneAuditID != first.ControlPlaneAuditID {
		t.Fatalf("expected replay audit id %q, got %q", first.ControlPlaneAuditID, replay.ControlPlaneAuditID)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/consent/user-1/withdraw", bytes.NewReader([]byte(`{
		"reason":"different payload"
	}`)))
	adminHeaders(req3, "admin-1", "idem-admin-cp-consent-1")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}
}

func TestAdminControlPlaneComplianceExportAndDeletionEmitAuditLog(t *testing.T) {
	server := newTestServer()

	exportReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/exports", bytes.NewReader([]byte(`{
		"user_id":"user-1",
		"format":"json",
		"reason":"dsar export"
	}`)))
	adminHeaders(exportReq, "admin-1", "idem-admin-cp-export-1")
	exportRR := httptest.NewRecorder()
	server.mux.ServeHTTP(exportRR, exportReq)
	if exportRR.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d body=%s", exportRR.Code, exportRR.Body.String())
	}
	var exportResp adminControlPlaneDataExportResponse
	if err := json.Unmarshal(exportRR.Body.Bytes(), &exportResp); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if exportResp.RequestID == "" || exportResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected export response: %+v", exportResp)
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/deletion-requests", bytes.NewReader([]byte(`{
		"user_id":"user-1",
		"reason":"gdpr deletion request"
	}`)))
	adminHeaders(deleteReq, "admin-1", "idem-admin-cp-delete-1")
	deleteRR := httptest.NewRecorder()
	server.mux.ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("expected deletion request 200, got %d body=%s", deleteRR.Code, deleteRR.Body.String())
	}
	var deleteResp adminControlPlaneDataExportResponse
	if err := json.Unmarshal(deleteRR.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode deletion response: %v", err)
	}
	if deleteResp.RequestID == "" || deleteResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected deletion response: %+v", deleteResp)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.compliance.deletion.started" {
		t.Fatalf("expected latest action admin.compliance.deletion.started, got %q", logs[0].Action)
	}
	if logs[1].Action != "admin.compliance.export.started" {
		t.Fatalf("expected previous action admin.compliance.export.started, got %q", logs[1].Action)
	}
}

func TestAdminControlPlaneLegalHoldCheckForbiddenAndAudit(t *testing.T) {
	server := newTestServer()

	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/admin/v1/compliance/legal-holds/check?entity_type=user&entity_id=user-1", nil)
	adminHeaders(forbiddenReq, "viewer-1", "idem-admin-cp-hold-check-forbidden")
	forbiddenRR := httptest.NewRecorder()
	server.mux.ServeHTTP(forbiddenRR, forbiddenReq)
	if forbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden 403, got %d body=%s", forbiddenRR.Code, forbiddenRR.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/compliance/legal-holds/check?entity_type=user&entity_id=user-1", nil)
	adminHeaders(req, "admin-1", "idem-admin-cp-hold-check-1")
	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected hold check 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var response adminControlPlaneLegalHoldCheckResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode hold check response: %v", err)
	}
	if response.EntityID != "user-1" || response.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected hold check response: %+v", response)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.compliance.legal_hold.checked" {
		t.Fatalf("expected admin.compliance.legal_hold.checked action, got %+v", logs)
	}
}

func TestAdminControlPlaneRunComplianceScanRequiresReason(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/compliance/legal/compliance-scans", bytes.NewReader([]byte(`{
		"report_type":"manual"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-scan-1")
	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneSupportSearchForbiddenByScope(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/support/tickets/search?q=refund", nil)
	adminHeaders(req, "viewer-1", "idem-admin-cp-support-search-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneSupportAssignAndUpdateEmitsAuditLog(t *testing.T) {
	server := newTestServer()

	assignReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/support/tickets/ticket-1/assign", bytes.NewReader([]byte(`{
		"agent_id":"agent-senior",
		"reason":"triage reassignment"
	}`)))
	adminHeaders(assignReq, "admin-1", "idem-admin-cp-support-assign-1")
	assignRR := httptest.NewRecorder()
	server.mux.ServeHTTP(assignRR, assignReq)
	if assignRR.Code != http.StatusOK {
		t.Fatalf("expected assign 200, got %d body=%s", assignRR.Code, assignRR.Body.String())
	}
	var assignResp adminControlPlaneSupportTicketResponse
	if err := json.Unmarshal(assignRR.Body.Bytes(), &assignResp); err != nil {
		t.Fatalf("decode assign response: %v", err)
	}
	if assignResp.TicketID != "ticket-1" || assignResp.AssignedAgentID != "agent-senior" || assignResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected assign response: %+v", assignResp)
	}

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/admin/v1/support/tickets/ticket-1", bytes.NewReader([]byte(`{
		"status":"open",
		"sub_status":"escalated",
		"priority":"high",
		"reason":"escalated after fraud-review signal"
	}`)))
	adminHeaders(updateReq, "admin-1", "idem-admin-cp-support-update-1")
	updateRR := httptest.NewRecorder()
	server.mux.ServeHTTP(updateRR, updateReq)
	if updateRR.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d body=%s", updateRR.Code, updateRR.Body.String())
	}
	var updateResp adminControlPlaneSupportTicketResponse
	if err := json.Unmarshal(updateRR.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateResp.TicketID != "ticket-1" || updateResp.SubStatus != "escalated" || updateResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected update response: %+v", updateResp)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two support audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.support.escalation.created" {
		t.Fatalf("expected latest action admin.support.escalation.created, got %q", logs[0].Action)
	}
	if logs[1].Action != "admin.support.ticket.updated" {
		t.Fatalf("expected previous action admin.support.ticket.updated, got %q", logs[1].Action)
	}
}

func TestAdminControlPlaneCreatorWorkflowEditorSaveEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/creator-workflow/editor/campaigns/camp-1/save", bytes.NewReader([]byte(`{
		"editor_id":"editor-1",
		"reason":"manual curation request"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-creator-editor-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.creator_workflow.editor.campaign.saved" {
		t.Fatalf("expected admin.creator_workflow.editor.campaign.saved action, got %+v", logs)
	}
}

func TestAdminControlPlaneCreatorWorkflowClippingExportEmitsAuditLog(t *testing.T) {
	server := newTestServer()

	createProjectReq := httptest.NewRequest(http.MethodPost, "/api/clipping/v1/projects", bytes.NewReader([]byte(`{
		"title":"admin workflow seed",
		"description":"seed",
		"source_url":"https://cdn.whop.dev/source.mp4",
		"source_type":"url"
	}`)))
	createProjectReq.Header.Set("Authorization", "Bearer token")
	createProjectReq.Header.Set("X-Request-Id", "req-creator-clip-seed")
	createProjectReq.Header.Set("X-User-Id", "creator-1")
	createProjectReq.Header.Set("Idempotency-Key", "idem-creator-clip-seed")
	createProjectReq.Header.Set("Content-Type", "application/json")
	createProjectRR := httptest.NewRecorder()
	server.mux.ServeHTTP(createProjectRR, createProjectReq)
	if createProjectRR.Code != http.StatusCreated {
		t.Fatalf("expected project create 201, got %d body=%s", createProjectRR.Code, createProjectRR.Body.String())
	}

	var created struct {
		Data struct {
			Project struct {
				ProjectID string `json:"project_id"`
			} `json:"project"`
		} `json:"data"`
	}
	if err := json.Unmarshal(createProjectRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode project create response: %v", err)
	}
	if created.Data.Project.ProjectID == "" {
		t.Fatalf("expected project_id in create response")
	}

	exportReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/creator-workflow/clipping/projects/"+created.Data.Project.ProjectID+"/export", bytes.NewReader([]byte(`{
		"user_id":"creator-1",
		"format":"mp4",
		"resolution":"1080x1920",
		"fps":30,
		"bitrate":"8m",
		"reason":"manual policy export"
	}`)))
	adminHeaders(exportReq, "admin-1", "idem-admin-cp-creator-clip-export-1")
	exportRR := httptest.NewRecorder()
	server.mux.ServeHTTP(exportRR, exportReq)
	if exportRR.Code != http.StatusOK {
		t.Fatalf("expected clipping export 200, got %d body=%s", exportRR.Code, exportRR.Body.String())
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.creator_workflow.clipping.export.requested" {
		t.Fatalf("expected admin.creator_workflow.clipping.export.requested action, got %+v", logs)
	}
}

func TestAdminControlPlaneCreatorWorkflowAutoClippingDeployEmitsAuditLog(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/v1/creator-workflow/auto-clipping/models/deploy", bytes.NewReader([]byte(`{
		"model_name":"xgboost_ensemble",
		"version_tag":"v1.3.0",
		"model_artifact_key":"s3://models/xgb_v1.3.0.pkl",
		"canary_percentage":5,
		"description":"improved emotion weighting",
		"reason":"qa approved rollout"
	}`)))
	adminHeaders(req, "admin-1", "idem-admin-cp-creator-auto-deploy-1")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.creator_workflow.auto_clipping.model.deployed" {
		t.Fatalf("expected admin.creator_workflow.auto_clipping.model.deployed action, got %+v", logs)
	}
}

func TestAdminControlPlaneRotateIntegrationKeyIdempotencyReplayAndConflict(t *testing.T) {
	server := newTestServer()
	bodyOne := []byte(`{
		"key_id":"key-1",
		"reason":"routine rotation"
	}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/keys/rotate", bytes.NewReader(bodyOne))
	adminHeaders(req1, "admin-1", "idem-admin-cp-int-key-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first rotate 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}
	var first adminControlPlaneRotateIntegrationKeyResponse
	if err := json.Unmarshal(rr1.Body.Bytes(), &first); err != nil {
		t.Fatalf("decode first rotate response: %v", err)
	}
	if first.RotationID == "" || first.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected first rotate response: %+v", first)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/keys/rotate", bytes.NewReader(bodyOne))
	adminHeaders(req2, "admin-1", "idem-admin-cp-int-key-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay rotate 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}
	var replay adminControlPlaneRotateIntegrationKeyResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &replay); err != nil {
		t.Fatalf("decode replay rotate response: %v", err)
	}
	if replay.RotationID != first.RotationID {
		t.Fatalf("expected replay rotation id %q, got %q", first.RotationID, replay.RotationID)
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/keys/rotate", bytes.NewReader([]byte(`{
		"key_id":"key-2",
		"reason":"changed payload"
	}`)))
	adminHeaders(req3, "admin-1", "idem-admin-cp-int-key-1")
	rr3 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected conflict 409, got %d body=%s", rr3.Code, rr3.Body.String())
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) == 0 || logs[0].Action != "admin.integration.key.rotated" {
		t.Fatalf("expected admin.integration.key.rotated action, got %+v", logs)
	}
}

func TestAdminControlPlaneWebhookReplayDisableAndQueries(t *testing.T) {
	server := newTestServer()

	replayReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/webhooks/wh-1/replay", bytes.NewReader([]byte(`{
		"reason":"redrive failed event"
	}`)))
	adminHeaders(replayReq, "admin-1", "idem-admin-cp-wh-replay-1")
	replayRR := httptest.NewRecorder()
	server.mux.ServeHTTP(replayRR, replayReq)
	if replayRR.Code != http.StatusOK {
		t.Fatalf("expected replay 200, got %d body=%s", replayRR.Code, replayRR.Body.String())
	}
	var replayResp adminControlPlaneReplayWebhookResponse
	if err := json.Unmarshal(replayRR.Body.Bytes(), &replayResp); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if replayResp.DeliveryID == "" || replayResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected replay response: %+v", replayResp)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/webhooks/wh-1/disable", bytes.NewReader([]byte(`{
		"reason":"partner endpoint unstable"
	}`)))
	adminHeaders(disableReq, "admin-1", "idem-admin-cp-wh-disable-1")
	disableRR := httptest.NewRecorder()
	server.mux.ServeHTTP(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("expected disable 200, got %d body=%s", disableRR.Code, disableRR.Body.String())
	}
	var disableResp adminControlPlaneDisableWebhookResponse
	if err := json.Unmarshal(disableRR.Body.Bytes(), &disableResp); err != nil {
		t.Fatalf("decode disable response: %v", err)
	}
	if disableResp.WebhookID != "wh-1" || disableResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected disable response: %+v", disableResp)
	}

	deliveriesReq := httptest.NewRequest(http.MethodGet, "/api/admin/v1/integrations/webhooks/wh-1/deliveries?limit=10", nil)
	adminHeaders(deliveriesReq, "admin-1", "idem-admin-cp-wh-deliveries-1")
	deliveriesRR := httptest.NewRecorder()
	server.mux.ServeHTTP(deliveriesRR, deliveriesReq)
	if deliveriesRR.Code != http.StatusOK {
		t.Fatalf("expected deliveries 200, got %d body=%s", deliveriesRR.Code, deliveriesRR.Body.String())
	}
	var deliveriesResp adminControlPlaneWebhookDeliveriesResponse
	if err := json.Unmarshal(deliveriesRR.Body.Bytes(), &deliveriesResp); err != nil {
		t.Fatalf("decode deliveries response: %v", err)
	}
	if len(deliveriesResp.Deliveries) == 0 {
		t.Fatalf("expected at least one delivery in response")
	}

	analyticsReq := httptest.NewRequest(http.MethodGet, "/api/admin/v1/integrations/webhooks/wh-1/analytics", nil)
	adminHeaders(analyticsReq, "admin-1", "idem-admin-cp-wh-analytics-1")
	analyticsRR := httptest.NewRecorder()
	server.mux.ServeHTTP(analyticsRR, analyticsReq)
	if analyticsRR.Code != http.StatusOK {
		t.Fatalf("expected analytics 200, got %d body=%s", analyticsRR.Code, analyticsRR.Body.String())
	}
	var analyticsResp adminControlPlaneWebhookAnalyticsResponse
	if err := json.Unmarshal(analyticsRR.Body.Bytes(), &analyticsResp); err != nil {
		t.Fatalf("decode analytics response: %v", err)
	}
	if analyticsResp.TotalDeliveries <= 0 {
		t.Fatalf("expected positive delivery count in analytics")
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two webhook audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.webhook.disabled" {
		t.Fatalf("expected latest action admin.webhook.disabled, got %q", logs[0].Action)
	}
	if logs[1].Action != "admin.webhook.replayed" {
		t.Fatalf("expected previous action admin.webhook.replayed, got %q", logs[1].Action)
	}
}

func TestAdminControlPlaneWebhookAnalyticsForbiddenByScope(t *testing.T) {
	server := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/v1/integrations/webhooks/wh-1/analytics", nil)
	adminHeaders(req, "viewer-1", "idem-admin-cp-wh-analytics-forbidden")

	rr := httptest.NewRecorder()
	server.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminControlPlaneMigrationPlanRunAndListEmitAuditLog(t *testing.T) {
	server := newTestServer()

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/platform-ops/migrations/plans", bytes.NewReader([]byte(`{
		"service_name":"M84-data-migration-service",
		"environment":"staging",
		"version":"2026.03.07",
		"plan":{"op":"backfill"},
		"dry_run":true,
		"risk_level":"medium",
		"reason":"approved dry-run"
	}`)))
	adminHeaders(createReq, "admin-1", "idem-admin-cp-mig-plan-1")
	createRR := httptest.NewRecorder()
	server.mux.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusOK {
		t.Fatalf("expected migration plan create 200, got %d body=%s", createRR.Code, createRR.Body.String())
	}
	var createResp adminControlPlaneMigrationPlanResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode migration plan response: %v", err)
	}
	if createResp.PlanID == "" || createResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected migration plan response: %+v", createResp)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/v1/platform-ops/migrations/plans", nil)
	adminHeaders(listReq, "admin-1", "idem-admin-cp-mig-list-1")
	listRR := httptest.NewRecorder()
	server.mux.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected migration plan list 200, got %d body=%s", listRR.Code, listRR.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/admin/v1/platform-ops/migrations/runs", bytes.NewReader([]byte(`{
		"plan_id":"`+createResp.PlanID+`",
		"reason":"execute approved run"
	}`)))
	adminHeaders(runReq, "admin-1", "idem-admin-cp-mig-run-1")
	runRR := httptest.NewRecorder()
	server.mux.ServeHTTP(runRR, runReq)
	if runRR.Code != http.StatusOK {
		t.Fatalf("expected migration run start 200, got %d body=%s", runRR.Code, runRR.Body.String())
	}
	var runResp adminControlPlaneMigrationRunResponse
	if err := json.Unmarshal(runRR.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode migration run response: %v", err)
	}
	if runResp.RunID == "" || runResp.ControlPlaneAuditID == "" {
		t.Fatalf("unexpected migration run response: %+v", runResp)
	}

	logs, err := server.adminDashboard.Store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list recent audit logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least two migration audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.backfill.completed" {
		t.Fatalf("expected latest action admin.backfill.completed, got %q", logs[0].Action)
	}
	if logs[1].Action != "admin.backfill.started" {
		t.Fatalf("expected previous action admin.backfill.started, got %q", logs[1].Action)
	}
}
