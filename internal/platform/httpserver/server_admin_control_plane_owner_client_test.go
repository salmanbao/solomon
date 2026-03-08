package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	distributionservice "solomon/contexts/campaign-editorial/distribution-service"
	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	authorization "solomon/contexts/identity-access/authorization-service"
	admindashboarderrors "solomon/contexts/internal-ops/admin-dashboard-service/domain/errors"
	admindashboardports "solomon/contexts/internal-ops/admin-dashboard-service/ports"
	admindashboardhttp "solomon/contexts/internal-ops/admin-dashboard-service/transport/http"
)

func TestNewServerFailsFastWhenProductionOwnerBaseURLsMissing(t *testing.T) {
	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, "")
	t.Setenv(adminM14BaseURLEnv, "http://m14.example")
	t.Setenv(adminM44BaseURLEnv, "http://m44.example")
	t.Setenv(adminM05BaseURLEnv, "http://m05.example")
	t.Setenv(adminM41BaseURLEnv, "http://m41.example")
	t.Setenv(adminM89BaseURLEnv, "http://m89.example")
	t.Setenv(adminM50BaseURLEnv, "http://m50.example")
	t.Setenv(adminM51BaseURLEnv, "http://m51.example")
	t.Setenv(adminM68BaseURLEnv, "http://m68.example")
	t.Setenv(adminM69BaseURLEnv, "http://m69.example")
	t.Setenv(adminM73BaseURLEnv, "http://m73.example")
	t.Setenv(adminM25BaseURLEnv, "http://m25.example")
	t.Setenv(adminM70BaseURLEnv, "http://m70.example")
	t.Setenv(adminM71BaseURLEnv, "http://m71.example")
	t.Setenv(adminM72BaseURLEnv, "http://m72.example")
	t.Setenv(adminM84BaseURLEnv, "http://m84.example")

	_, err := newServerForAdminControlPlaneTest()
	if err == nil {
		t.Fatalf("expected startup config error when %s is missing", adminM39BaseURLEnv)
	}
	if !strings.Contains(err.Error(), adminM39BaseURLEnv) {
		t.Fatalf("expected error to mention missing %s, got %v", adminM39BaseURLEnv, err)
	}
}

func TestNewServerFailsFastWhenProductionComplianceOwnerBaseURLsMissing(t *testing.T) {
	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, "http://m39.example")
	t.Setenv(adminM14BaseURLEnv, "http://m14.example")
	t.Setenv(adminM44BaseURLEnv, "http://m44.example")
	t.Setenv(adminM05BaseURLEnv, "http://m05.example")
	t.Setenv(adminM41BaseURLEnv, "http://m41.example")
	t.Setenv(adminM89BaseURLEnv, "http://m89.example")
	t.Setenv(adminM50BaseURLEnv, "")
	t.Setenv(adminM51BaseURLEnv, "http://m51.example")
	t.Setenv(adminM68BaseURLEnv, "http://m68.example")
	t.Setenv(adminM69BaseURLEnv, "http://m69.example")
	t.Setenv(adminM73BaseURLEnv, "http://m73.example")
	t.Setenv(adminM25BaseURLEnv, "http://m25.example")
	t.Setenv(adminM70BaseURLEnv, "http://m70.example")
	t.Setenv(adminM71BaseURLEnv, "http://m71.example")
	t.Setenv(adminM72BaseURLEnv, "http://m72.example")
	t.Setenv(adminM84BaseURLEnv, "http://m84.example")

	_, err := newServerForAdminControlPlaneTest()
	if err == nil {
		t.Fatalf("expected startup config error when %s is missing", adminM50BaseURLEnv)
	}
	if !strings.Contains(err.Error(), adminM50BaseURLEnv) {
		t.Fatalf("expected error to mention missing %s, got %v", adminM50BaseURLEnv, err)
	}
}

