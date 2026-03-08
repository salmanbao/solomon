package httpserver

import (
	"fmt"
	"net/http"
	"strings"
)

type adminRouteOwner string

const (
	adminRouteOwnerM20 adminRouteOwner = "M20"
	adminRouteOwnerM86 adminRouteOwner = "M86"
)

var adminRouteOwnership = map[string]adminRouteOwner{
	"POST /api/admin/v1/impersonation/start":                                    adminRouteOwnerM20,
	"POST /api/admin/v1/impersonation/end":                                      adminRouteOwnerM20,
	"POST /api/admin/v1/users/{user_id}/wallet/adjust":                          adminRouteOwnerM20,
	"GET /api/admin/v1/users/{user_id}/wallet/history":                          adminRouteOwnerM20,
	"POST /api/admin/v1/users/{user_id}/ban":                                    adminRouteOwnerM20,
	"POST /api/admin/v1/users/{user_id}/unban":                                  adminRouteOwnerM20,
	"GET /api/admin/v1/users/search":                                            adminRouteOwnerM20,
	"POST /api/admin/v1/users/bulk-action":                                      adminRouteOwnerM20,
	"POST /api/admin/v1/campaigns/{campaign_id}/pause":                          adminRouteOwnerM20,
	"PATCH /api/admin/v1/campaigns/{campaign_id}/adjust":                        adminRouteOwnerM20,
	"POST /api/admin/v1/submissions/{submission_id}/override":                   adminRouteOwnerM20,
	"GET /api/admin/v1/feature-flags":                                           adminRouteOwnerM20,
	"POST /api/admin/v1/feature-flags/{flag_key}/toggle":                        adminRouteOwnerM20,
	"GET /api/admin/v1/analytics/dashboard":                                     adminRouteOwnerM20,
	"GET /api/admin/v1/audit-logs":                                              adminRouteOwnerM20,
	"GET /api/admin/v1/audit-logs/export":                                       adminRouteOwnerM20,
	"POST /api/admin/v1/actions/log":                                            adminRouteOwnerM86,
	"POST /api/admin/v1/identity/roles/grant":                                   adminRouteOwnerM86,
	"POST /api/admin/v1/moderation/decisions":                                   adminRouteOwnerM86,
	"POST /api/admin/v1/abuse-prevention/lockouts/{user_id}/release":            adminRouteOwnerM86,
	"POST /api/admin/v1/finance/refunds":                                        adminRouteOwnerM86,
	"POST /api/admin/v1/finance/billing/invoices/{invoice_id}/refund":           adminRouteOwnerM86,
	"POST /api/admin/v1/finance/rewards/recalculate":                            adminRouteOwnerM86,
	"POST /api/admin/v1/finance/affiliates/{affiliate_id}/suspend":              adminRouteOwnerM86,
	"POST /api/admin/v1/finance/affiliates/{affiliate_id}/attributions":         adminRouteOwnerM86,
	"POST /api/admin/v1/finance/payouts/{payout_id}/retry":                      adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/disputes/{dispute_id}/resolve":               adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/disputes/{dispute_id}/reopen":                adminRouteOwnerM86,
	"GET /api/admin/v1/compliance/consent/{user_id}":                            adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/consent/{user_id}/update":                    adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/consent/{user_id}/withdraw":                  adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/exports":                                     adminRouteOwnerM86,
	"GET /api/admin/v1/compliance/exports/{request_id}":                         adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/deletion-requests":                           adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/retention/legal-holds":                       adminRouteOwnerM86,
	"GET /api/admin/v1/compliance/legal-holds/check":                            adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/legal-holds/{hold_id}/release":               adminRouteOwnerM86,
	"POST /api/admin/v1/compliance/legal/compliance-scans":                      adminRouteOwnerM86,
	"GET /api/admin/v1/support/tickets/{ticket_id}":                             adminRouteOwnerM86,
	"GET /api/admin/v1/support/tickets/search":                                  adminRouteOwnerM86,
	"POST /api/admin/v1/support/tickets/{ticket_id}/assign":                     adminRouteOwnerM86,
	"PATCH /api/admin/v1/support/tickets/{ticket_id}":                           adminRouteOwnerM86,
	"POST /api/admin/v1/creator-workflow/editor/campaigns/{campaign_id}/save":   adminRouteOwnerM86,
	"POST /api/admin/v1/creator-workflow/clipping/projects/{project_id}/export": adminRouteOwnerM86,
	"POST /api/admin/v1/creator-workflow/auto-clipping/models/deploy":           adminRouteOwnerM86,
	"POST /api/admin/v1/integrations/keys/rotate":                               adminRouteOwnerM86,
	"POST /api/admin/v1/integrations/workflows/test":                            adminRouteOwnerM86,
	"POST /api/admin/v1/integrations/webhooks/{webhook_id}/replay":              adminRouteOwnerM86,
	"POST /api/admin/v1/integrations/webhooks/{webhook_id}/disable":             adminRouteOwnerM86,
	"GET /api/admin/v1/integrations/webhooks/{webhook_id}/deliveries":           adminRouteOwnerM86,
	"GET /api/admin/v1/integrations/webhooks/{webhook_id}/analytics":            adminRouteOwnerM86,
	"POST /api/admin/v1/platform-ops/migrations/plans":                          adminRouteOwnerM86,
	"GET /api/admin/v1/platform-ops/migrations/plans":                           adminRouteOwnerM86,
	"POST /api/admin/v1/platform-ops/migrations/runs":                           adminRouteOwnerM86,
}

func validateAdminRouteOwnership(owner adminRouteOwner, pattern string) error {
	if !strings.Contains(pattern, " /api/admin/v1/") {
		return fmt.Errorf("admin route must target /api/admin/v1 namespace: %s", pattern)
	}
	expectedOwner, ok := adminRouteOwnership[pattern]
	if !ok {
		return fmt.Errorf("admin route missing ownership declaration: %s", pattern)
	}
	if expectedOwner != owner {
		return fmt.Errorf("admin route ownership mismatch for %s: expected %s got %s", pattern, expectedOwner, owner)
	}
	return nil
}

func (s *Server) registerAdminOwnedRoute(owner adminRouteOwner, pattern string, handler http.HandlerFunc) {
	if err := validateAdminRouteOwnership(owner, pattern); err != nil {
		panic(err)
	}
	s.mux.HandleFunc(pattern, handler)
}
