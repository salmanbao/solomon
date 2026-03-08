package application

import (
	"context"
	"testing"

	"solomon/contexts/internal-ops/admin-dashboard-service/adapters/memory"
	domainerrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	"solomon/contexts/internal-ops/admin-dashboard-service/ports"
)

type fakeAuthorizationClient struct {
	calls  int
	result ports.RoleGrantResult
	err    error
}

func (f *fakeAuthorizationClient) GrantRole(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.RoleGrantResult, error) {
	f.calls++
	if f.err != nil {
		return ports.RoleGrantResult{}, f.err
	}
	return f.result, nil
}

type fakeModerationClient struct {
	approveCalls int
	rejectCalls  int
	result       ports.ModerationDecisionResult
	err          error
}

func (f *fakeModerationClient) ApproveSubmission(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.ModerationDecisionResult, error) {
	f.approveCalls++
	if f.err != nil {
		return ports.ModerationDecisionResult{}, f.err
	}
	out := f.result
	out.Action = "approved"
	out.QueueStatus = "approved"
	return out, nil
}

type fakeAbusePreventionClient struct {
	calls  int
	result ports.AbuseLockoutResult
	err    error
}

func (f *fakeAbusePreventionClient) ReleaseLockout(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.AbuseLockoutResult, error) {
	f.calls++
	if f.err != nil {
		return ports.AbuseLockoutResult{}, f.err
	}
	return f.result, nil
}

func (f *fakeModerationClient) RejectSubmission(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.ModerationDecisionResult, error) {
	f.rejectCalls++
	if f.err != nil {
		return ports.ModerationDecisionResult{}, f.err
	}
	out := f.result
	out.Action = "rejected"
	out.QueueStatus = "rejected"
	return out, nil
}

type fakeFinanceClient struct {
	calls  int
	result ports.FinanceRefundResult
	err    error
}

func (f *fakeFinanceClient) CreateRefund(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ float64,
	_ string,
	_ string,
) (ports.FinanceRefundResult, error) {
	f.calls++
	if f.err != nil {
		return ports.FinanceRefundResult{}, f.err
	}
	return f.result, nil
}

type fakePayoutClient struct {
	calls  int
	result ports.PayoutRetryResult
	err    error
}

func (f *fakePayoutClient) RetryFailedPayout(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.PayoutRetryResult, error) {
	f.calls++
	if f.err != nil {
		return ports.PayoutRetryResult{}, f.err
	}
	return f.result, nil
}

type fakeResolutionClient struct {
	calls  int
	result ports.DisputeResolutionResult
	err    error
}

func (f *fakeResolutionClient) ResolveDispute(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ float64,
	_ string,
) (ports.DisputeResolutionResult, error) {
	f.calls++
	if f.err != nil {
		return ports.DisputeResolutionResult{}, f.err
	}
	return f.result, nil
}

type fakeEditorWorkflowClient struct {
	calls  int
	result ports.EditorCampaignSaveResult
	err    error
}

func (f *fakeEditorWorkflowClient) SaveCampaign(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
) (ports.EditorCampaignSaveResult, error) {
	f.calls++
	if f.err != nil {
		return ports.EditorCampaignSaveResult{}, f.err
	}
	return f.result, nil
}

type fakeClippingWorkflowClient struct {
	calls  int
	result ports.ClippingExportResult
	err    error
}

func (f *fakeClippingWorkflowClient) RequestExport(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ string,
	_ int,
	_ string,
	_ string,
	_ string,
) (ports.ClippingExportResult, error) {
	f.calls++
	if f.err != nil {
		return ports.ClippingExportResult{}, f.err
	}
	return f.result, nil
}

type fakeAutoClippingClient struct {
	calls              int
	lastIdempotencyKey string
	result             ports.AutoClippingModelDeployResult
	err                error
}

func (f *fakeAutoClippingClient) DeployModel(
	_ context.Context,
	_ string,
	_ ports.AutoClippingModelDeployInput,
	idempotencyKey string,
) (ports.AutoClippingModelDeployResult, error) {
	f.calls++
	f.lastIdempotencyKey = idempotencyKey
	if f.err != nil {
		return ports.AutoClippingModelDeployResult{}, f.err
	}
	return f.result, nil
}