func TestNewServerFailsFastWhenProductionIntegrationOwnerBaseURLsMissing(t *testing.T) {
	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, "http://m39.example")
	t.Setenv(adminM14BaseURLEnv, "http://m14.example")
	t.Setenv(adminM44BaseURLEnv, "http://m44.example")
	t.Setenv(adminM05BaseURLEnv, "http://m05.example")
	t.Setenv(adminM41BaseURLEnv, "http://m41.example")
	t.Setenv(adminM89BaseURLEnv, "http://m89.example")
	t.Setenv(adminM50BaseURLEnv, "http://m50.example")
	t.Setenv(adminM51BaseURLEnv, "http://m51.example")
	t.Setenv(adminM68BaseURLEnv, "http://m68.example")
	t.Setenv(adminM69BaseURLEnv, "http://m69.example")
	t.Setenv(adminM73BaseURLEnv, "http://m73.example")
	t.Setenv(adminM25BaseURLEnv, "http://m25.example")
	t.Setenv(adminM70BaseURLEnv, "")
	t.Setenv(adminM71BaseURLEnv, "http://m71.example")
	t.Setenv(adminM72BaseURLEnv, "http://m72.example")
	t.Setenv(adminM84BaseURLEnv, "http://m84.example")

	_, err := newServerForAdminControlPlaneTest()
	if err == nil {
		t.Fatalf("expected startup config error when %s is missing", adminM70BaseURLEnv)
	}
	if !strings.Contains(err.Error(), adminM70BaseURLEnv) {
		t.Fatalf("expected error to mention missing %s, got %v", adminM70BaseURLEnv, err)
	}
}

func TestNewServerAllowsExplicitNonProductionFallback(t *testing.T) {
	t.Setenv(adminRuntimeModeEnv, "test")
	t.Setenv(adminM39BaseURLEnv, "")
	t.Setenv(adminM14BaseURLEnv, "")
	t.Setenv(adminM44BaseURLEnv, "")
	t.Setenv(adminM05BaseURLEnv, "")
	t.Setenv(adminM41BaseURLEnv, "")
	t.Setenv(adminM89BaseURLEnv, "")
	t.Setenv(adminM50BaseURLEnv, "")
	t.Setenv(adminM51BaseURLEnv, "")
	t.Setenv(adminM68BaseURLEnv, "")
	t.Setenv(adminM69BaseURLEnv, "")
	t.Setenv(adminM73BaseURLEnv, "")
	t.Setenv(adminM25BaseURLEnv, "")
	t.Setenv(adminM70BaseURLEnv, "")
	t.Setenv(adminM71BaseURLEnv, "")
	t.Setenv(adminM72BaseURLEnv, "")
	t.Setenv(adminM84BaseURLEnv, "")

	server, err := newServerForAdminControlPlaneTest()
	if err != nil {
		t.Fatalf("new server with test runtime failed: %v", err)
	}

	_, err = server.adminDashboard.Handler.CreateFinanceRefundHandler(
		context.Background(),
		"admin-1",
		"idem-fallback-finance",
		admindashboardhttp.CreateFinanceRefundRequest{
			TransactionID: "txn-1",
			UserID:        "user-1",
			Amount:        5,
			Reason:        "fallback-path",
		},
	)
	if err != nil {
		t.Fatalf("expected fallback finance client call to succeed in test mode: %v", err)
	}
}

func TestFinanceOwnerClientRetriesTransientFailures(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/v1/admin/transactions/txn-1/refund" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "idem-fin-retry" {
			t.Fatalf("expected idempotency key idem-fin-retry, got %q", got)
		}
		if atomic.LoadInt32(&calls) < 3 {
			writeOwnerErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"refund_id":      "refund_1",
			"transaction_id": "txn-1",
			"user_id":        "user-1",
			"amount":         25.0,
			"reason":         "duplicate",
			"created_at":     "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	client := controlPlaneFinanceClient{baseURL: owner.URL, client: &http.Client{Timeout: ownerRequestTimeout}}
	result, err := client.CreateRefund(context.Background(), "admin-1", "txn-1", "user-1", 25, "duplicate", "idem-fin-retry")
	if err != nil {
		t.Fatalf("expected retry to eventually succeed: %v", err)
	}
	if result.RefundID != "refund_1" {
		t.Fatalf("unexpected refund result: %+v", result)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected exactly 3 attempts, got %d", calls)
	}
}

func TestFinanceOwnerClientTimeoutMapsToDependencyUnavailable(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(120 * time.Millisecond)
		writeOwnerSuccessEnvelope(w, map[string]any{
			"refund_id":      "refund_timeout",
			"transaction_id": "txn-1",
			"user_id":        "user-1",
			"amount":         1.0,
			"reason":         "slow",
			"created_at":     "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	client := controlPlaneFinanceClient{baseURL: owner.URL, client: &http.Client{Timeout: 20 * time.Millisecond}}
	_, err := client.CreateRefund(context.Background(), "admin-1", "txn-1", "user-1", 1, "slow", "idem-fin-timeout")
	if !errors.Is(err, admindashboarderrors.ErrDependencyUnavailable) {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}
	if atomic.LoadInt32(&calls) != ownerRetryMaxAttempt {
		t.Fatalf("expected %d timeout retries, got %d", ownerRetryMaxAttempt, calls)
	}
}

func TestDeveloperPortalOwnerClientRetriesTransientFailures(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/api/v1/developers/api-keys/key-1/rotate" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "idem-dev-retry" {
			t.Fatalf("expected idempotency key idem-dev-retry, got %q", got)
		}
		if atomic.LoadInt32(&calls) < 3 {
			writeOwnerErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"rotation_id":  "rot-1",
			"developer_id": "dev-1",
			"old_key": map[string]any{
				"key_id": "key-1",
			},
			"new_key": map[string]any{
				"key_id": "key-2",
			},
			"created_at": "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	client := controlPlaneDeveloperPortalClient{baseURL: owner.URL, client: &http.Client{Timeout: ownerRequestTimeout}}
	result, err := client.RotateAPIKey(context.Background(), "admin-1", "key-1", "idem-dev-retry")
	if err != nil {
		t.Fatalf("expected retry to eventually succeed: %v", err)
	}
	if result.RotationID != "rot-1" || result.NewKeyID != "key-2" {
		t.Fatalf("unexpected rotate result: %+v", result)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected exactly 3 attempts, got %d", calls)
	}
}

func TestIntegrationHubOwnerClientTimeoutMapsToDependencyUnavailable(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(120 * time.Millisecond)
		writeOwnerSuccessEnvelope(w, map[string]any{
			"execution_id": "exec-timeout",
			"workflow_id":  "wf-1",
			"status":       "success",
			"test_run":     true,
			"started_at":   "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	client := controlPlaneIntegrationHubClient{baseURL: owner.URL, client: &http.Client{Timeout: 20 * time.Millisecond}}
	_, err := client.TestWorkflow(context.Background(), "admin-1", "wf-1", "idem-hub-timeout")
	if !errors.Is(err, admindashboarderrors.ErrDependencyUnavailable) {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}
	if atomic.LoadInt32(&calls) != ownerRetryMaxAttempt {
		t.Fatalf("expected %d timeout retries, got %d", ownerRetryMaxAttempt, calls)
	}
}

func TestWebhookManagerOwnerClientTimeoutMapsToDependencyUnavailable(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(120 * time.Millisecond)
		writeOwnerSuccessEnvelope(w, map[string]any{
			"total_deliveries":      2,
			"successful_deliveries": 2,
			"failed_deliveries":     0,
			"success_rate":          1.0,
			"avg_latency_ms":        23.0,
			"p95_latency_ms":        30.0,
			"p99_latency_ms":        35.0,
			"by_event_type": map[string]any{
				"submission.created": map[string]any{
					"total":       2,
					"success":     2,
					"failed":      0,
					"avg_latency": 23.0,
				},
			},
		})
	}))
	defer owner.Close()

	client := controlPlaneWebhookManagerClient{baseURL: owner.URL, client: &http.Client{Timeout: 20 * time.Millisecond}}
	_, err := client.GetAnalytics(context.Background(), "admin-1", "wh-1")
	if !errors.Is(err, admindashboarderrors.ErrDependencyUnavailable) {
		t.Fatalf("expected dependency unavailable, got %v", err)
	}
	if atomic.LoadInt32(&calls) != ownerRetryMaxAttempt {
		t.Fatalf("expected %d timeout retries, got %d", ownerRetryMaxAttempt, calls)
	}
}

func TestDataMigrationOwnerClientRetriesTransientFailures(t *testing.T) {
	var calls int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.URL.Path != "/plans" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "idem-m84-retry" {
			t.Fatalf("expected idempotency key idem-m84-retry, got %q", got)
		}
		if atomic.LoadInt32(&calls) < 3 {
			writeOwnerErrorEnvelope(w, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"plan_id":           "plan-1",
			"service_name":      "M84-data-migration-service",
			"environment":       "staging",
			"version":           "2026.03.07",
			"plan":              map[string]any{"op": "backfill"},
			"status":            "planned",
			"dry_run":           true,
			"risk_level":        "medium",
			"staging_validated": true,
			"backup_required":   true,
			"created_by":        "admin-1",
			"created_at":        "2026-03-03T00:00:00Z",
			"updated_at":        "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	client := controlPlaneDataMigrationClient{baseURL: owner.URL, client: &http.Client{Timeout: ownerRequestTimeout}}
	result, err := client.CreatePlan(
		context.Background(),
		"admin-1",
		"M84-data-migration-service",
		"staging",
		"2026.03.07",
		map[string]interface{}{"op": "backfill"},
		true,
		"medium",
		"idem-m84-retry",
	)
	if err != nil {
		t.Fatalf("expected retry to eventually succeed: %v", err)
	}
	if result.PlanID != "plan-1" || result.ServiceName != "M84-data-migration-service" {
		t.Fatalf("unexpected migration plan result: %+v", result)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected exactly 3 attempts, got %d", calls)
	}
}

func TestFinanceOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/transactions/txn-1/refund",
		func(baseURL string) error {
			client := controlPlaneFinanceClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateRefund(context.Background(), "admin-1", "txn-1", "user-1", 9, "map", "idem-fin-map")
			return err
		},
	)
}

func TestDeveloperPortalOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/developers/api-keys/key-1/rotate",
		func(baseURL string) error {
			client := controlPlaneDeveloperPortalClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.RotateAPIKey(context.Background(), "admin-1", "key-1", "idem-dev-map")
			return err
		},
	)
}

func TestIntegrationHubOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/workflows/wf-1/test",
		func(baseURL string) error {
			client := controlPlaneIntegrationHubClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.TestWorkflow(context.Background(), "admin-1", "wf-1", "idem-hub-map")
			return err
		},
	)
}

func TestWebhookManagerOwnerClientMapsForbiddenNotFoundConflictReplay(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/webhooks/wh-1/test",
		func(baseURL string) error {
			client := controlPlaneWebhookManagerClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.ReplayWebhook(context.Background(), "admin-1", "wh-1", "idem-webhook-replay-map")
			return err
		},
	)
}

func TestDataMigrationOwnerClientMapsForbiddenNotFoundConflictCreateRun(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/runs",
		func(baseURL string) error {
			client := controlPlaneDataMigrationClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateRun(context.Background(), "admin-1", "plan-1", "idem-m84-run-map")
			return err
		},
	)
}

func TestPayoutOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/payouts/pay-1/retry",
		func(baseURL string) error {
			client := controlPlanePayoutClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.RetryFailedPayout(context.Background(), "admin-1", "pay-1", "map", "idem-pay-map")
			return err
		},
	)
}

func TestBillingOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/invoices/inv-1/refund",
		func(baseURL string) error {
			client := controlPlaneBillingClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateInvoiceRefund(context.Background(), "admin-1", "inv-1", "line-1", 5, "map", "idem-billing-map")
			return err
		},
	)
}

func TestRewardOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/rewards/recalculate",
		func(baseURL string) error {
			client := controlPlaneRewardClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.RecalculateReward(
				context.Background(),
				"admin-1",
				"user-1",
				"sub-1",
				"camp-1",
				1200,
				2.5,
				0,
				time.Time{},
				"map",
				"idem-reward-map",
			)
			return err
		},
	)
}

func TestResolutionOwnerClientMapsForbiddenNotFoundConflictResolve(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/disputes/dispute-1/resolve",
		func(baseURL string) error {
			client := controlPlaneResolutionClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.ResolveDispute(context.Background(), "admin-1", "dispute-1", "resolve", "map", "notes", 3, "idem-dispute-resolve-map")
			return err
		},
	)
}

func TestResolutionOwnerClientMapsForbiddenNotFoundConflictReopen(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/disputes/dispute-1/reopen",
		func(baseURL string) error {
			client := controlPlaneResolutionClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.ResolveDispute(context.Background(), "admin-1", "dispute-1", "reopen", "map", "notes", 0, "idem-dispute-reopen-map")
			return err
		},
	)
}

func TestConsentOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/consent/user-1/withdraw",
		func(baseURL string) error {
			client := controlPlaneConsentClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.WithdrawConsent(context.Background(), "admin-1", "user-1", "all", "map", "idem-consent-map")
			return err
		},
	)
}

func TestPortabilityOwnerClientMapsForbiddenNotFoundConflictExport(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/exports",
		func(baseURL string) error {
			client := controlPlanePortabilityClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateExport(context.Background(), "admin-1", "user-1", "json", "map", "idem-export-map")
			return err
		},
	)
}

func TestPortabilityOwnerClientMapsForbiddenNotFoundConflictErase(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/exports/erase",
		func(baseURL string) error {
			client := controlPlanePortabilityClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateEraseRequest(context.Background(), "admin-1", "user-1", "map", "idem-erase-map")
			return err
		},
	)
}

func TestPortabilityOwnerClientMapsForbiddenNotFoundConflictGet(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/exports/req-1",
		func(baseURL string) error {
			client := controlPlanePortabilityClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.GetExport(context.Background(), "admin-1", "req-1")
			return err
		},
	)
}

func TestRetentionOwnerClientMapsForbiddenNotFoundConflict(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/retention/legal-holds",
		func(baseURL string) error {
			client := controlPlaneRetentionClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CreateLegalHold(context.Background(), "admin-1", "user-1", "messages", "map", nil, "idem-retention-hold-map")
			return err
		},
	)
}

func TestLegalOwnerClientMapsForbiddenNotFoundConflictRelease(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/legal/holds/hold-1/release",
		func(baseURL string) error {
			client := controlPlaneLegalClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.ReleaseHold(context.Background(), "admin-1", "hold-1", "map", "idem-legal-release-map")
			return err
		},
	)
}

func TestLegalOwnerClientMapsForbiddenNotFoundConflictCheck(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/legal/holds/check",
		func(baseURL string) error {
			client := controlPlaneLegalClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.CheckHold(context.Background(), "admin-1", "user", "user-1")
			return err
		},
	)
}

func TestLegalOwnerClientMapsForbiddenNotFoundConflictScan(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/admin/legal/compliance/scan",
		func(baseURL string) error {
			client := controlPlaneLegalClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.RunComplianceScan(context.Background(), "admin-1", "manual", "idem-legal-scan-map")
			return err
		},
	)
}

func TestSupportOwnerClientMapsForbiddenNotFoundConflictGet(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/api/v1/support/admin/tickets/ticket-1",
		func(baseURL string) error {
			client := controlPlaneSupportClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.GetTicket(context.Background(), "admin-1", "ticket-1")
			return err
		},
	)
}