type fakeDeveloperPortalClient struct {
	calls  int
	result ports.IntegrationKeyRotationResult
	err    error
}

func (f *fakeDeveloperPortalClient) RotateAPIKey(
	_ context.Context,
	_ string,
	_ string,
	_ string,
) (ports.IntegrationKeyRotationResult, error) {
	f.calls++
	if f.err != nil {
		return ports.IntegrationKeyRotationResult{}, f.err
	}
	return f.result, nil
}

type fakeWebhookManagerClient struct {
	replayCalls   int
	disableCalls  int
	replayResult  ports.WebhookReplayResult
	disableResult ports.WebhookEndpointResult
	err           error
}

func (f *fakeWebhookManagerClient) ReplayWebhook(
	_ context.Context,
	_ string,
	_ string,
	_ string,
) (ports.WebhookReplayResult, error) {
	f.replayCalls++
	if f.err != nil {
		return ports.WebhookReplayResult{}, f.err
	}
	return f.replayResult, nil
}

func (f *fakeWebhookManagerClient) DisableWebhook(
	_ context.Context,
	_ string,
	_ string,
	_ string,
) (ports.WebhookEndpointResult, error) {
	f.disableCalls++
	if f.err != nil {
		return ports.WebhookEndpointResult{}, f.err
	}
	return f.disableResult, nil
}

func (f *fakeWebhookManagerClient) ListDeliveries(
	_ context.Context,
	_ string,
	_ string,
	_ int,
) ([]ports.WebhookDeliveryResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []ports.WebhookDeliveryResult{}, nil
}

func (f *fakeWebhookManagerClient) GetAnalytics(
	_ context.Context,
	_ string,
	_ string,
) (ports.WebhookAnalyticsResult, error) {
	if f.err != nil {
		return ports.WebhookAnalyticsResult{}, f.err
	}
	return ports.WebhookAnalyticsResult{ByEventType: map[string]ports.WebhookAnalyticsMetrics{}}, nil
}

type fakeDataMigrationClient struct {
	createPlanCalls int
	createRunCalls  int
	planResult      ports.MigrationPlanResult
	runResult       ports.MigrationRunResult
	err             error
}

func (f *fakeDataMigrationClient) CreatePlan(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
	_ map[string]interface{},
	_ bool,
	_ string,
	_ string,
) (ports.MigrationPlanResult, error) {
	f.createPlanCalls++
	if f.err != nil {
		return ports.MigrationPlanResult{}, f.err
	}
	return f.planResult, nil
}

func (f *fakeDataMigrationClient) ListPlans(
	_ context.Context,
	_ string,
) ([]ports.MigrationPlanResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []ports.MigrationPlanResult{f.planResult}, nil
}

func (f *fakeDataMigrationClient) CreateRun(
	_ context.Context,
	_ string,
	_ string,
	_ string,
) (ports.MigrationRunResult, error) {
	f.createRunCalls++
	if f.err != nil {
		return ports.MigrationRunResult{}, f.err
	}
	return f.runResult, nil
}

func TestGrantIdentityRoleIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	authz := &fakeAuthorizationClient{
		result: ports.RoleGrantResult{
			AssignmentID: "asg_1",
			UserID:       "user_1",
			RoleID:       "trust_safety_moderator",
			AuditLogID:   "authz_audit_1",
		},
	}
	svc := Service{
		Repo:                store,
		Idempotency:         store,
		AuthorizationClient: authz,
		Clock:               store,
	}

	input := GrantIdentityRoleInput{
		ActorID:       "admin_1",
		UserID:        "user_1",
		RoleID:        "trust_safety_moderator",
		Reason:        "assign moderation operator role",
		SourceIP:      "127.0.0.1",
		CorrelationID: "corr-1",
	}
	got1, err := svc.GrantIdentityRole(context.Background(), "idem-grant-1", input)
	if err != nil {
		t.Fatalf("first grant failed: %v", err)
	}
	got2, err := svc.GrantIdentityRole(context.Background(), "idem-grant-1", input)
	if err != nil {
		t.Fatalf("replay grant failed: %v", err)
	}

	if authz.calls != 1 {
		t.Fatalf("expected one owner call, got %d", authz.calls)
	}
	if got1.AssignmentID != "asg_1" || got2.AssignmentID != "asg_1" {
		t.Fatalf("unexpected assignment IDs: first=%q second=%q", got1.AssignmentID, got2.AssignmentID)
	}
	if got1.ControlPlaneAuditID == "" || got2.ControlPlaneAuditID == "" {
		t.Fatalf("control-plane audit id missing")
	}

	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(logs))
	}
	if logs[0].Action != "admin.identity.role.granted" {
		t.Fatalf("unexpected audit action %q", logs[0].Action)
	}
	if logs[0].TargetID != "user_1" {
		t.Fatalf("unexpected audit target %q", logs[0].TargetID)
	}
}

func TestGrantIdentityRoleIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	authz := &fakeAuthorizationClient{
		result: ports.RoleGrantResult{
			AssignmentID: "asg_1",
			UserID:       "user_1",
			RoleID:       "trust_safety_moderator",
			AuditLogID:   "authz_audit_1",
		},
	}
	svc := Service{
		Repo:                store,
		Idempotency:         store,
		AuthorizationClient: authz,
		Clock:               store,
	}

	_, err := svc.GrantIdentityRole(context.Background(), "idem-grant-cf", GrantIdentityRoleInput{
		ActorID: "admin_1",
		UserID:  "user_1",
		RoleID:  "trust_safety_moderator",
		Reason:  "initial assignment",
	})
	if err != nil {
		t.Fatalf("seed grant failed: %v", err)
	}
	_, err = svc.GrantIdentityRole(context.Background(), "idem-grant-cf", GrantIdentityRoleInput{
		ActorID: "admin_1",
		UserID:  "user_1",
		RoleID:  "ops_admin",
		Reason:  "changed role with same key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestModerateSubmissionRejectRequiresReason(t *testing.T) {
	store := memory.NewStore()
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		ModerationClient: &fakeModerationClient{},
		Clock:            store,
	}

	_, err := svc.ModerateSubmission(context.Background(), "idem-mod-1", ModerateSubmissionInput{
		ActorID:      "admin_1",
		SubmissionID: "sub_1",
		CampaignID:   "cmp_1",
		Action:       "reject",
	})
	if err == nil {
		t.Fatalf("expected error for empty reject reason")
	}
	if err != domainerrors.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestModerateSubmissionApproveIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	mod := &fakeModerationClient{
		result: ports.ModerationDecisionResult{
			DecisionID:   "dec_1",
			SubmissionID: "sub_1",
			CampaignID:   "cmp_1",
			ModeratorID:  "admin_1",
		},
	}
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		ModerationClient: mod,
		Clock:            store,
	}

	input := ModerateSubmissionInput{
		ActorID:       "admin_1",
		SubmissionID:  "sub_1",
		CampaignID:    "cmp_1",
		Action:        "approve",
		Reason:        "policy compliant",
		SourceIP:      "127.0.0.1",
		CorrelationID: "corr-2",
	}
	got1, err := svc.ModerateSubmission(context.Background(), "idem-mod-2", input)
	if err != nil {
		t.Fatalf("first moderation failed: %v", err)
	}
	got2, err := svc.ModerateSubmission(context.Background(), "idem-mod-2", input)
	if err != nil {
		t.Fatalf("replay moderation failed: %v", err)
	}

	if mod.approveCalls != 1 || mod.rejectCalls != 0 {
		t.Fatalf("unexpected owner call counts approve=%d reject=%d", mod.approveCalls, mod.rejectCalls)
	}
	if got1.Action != "approved" || got2.Action != "approved" {
		t.Fatalf("unexpected action values first=%q second=%q", got1.Action, got2.Action)
	}

	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(logs))
	}
	if logs[0].Action != "admin.submission.moderated" {
		t.Fatalf("unexpected audit action %q", logs[0].Action)
	}
	if logs[0].TargetID != "sub_1" {
		t.Fatalf("unexpected audit target %q", logs[0].TargetID)
	}
}

func TestModerateSubmissionIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	mod := &fakeModerationClient{
		result: ports.ModerationDecisionResult{
			DecisionID:   "dec_1",
			SubmissionID: "sub_1",
			CampaignID:   "cmp_1",
			ModeratorID:  "admin_1",
		},
	}
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		ModerationClient: mod,
		Clock:            store,
	}

	_, err := svc.ModerateSubmission(context.Background(), "idem-mod-cf", ModerateSubmissionInput{
		ActorID:      "admin_1",
		SubmissionID: "sub_1",
		CampaignID:   "cmp_1",
		Action:       "approve",
		Reason:       "policy compliant",
	})
	if err != nil {
		t.Fatalf("seed moderation failed: %v", err)
	}
	_, err = svc.ModerateSubmission(context.Background(), "idem-mod-cf", ModerateSubmissionInput{
		ActorID:      "admin_1",
		SubmissionID: "sub_1",
		CampaignID:   "cmp_1",
		Action:       "reject",
		Reason:       "changed decision same key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestReleaseAbuseLockoutIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	abuse := &fakeAbusePreventionClient{
		result: ports.AbuseLockoutResult{
			ThreatID:        "threat_user-200",
			UserID:          "user-200",
			Status:          "released",
			OwnerAuditLogID: "abuse_audit_1",
		},
	}
	svc := Service{
		Repo:                  store,
		Idempotency:           store,
		AbusePreventionClient: abuse,
		Clock:                 store,
	}

	input := ReleaseAbuseLockoutInput{
		ActorID:       "admin-1",
		UserID:        "user-200",
		Reason:        "false positive lockout",
		SourceIP:      "127.0.0.1",
		CorrelationID: "corr-abuse-cp-1",
	}
	first, err := svc.ReleaseAbuseLockout(context.Background(), "idem-abuse-cp-1", input)
	if err != nil {
		t.Fatalf("first release failed: %v", err)
	}
	replay, err := svc.ReleaseAbuseLockout(context.Background(), "idem-abuse-cp-1", input)
	if err != nil {
		t.Fatalf("replay release failed: %v", err)
	}

	if abuse.calls != 1 {
		t.Fatalf("expected one owner call, got %d", abuse.calls)
	}
	if replay.ThreatID != first.ThreatID {
		t.Fatalf("expected replay threat id %q, got %q", first.ThreatID, replay.ThreatID)
	}
	if first.ControlPlaneAuditID == "" || replay.ControlPlaneAuditID == "" {
		t.Fatalf("expected control-plane audit ids")
	}

	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one audit log, got %d", len(logs))
	}
	if logs[0].Action != "admin.abuse.lockout.released" {
		t.Fatalf("unexpected audit action %q", logs[0].Action)
	}
}

func TestReleaseAbuseLockoutIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	abuse := &fakeAbusePreventionClient{
		result: ports.AbuseLockoutResult{
			ThreatID:        "threat_user-200",
			UserID:          "user-200",
			Status:          "released",
			OwnerAuditLogID: "abuse_audit_1",
		},
	}
	svc := Service{
		Repo:                  store,
		Idempotency:           store,
		AbusePreventionClient: abuse,
		Clock:                 store,
	}

	_, err := svc.ReleaseAbuseLockout(context.Background(), "idem-abuse-cp-cf", ReleaseAbuseLockoutInput{
		ActorID: "admin-1",
		UserID:  "user-200",
		Reason:  "seed release",
	})
	if err != nil {
		t.Fatalf("seed release failed: %v", err)
	}
	_, err = svc.ReleaseAbuseLockout(context.Background(), "idem-abuse-cp-cf", ReleaseAbuseLockoutInput{
		ActorID: "admin-1",
		UserID:  "user-200",
		Reason:  "different payload same key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestCreateFinanceRefundIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	finance := &fakeFinanceClient{
		result: ports.FinanceRefundResult{
			RefundID:      "refund_1",
			TransactionID: "txn_1",
			UserID:        "user_1",
			Amount:        25,
			Reason:        "duplicate charge",
			CreatedAt:     store.Now(),
		},
	}
	svc := Service{
		Repo:          store,
		Idempotency:   store,
		FinanceClient: finance,
		Clock:         store,
	}

	input := CreateFinanceRefundInput{
		ActorID:       "admin_1",
		TransactionID: "txn_1",
		UserID:        "user_1",
		Amount:        25,
		Reason:        "duplicate charge",
	}
	first, err := svc.CreateFinanceRefund(context.Background(), "idem-fin-1", input)
	if err != nil {
		t.Fatalf("first refund failed: %v", err)
	}
	replay, err := svc.CreateFinanceRefund(context.Background(), "idem-fin-1", input)
	if err != nil {
		t.Fatalf("replay refund failed: %v", err)
	}

	if finance.calls != 1 {
		t.Fatalf("expected one owner call, got %d", finance.calls)
	}
	if replay.RefundID != first.RefundID {
		t.Fatalf("expected replay refund id %q, got %q", first.RefundID, replay.RefundID)
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.finance.refund.created" {
		t.Fatalf("expected finance refund audit log, got %+v", logs)
	}
}