func TestAutoClippingOwnerClientMapsForbiddenNotFoundConflictDeploy(t *testing.T) {
	testOwnerErrorMapping(
		t,
		"/v1/admin/models/deploy",
		func(baseURL string) error {
			client := controlPlaneAutoClippingClient{baseURL: baseURL, client: &http.Client{Timeout: ownerRequestTimeout}}
			_, err := client.DeployModel(context.Background(), "admin-1", admindashboardports.AutoClippingModelDeployInput{
				ModelName:        "xgboost_ensemble",
				VersionTag:       "v1.3.0",
				ModelArtifactKey: "s3://models/xgb_v1.3.0.pkl",
				CanaryPercentage: 5,
				Description:      "improved quality",
				Reason:           "qa approved",
			}, "idem-auto-deploy-map")
			return err
		},
	)
}

func TestAdminFinanceControlPlaneReplayCallsOwnerOnceWithChildIdempotencyKey(t *testing.T) {
	var calls int32
	var lastIdempotencyKey atomic.Value
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		lastIdempotencyKey.Store(r.Header.Get("Idempotency-Key"))
		if r.URL.Path != "/v1/admin/transactions/txn-1/refund" {
			writeOwnerErrorEnvelope(w, http.StatusNotFound, "not_found")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"refund_id":      "refund_owner_1",
			"transaction_id": "txn-1",
			"user_id":        "user-1",
			"amount":         25.0,
			"reason":         "duplicate charge",
			"created_at":     "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, owner.URL)
	t.Setenv(adminM14BaseURLEnv, owner.URL)
	t.Setenv(adminM44BaseURLEnv, owner.URL)
	t.Setenv(adminM05BaseURLEnv, owner.URL)
	t.Setenv(adminM41BaseURLEnv, owner.URL)
	t.Setenv(adminM89BaseURLEnv, owner.URL)
	t.Setenv(adminM50BaseURLEnv, owner.URL)
	t.Setenv(adminM51BaseURLEnv, owner.URL)
	t.Setenv(adminM68BaseURLEnv, owner.URL)
	t.Setenv(adminM69BaseURLEnv, owner.URL)
	t.Setenv(adminM73BaseURLEnv, owner.URL)
	t.Setenv(adminM25BaseURLEnv, owner.URL)
	t.Setenv(adminM70BaseURLEnv, owner.URL)
	t.Setenv(adminM71BaseURLEnv, owner.URL)
	t.Setenv(adminM72BaseURLEnv, owner.URL)
	t.Setenv(adminM84BaseURLEnv, owner.URL)

	server, err := newServerForAdminControlPlaneTest()
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := []byte(`{
		"transaction_id":"txn-1",
		"user_id":"user-1",
		"amount":25,
		"reason":"duplicate charge"
	}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader(body))
	adminHeaders(req1, "admin-1", "idem-parent-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first control-plane response 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/finance/refunds", bytes.NewReader(body))
	adminHeaders(req2, "admin-1", "idem-parent-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay control-plane response 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected owner endpoint to be called once due control-plane idempotency, got %d", calls)
	}
	gotChildKey, _ := lastIdempotencyKey.Load().(string)
	wantChildKey := "m86:finance_refund:idem-parent-1"
	if gotChildKey != wantChildKey {
		t.Fatalf("expected child idempotency key %q, got %q", wantChildKey, gotChildKey)
	}
}

func TestAdminIntegrationKeyControlPlaneReplayCallsOwnerOnceWithChildIdempotencyKey(t *testing.T) {
	var calls int32
	var lastIdempotencyKey atomic.Value
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		lastIdempotencyKey.Store(r.Header.Get("Idempotency-Key"))
		if r.URL.Path != "/api/v1/developers/api-keys/key-1/rotate" {
			writeOwnerErrorEnvelope(w, http.StatusNotFound, "not_found")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"rotation_id":  "rot-owner-1",
			"developer_id": "dev-1",
			"old_key": map[string]any{
				"key_id": "key-1",
			},
			"new_key": map[string]any{
				"key_id": "key-2",
			},
			"created_at": "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, owner.URL)
	t.Setenv(adminM14BaseURLEnv, owner.URL)
	t.Setenv(adminM44BaseURLEnv, owner.URL)
	t.Setenv(adminM05BaseURLEnv, owner.URL)
	t.Setenv(adminM41BaseURLEnv, owner.URL)
	t.Setenv(adminM89BaseURLEnv, owner.URL)
	t.Setenv(adminM50BaseURLEnv, owner.URL)
	t.Setenv(adminM51BaseURLEnv, owner.URL)
	t.Setenv(adminM68BaseURLEnv, owner.URL)
	t.Setenv(adminM69BaseURLEnv, owner.URL)
	t.Setenv(adminM73BaseURLEnv, owner.URL)
	t.Setenv(adminM25BaseURLEnv, owner.URL)
	t.Setenv(adminM70BaseURLEnv, owner.URL)
	t.Setenv(adminM71BaseURLEnv, owner.URL)
	t.Setenv(adminM72BaseURLEnv, owner.URL)
	t.Setenv(adminM84BaseURLEnv, owner.URL)

	server, err := newServerForAdminControlPlaneTest()
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := []byte(`{
		"key_id":"key-1",
		"reason":"routine rotation"
	}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/keys/rotate", bytes.NewReader(body))
	adminHeaders(req1, "admin-1", "idem-parent-int-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first control-plane response 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/integrations/keys/rotate", bytes.NewReader(body))
	adminHeaders(req2, "admin-1", "idem-parent-int-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay control-plane response 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected owner endpoint to be called once due control-plane idempotency, got %d", calls)
	}
	gotChildKey, _ := lastIdempotencyKey.Load().(string)
	wantChildKey := "m86:integration_key_rotate:idem-parent-int-1"
	if gotChildKey != wantChildKey {
		t.Fatalf("expected child idempotency key %q, got %q", wantChildKey, gotChildKey)
	}
}

func TestAdminMigrationPlanControlPlaneReplayCallsOwnerOnceWithChildIdempotencyKey(t *testing.T) {
	var calls int32
	var lastIdempotencyKey atomic.Value
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		lastIdempotencyKey.Store(r.Header.Get("Idempotency-Key"))
		if r.URL.Path != "/plans" {
			writeOwnerErrorEnvelope(w, http.StatusNotFound, "not_found")
			return
		}
		writeOwnerSuccessEnvelope(w, map[string]any{
			"plan_id":           "plan-owner-1",
			"service_name":      "M84-data-migration-service",
			"environment":       "staging",
			"version":           "2026.03.07",
			"plan":              map[string]any{"op": "backfill"},
			"status":            "planned",
			"dry_run":           true,
			"risk_level":        "medium",
			"staging_validated": true,
			"backup_required":   true,
			"created_by":        "admin-1",
			"created_at":        "2026-03-03T00:00:00Z",
			"updated_at":        "2026-03-03T00:00:00Z",
		})
	}))
	defer owner.Close()

	t.Setenv(adminRuntimeModeEnv, "production")
	t.Setenv(adminM39BaseURLEnv, owner.URL)
	t.Setenv(adminM14BaseURLEnv, owner.URL)
	t.Setenv(adminM44BaseURLEnv, owner.URL)
	t.Setenv(adminM05BaseURLEnv, owner.URL)
	t.Setenv(adminM41BaseURLEnv, owner.URL)
	t.Setenv(adminM89BaseURLEnv, owner.URL)
	t.Setenv(adminM50BaseURLEnv, owner.URL)
	t.Setenv(adminM51BaseURLEnv, owner.URL)
	t.Setenv(adminM68BaseURLEnv, owner.URL)
	t.Setenv(adminM69BaseURLEnv, owner.URL)
	t.Setenv(adminM73BaseURLEnv, owner.URL)
	t.Setenv(adminM25BaseURLEnv, owner.URL)
	t.Setenv(adminM70BaseURLEnv, owner.URL)
	t.Setenv(adminM71BaseURLEnv, owner.URL)
	t.Setenv(adminM72BaseURLEnv, owner.URL)
	t.Setenv(adminM84BaseURLEnv, owner.URL)

	server, err := newServerForAdminControlPlaneTest()
	if err != nil {
		t.Fatalf("new server failed: %v", err)
	}

	body := []byte(`{
		"service_name":"M84-data-migration-service",
		"environment":"staging",
		"version":"2026.03.07",
		"plan":{"op":"backfill"},
		"dry_run":true,
		"risk_level":"medium",
		"reason":"approved dry-run"
	}`)

	req1 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/platform-ops/migrations/plans", bytes.NewReader(body))
	adminHeaders(req1, "admin-1", "idem-parent-m84-1")
	rr1 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected first control-plane response 200, got %d body=%s", rr1.Code, rr1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/admin/v1/platform-ops/migrations/plans", bytes.NewReader(body))
	adminHeaders(req2, "admin-1", "idem-parent-m84-1")
	rr2 := httptest.NewRecorder()
	server.mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected replay control-plane response 200, got %d body=%s", rr2.Code, rr2.Body.String())
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected owner endpoint to be called once due control-plane idempotency, got %d", calls)
	}
	gotChildKey, _ := lastIdempotencyKey.Load().(string)
	wantChildKey := "m86:backfill_plan_create:idem-parent-m84-1"
	if gotChildKey != wantChildKey {
		t.Fatalf("expected child idempotency key %q, got %q", wantChildKey, gotChildKey)
	}
}