func TestRetryPayoutIdempotencyConflict(t *testing.T) {
	store := memory.NewStore()
	payout := &fakePayoutClient{
		result: ports.PayoutRetryResult{
			PayoutID:    "pay_1",
			UserID:      "user_1",
			Status:      "paid",
			ProcessedAt: store.Now(),
		},
	}
	svc := Service{
		Repo:         store,
		Idempotency:  store,
		PayoutClient: payout,
		Clock:        store,
	}

	_, err := svc.RetryPayout(context.Background(), "idem-pay-1", RetryPayoutInput{
		ActorID:  "admin_1",
		PayoutID: "pay_1",
		Reason:   "provider recovered",
	})
	if err != nil {
		t.Fatalf("seed payout retry failed: %v", err)
	}
	_, err = svc.RetryPayout(context.Background(), "idem-pay-1", RetryPayoutInput{
		ActorID:  "admin_1",
		PayoutID: "pay_1",
		Reason:   "changed reason",
	})
	if err != domainerrors.ErrIdempotencyConflict {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestResolveDisputeReopenEmitsStatusAudit(t *testing.T) {
	store := memory.NewStore()
	resolution := &fakeResolutionClient{
		result: ports.DisputeResolutionResult{
			DisputeID:   "dispute_1",
			Status:      "reopened",
			ProcessedAt: store.Now(),
		},
	}
	svc := Service{
		Repo:             store,
		Idempotency:      store,
		ResolutionClient: resolution,
		Clock:            store,
	}

	_, err := svc.ResolveDispute(context.Background(), "idem-dsp-1", ResolveDisputeInput{
		ActorID:   "admin_1",
		DisputeID: "dispute_1",
		Action:    "reopen",
		Reason:    "new evidence provided",
	})
	if err != nil {
		t.Fatalf("reopen dispute failed: %v", err)
	}

	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.dispute.status.changed" {
		t.Fatalf("expected status changed audit log, got %+v", logs)
	}
}

func TestSaveEditorCampaignIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	editor := &fakeEditorWorkflowClient{
		result: ports.EditorCampaignSaveResult{
			CampaignID: "camp-1",
			Saved:      true,
		},
	}
	svc := Service{
		Repo:                 store,
		Idempotency:          store,
		EditorWorkflowClient: editor,
		Clock:                store,
	}

	input := SaveEditorCampaignInput{
		ActorID:    "admin-1",
		EditorID:   "editor-1",
		CampaignID: "camp-1",
		Reason:     "campaign curation",
	}
	first, err := svc.SaveEditorCampaign(context.Background(), "idem-editor-save-1", input)
	if err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	replay, err := svc.SaveEditorCampaign(context.Background(), "idem-editor-save-1", input)
	if err != nil {
		t.Fatalf("replay save failed: %v", err)
	}
	if editor.calls != 1 {
		t.Fatalf("expected one owner call, got %d", editor.calls)
	}
	if first.CampaignID != replay.CampaignID || !replay.Saved {
		t.Fatalf("unexpected replay result: first=%+v replay=%+v", first, replay)
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.creator_workflow.editor.campaign.saved" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestRequestClippingExportIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	clipping := &fakeClippingWorkflowClient{
		result: ports.ClippingExportResult{
			ExportID:    "exp-1",
			ProjectID:   "proj-1",
			Status:      "queued",
			CreatedAt:   store.Now(),
			CompletedAt: nil,
		},
	}
	svc := Service{
		Repo:                   store,
		Idempotency:            store,
		ClippingWorkflowClient: clipping,
		Clock:                  store,
	}

	input := RequestClippingExportInput{
		ActorID:    "admin-1",
		UserID:     "creator-1",
		ProjectID:  "proj-1",
		Format:     "mp4",
		Resolution: "1080x1920",
		FPS:        30,
		Bitrate:    "8m",
		Reason:     "policy review export",
	}
	first, err := svc.RequestClippingExport(context.Background(), "idem-clip-export-1", input)
	if err != nil {
		t.Fatalf("first export failed: %v", err)
	}
	replay, err := svc.RequestClippingExport(context.Background(), "idem-clip-export-1", input)
	if err != nil {
		t.Fatalf("replay export failed: %v", err)
	}
	if clipping.calls != 1 {
		t.Fatalf("expected one owner call, got %d", clipping.calls)
	}
	if first.ExportID != replay.ExportID {
		t.Fatalf("unexpected replay export ids first=%q replay=%q", first.ExportID, replay.ExportID)
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.creator_workflow.clipping.export.requested" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestDeployAutoClippingModelIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	auto := &fakeAutoClippingClient{
		result: ports.AutoClippingModelDeployResult{
			ModelVersionID:   "model-1",
			DeploymentStatus: "canary_5pct",
			DeployedAt:       store.Now(),
			Message:          "deployed",
		},
	}
	svc := Service{
		Repo:               store,
		Idempotency:        store,
		AutoClippingClient: auto,
		Clock:              store,
	}

	input := DeployAutoClippingModelInput{
		ActorID:          "admin-1",
		ModelName:        "xgboost_ensemble",
		VersionTag:       "v1.0.1",
		ModelArtifactKey: "s3://models/xgb_v1.0.1.pkl",
		CanaryPercentage: 5,
		Description:      "small upgrade",
		Reason:           "quality uplift",
	}
	first, err := svc.DeployAutoClippingModel(context.Background(), "idem-auto-deploy-1", input)
	if err != nil {
		t.Fatalf("first deploy failed: %v", err)
	}
	replay, err := svc.DeployAutoClippingModel(context.Background(), "idem-auto-deploy-1", input)
	if err != nil {
		t.Fatalf("replay deploy failed: %v", err)
	}
	if auto.calls != 1 {
		t.Fatalf("expected one owner call, got %d", auto.calls)
	}
	if auto.lastIdempotencyKey != "m86:creator_workflow_auto_clipping_deploy:idem-auto-deploy-1" {
		t.Fatalf("expected owner idempotency key propagation, got %q", auto.lastIdempotencyKey)
	}
	if first.ModelVersionID != replay.ModelVersionID {
		t.Fatalf("unexpected model version replay first=%q replay=%q", first.ModelVersionID, replay.ModelVersionID)
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.creator_workflow.auto_clipping.model.deployed" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestRotateIntegrationKeyIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	developerPortal := &fakeDeveloperPortalClient{
		result: ports.IntegrationKeyRotationResult{
			RotationID:  "rot-1",
			DeveloperID: "dev-1",
			OldKeyID:    "key-old",
			NewKeyID:    "key-new",
			CreatedAt:   store.Now(),
		},
	}
	svc := Service{
		Repo:                  store,
		Idempotency:           store,
		DeveloperPortalClient: developerPortal,
		Clock:                 store,
	}

	input := RotateIntegrationKeyInput{
		ActorID: "admin-1",
		KeyID:   "key-old",
		Reason:  "key hygiene rotation",
	}
	first, err := svc.RotateIntegrationKey(context.Background(), "idem-int-key-1", input)
	if err != nil {
		t.Fatalf("first rotate failed: %v", err)
	}
	replay, err := svc.RotateIntegrationKey(context.Background(), "idem-int-key-1", input)
	if err != nil {
		t.Fatalf("replay rotate failed: %v", err)
	}
	if developerPortal.calls != 1 {
		t.Fatalf("expected one owner call, got %d", developerPortal.calls)
	}
	if first.RotationID != replay.RotationID {
		t.Fatalf("expected idempotent rotation replay")
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.integration.key.rotated" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestReplayWebhookIdempotentReplay(t *testing.T) {
	store := memory.NewStore()
	webhooks := &fakeWebhookManagerClient{
		replayResult: ports.WebhookReplayResult{
			DeliveryID: "del-1",
			WebhookID:  "wh-1",
			Status:     "success",
			HTTPStatus: 200,
			LatencyMS:  22,
			Timestamp:  store.Now(),
		},
	}
	svc := Service{
		Repo:                 store,
		Idempotency:          store,
		WebhookManagerClient: webhooks,
		Clock:                store,
	}

	input := ReplayWebhookInput{
		ActorID:   "admin-1",
		WebhookID: "wh-1",
		Reason:    "retry failed delivery",
	}
	first, err := svc.ReplayWebhook(context.Background(), "idem-wh-replay-1", input)
	if err != nil {
		t.Fatalf("first replay failed: %v", err)
	}
	replay, err := svc.ReplayWebhook(context.Background(), "idem-wh-replay-1", input)
	if err != nil {
		t.Fatalf("replay replay failed: %v", err)
	}
	if webhooks.replayCalls != 1 {
		t.Fatalf("expected one owner replay call, got %d", webhooks.replayCalls)
	}
	if first.DeliveryID != replay.DeliveryID {
		t.Fatalf("expected idempotent replay delivery id")
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) != 1 || logs[0].Action != "admin.webhook.replayed" {
		t.Fatalf("unexpected logs: %+v", logs)
	}
}

func TestCreateMigrationPlanAndRunAudits(t *testing.T) {
	store := memory.NewStore()
	migrations := &fakeDataMigrationClient{
		planResult: ports.MigrationPlanResult{
			PlanID:      "plan-1",
			ServiceName: "M84-data-migration-service",
			Environment: "staging",
			Version:     "2026.03.07",
			Plan:        map[string]interface{}{"op": "backfill"},
			Status:      "validated",
			DryRun:      true,
			RiskLevel:   "medium",
			CreatedBy:   "admin-1",
			CreatedAt:   store.Now(),
			UpdatedAt:   store.Now(),
		},
		runResult: ports.MigrationRunResult{
			RunID:             "run-1",
			PlanID:            "plan-1",
			Status:            "completed",
			OperatorID:        "admin-1",
			SnapshotCreated:   true,
			RollbackAvailable: true,
			ValidationStatus:  "passed",
			BackfillJobID:     "bf-1",
			StartedAt:         store.Now(),
			CompletedAt:       store.Now(),
		},
	}
	svc := Service{
		Repo:                store,
		Idempotency:         store,
		DataMigrationClient: migrations,
		Clock:               store,
	}

	_, err := svc.CreateMigrationPlan(context.Background(), "idem-mig-plan-1", CreateMigrationPlanInput{
		ActorID:     "admin-1",
		ServiceName: "M84-data-migration-service",
		Environment: "staging",
		Version:     "2026.03.07",
		Plan:        map[string]interface{}{"op": "backfill"},
		DryRun:      true,
		RiskLevel:   "medium",
		Reason:      "approved dry-run",
	})
	if err != nil {
		t.Fatalf("create migration plan failed: %v", err)
	}
	_, err = svc.StartMigrationRun(context.Background(), "idem-mig-run-1", StartMigrationRunInput{
		ActorID: "admin-1",
		PlanID:  "plan-1",
		Reason:  "execute approved run",
	})
	if err != nil {
		t.Fatalf("start migration run failed: %v", err)
	}
	if migrations.createPlanCalls != 1 || migrations.createRunCalls != 1 {
		t.Fatalf("unexpected owner call counts plan=%d run=%d", migrations.createPlanCalls, migrations.createRunCalls)
	}
	logs, err := store.ListRecentAuditLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", len(logs))
	}
	if logs[0].Action != "admin.backfill.completed" || logs[1].Action != "admin.backfill.started" {
		t.Fatalf("unexpected backfill audit actions: first=%q second=%q", logs[0].Action, logs[1].Action)
	}
}