func testOwnerErrorMapping(t *testing.T, expectedPath string, invoke func(baseURL string) error) {
	t.Helper()

	testCases := []struct {
		name       string
		statusCode int
		errorCode  string
		wantErr    error
	}{
		{name: "forbidden", statusCode: http.StatusForbidden, errorCode: "forbidden", wantErr: admindashboarderrors.ErrUnauthorized},
		{name: "not_found", statusCode: http.StatusNotFound, errorCode: "not_found", wantErr: admindashboarderrors.ErrNotFound},
		{name: "conflict", statusCode: http.StatusConflict, errorCode: "conflict", wantErr: admindashboarderrors.ErrConflict},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != expectedPath {
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
				writeOwnerErrorEnvelope(w, tc.statusCode, tc.errorCode)
			}))
			defer owner.Close()

			err := invoke(owner.URL)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestMapMeshErrorCodeSupportsTopLevelEnvelope(t *testing.T) {
	err, ok := mapMeshErrorCode([]byte(`{"status":"error","code":"forbidden","message":"denied"}`))
	if !ok {
		t.Fatalf("expected top-level envelope to map")
	}
	if !errors.Is(err, admindashboarderrors.ErrUnauthorized) {
		t.Fatalf("expected unauthorized mapping, got %v", err)
	}
}

func writeOwnerSuccessEnvelope(w http.ResponseWriter, data map[string]any) {
	writeOwnerJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func writeOwnerErrorEnvelope(w http.ResponseWriter, status int, code string) {
	writeOwnerJSON(w, status, map[string]any{
		"status": "error",
		"error": map[string]any{
			"code":    code,
			"message": fmt.Sprintf("owner error: %s", code),
		},
	})
}

func writeOwnerJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func newServerForAdminControlPlaneTest() (*Server, error) {
	return New(
		contentlibrarymarketplace.NewInMemoryModule(nil, slog.Default()),
		authorization.NewInMemoryModule(slog.Default()),
		campaignservice.NewInMemoryModule(nil, slog.Default()),
		submissionservice.NewInMemoryModule(nil, slog.Default()),
		distributionservice.NewInMemoryModule(nil, slog.Default()),
		votingengine.NewInMemoryModule(nil, slog.Default()),
		slog.Default(),
		":0",
	)
}
