package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	campaigndiscoveryservice "solomon/contexts/campaign-editorial/campaign-discovery-service"
	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	campaignerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	campaignhttp "solomon/contexts/campaign-editorial/campaign-service/transport/http"
	clippingtoolservice "solomon/contexts/campaign-editorial/clipping-tool-service"
	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	marketplacedomainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	marketplacehttp "solomon/contexts/campaign-editorial/content-library-marketplace/transport/http"
	distributionservice "solomon/contexts/campaign-editorial/distribution-service"
	distributionerrors "solomon/contexts/campaign-editorial/distribution-service/domain/errors"
	distributionhttp "solomon/contexts/campaign-editorial/distribution-service/transport/http"
	submissionservice "solomon/contexts/campaign-editorial/submission-service"
	submissionerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	submissionhttp "solomon/contexts/campaign-editorial/submission-service/transport/http"
	votingengine "solomon/contexts/campaign-editorial/voting-engine"
	votingerrors "solomon/contexts/campaign-editorial/voting-engine/domain/errors"
	votinghttp "solomon/contexts/campaign-editorial/voting-engine/transport/http"
	chatservice "solomon/contexts/community-experience/chat-service"
	chatdomainerrors "solomon/contexts/community-experience/chat-service/domain/errors"
	chathttp "solomon/contexts/community-experience/chat-service/transport/http"
	communityhealthservice "solomon/contexts/community-experience/community-health-service"
	productservice "solomon/contexts/community-experience/product-service"
	productdomainerrors "solomon/contexts/community-experience/product-service/domain/errors"
	producthttp "solomon/contexts/community-experience/product-service/transport/http"
	reputationservice "solomon/contexts/community-experience/reputation-service"
	storefrontservice "solomon/contexts/community-experience/storefront-service"
	subscriptionservice "solomon/contexts/community-experience/subscription-service"
	authorization "solomon/contexts/identity-access/authorization-service"
	authzerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	authzhttp "solomon/contexts/identity-access/authorization-service/transport/http"
	onboardingservice "solomon/contexts/identity-access/onboarding-service"
	superadmindashboard "solomon/contexts/internal-ops/super-admin-dashboard"
	superadmindomainerrors "solomon/contexts/internal-ops/super-admin-dashboard/domain/errors"
	superadminhttp "solomon/contexts/internal-ops/super-admin-dashboard/transport/http"
	teammanagementservice "solomon/contexts/internal-ops/team-management-service"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "solomon/internal/platform/httpserver/docs"
)

type Server struct {
	mux               *http.ServeMux
	logger            *slog.Logger
	addr              string
	httpServer        *http.Server
	marketplace       contentlibrarymarketplace.Module
	authorization     authorization.Module
	campaign          campaignservice.Module
	campaignDiscovery campaigndiscoveryservice.Module
	clippingTool      clippingtoolservice.Module
	submission        submissionservice.Module
	distribution      distributionservice.Module
	voting            votingengine.Module
	chat              chatservice.Module
	reputation        reputationservice.Module
	communityHealth   communityhealthservice.Module
	product           productservice.Module
	storefront        storefrontservice.Module
	subscription      subscriptionservice.Module
	onboarding        onboardingservice.Module
	superAdmin        superadmindashboard.Module
	teamManagement    teammanagementservice.Module
}

func New(
	marketplace contentlibrarymarketplace.Module,
	authorizationModule authorization.Module,
	campaignModule campaignservice.Module,
	submissionModule submissionservice.Module,
	distributionModule distributionservice.Module,
	votingModule votingengine.Module,
	logger *slog.Logger,
	addr string,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if addr == "" {
		addr = ":8080"
	}
	s := &Server{
		mux:               http.NewServeMux(),
		logger:            logger,
		addr:              addr,
		marketplace:       marketplace,
		authorization:     authorizationModule,
		campaign:          campaignModule,
		campaignDiscovery: campaigndiscoveryservice.NewInMemoryModule(logger),
		clippingTool:      clippingtoolservice.NewInMemoryModule(logger),
		submission:        submissionModule,
		distribution:      distributionModule,
		voting:            votingModule,
		chat:              chatservice.NewInMemoryModule(logger),
		reputation:        reputationservice.NewInMemoryModule(logger),
		communityHealth:   communityhealthservice.NewInMemoryModule(logger),
		product:           productservice.NewInMemoryModule(logger),
		storefront:        storefrontservice.NewInMemoryModule(logger),
		subscription:      subscriptionservice.NewInMemoryModule(logger),
		onboarding:        onboardingservice.NewInMemoryModule(logger),
		superAdmin:        superadmindashboard.NewInMemoryModule(logger),
		teamManagement:    teammanagementservice.NewInMemoryModule(logger),
	}
	s.registerRoutes()
	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: s.mux,
	}
	return s
}

func (s *Server) Start() error {
	s.logger.Info("http server starting",
		"event", "http_server_starting",
		"module", "internal/platform/httpserver",
		"layer", "platform",
		"addr", s.addr,
	)
	if s.httpServer == nil {
		s.httpServer = &http.Server{
			Addr:    s.addr,
			Handler: s.mux,
		}
	}
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) registerRoutes() {
	s.mux.Handle("/swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	// M09
	s.mux.HandleFunc("GET /library/clips", s.handleListClips)
	s.mux.HandleFunc("GET /library/clips/{clip_id}", s.handleGetClip)
	s.mux.HandleFunc("GET /library/clips/{clip_id}/preview", s.handleGetClipPreview)
	s.mux.HandleFunc("POST /library/clips/{clip_id}/claim", s.handleClaimClip)
	s.mux.HandleFunc("POST /library/clips/{clip_id}/download", s.handleDownloadClip)
	s.mux.HandleFunc("GET /library/claims", s.handleListClaims)
	s.mux.HandleFunc("GET /v1/marketplace/clips", s.handleListClips)
	s.mux.HandleFunc("GET /v1/marketplace/clips/{clip_id}", s.handleGetClip)
	s.mux.HandleFunc("GET /v1/marketplace/clips/{clip_id}/preview", s.handleGetClipPreview)
	s.mux.HandleFunc("POST /v1/marketplace/clips/{clip_id}/claim", s.handleClaimClip)
	s.mux.HandleFunc("POST /v1/marketplace/clips/{clip_id}/download", s.handleDownloadClip)
	s.mux.HandleFunc("GET /v1/marketplace/claims", s.handleListClaims)

	// M21
	s.mux.HandleFunc("POST /api/authz/v1/check", s.handleAuthzCheck)
	s.mux.HandleFunc("POST /api/authz/v1/check-batch", s.handleAuthzCheckBatch)
	s.mux.HandleFunc("GET /api/authz/v1/users/{user_id}/roles", s.handleAuthzListUserRoles)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/grant", s.handleAuthzGrantRole)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/revoke", s.handleAuthzRevokeRole)
	s.mux.HandleFunc("POST /api/authz/v1/delegations", s.handleAuthzCreateDelegation)

	// M20
	s.mux.HandleFunc("POST /api/admin/v1/impersonation/start", s.handleAdminStartImpersonation)
	s.mux.HandleFunc("POST /api/admin/v1/impersonation/end", s.handleAdminEndImpersonation)
	s.mux.HandleFunc("POST /api/admin/v1/users/{user_id}/wallet/adjust", s.handleAdminAdjustWallet)
	s.mux.HandleFunc("GET /api/admin/v1/users/{user_id}/wallet/history", s.handleAdminWalletHistory)
	s.mux.HandleFunc("POST /api/admin/v1/users/{user_id}/ban", s.handleAdminBanUser)
	s.mux.HandleFunc("POST /api/admin/v1/users/{user_id}/unban", s.handleAdminUnbanUser)
	s.mux.HandleFunc("GET /api/admin/v1/users/search", s.handleAdminSearchUsers)
	s.mux.HandleFunc("POST /api/admin/v1/users/bulk-action", s.handleAdminBulkAction)
	s.mux.HandleFunc("POST /api/admin/v1/campaigns/{campaign_id}/pause", s.handleAdminPauseCampaign)
	s.mux.HandleFunc("PATCH /api/admin/v1/campaigns/{campaign_id}/adjust", s.handleAdminAdjustCampaign)
	s.mux.HandleFunc("POST /api/admin/v1/submissions/{submission_id}/override", s.handleAdminOverrideSubmission)
	s.mux.HandleFunc("GET /api/admin/v1/feature-flags", s.handleAdminFeatureFlags)
	s.mux.HandleFunc("POST /api/admin/v1/feature-flags/{flag_key}/toggle", s.handleAdminToggleFeatureFlag)
	s.mux.HandleFunc("GET /api/admin/v1/analytics/dashboard", s.handleAdminAnalyticsDashboard)
	s.mux.HandleFunc("GET /api/admin/v1/audit-logs", s.handleAdminAuditLogs)
	s.mux.HandleFunc("GET /api/admin/v1/audit-logs/export", s.handleAdminAuditLogsExport)

	// M60
	s.mux.HandleFunc("GET /api/v1/products", s.handleProductList)
	s.mux.HandleFunc("POST /api/v1/products", s.handleProductCreate)
	s.mux.HandleFunc("GET /api/v1/products/{product_id}/access", s.handleProductCheckAccess)
	s.mux.HandleFunc("POST /api/v1/products/{id}/purchase", s.handleProductPurchase)
	s.mux.HandleFunc("POST /api/v1/products/{id}/fulfill", s.handleProductFulfill)
	s.mux.HandleFunc("POST /api/v1/admin/products/{product_id}/inventory", s.handleProductAdjustInventory)
	s.mux.HandleFunc("PUT /api/v1/products/{product_id}/media/reorder", s.handleProductMediaReorder)
	s.mux.HandleFunc("GET /api/v1/discover", s.handleProductDiscover)
	s.mux.HandleFunc("GET /api/v1/search", s.handleProductSearch)
	s.mux.HandleFunc("GET /api/v1/users/{user_id}/data-export", s.handleProductUserDataExport)
	s.mux.HandleFunc("POST /api/v1/users/{user_id}/delete-account", s.handleProductDeleteAccount)

	// M92
	s.mux.HandleFunc("POST /storefronts", s.handleStorefrontCreate)
	s.mux.HandleFunc("PATCH /storefronts/{storefrontId}", s.handleStorefrontUpdate)
	s.mux.HandleFunc("GET /storefronts/{identifier}", s.handleStorefrontGet)
	s.mux.HandleFunc("POST /storefronts/{storefrontId}/publish", s.handleStorefrontPublish)
	s.mux.HandleFunc("POST /storefronts/{storefrontId}/reports", s.handleStorefrontReport)
	s.mux.HandleFunc("POST /api/storefront/v1/internal/events/product-published", s.handleStorefrontProductPublishedEvent)
	s.mux.HandleFunc("POST /api/storefront/v1/internal/projections/subscriptions", s.handleStorefrontSubscriptionProjection)

	// M46
	s.mux.HandleFunc("POST /api/v1/chat/messages", s.handleChatPostMessage)
	s.mux.HandleFunc("PUT /api/v1/chat/messages/{message_id}", s.handleChatEditMessage)
	s.mux.HandleFunc("DELETE /api/v1/chat/messages/{message_id}", s.handleChatDeleteMessage)
	s.mux.HandleFunc("GET /api/v1/chat/channels/{channel_id}/messages", s.handleChatListMessages)
	s.mux.HandleFunc("GET /api/v1/chat/channels/{channel_id}/unread-count", s.handleChatUnreadCount)
	s.mux.HandleFunc("GET /api/v1/chat/search", s.handleChatSearch)
	s.mux.HandleFunc("GET /api/v1/chat/messages", s.handleChatBackfill)
	s.mux.HandleFunc("GET /api/v1/chat/poll", s.handleChatPoll)
	s.mux.HandleFunc("GET /api/v1/chat/messages/subscribe", s.handleChatPoll)
	s.mux.HandleFunc("POST /api/v1/chat/messages/{message_id}/reactions", s.handleChatAddReaction)
	s.mux.HandleFunc("DELETE /api/v1/chat/messages/{message_id}/reactions/{emoji}", s.handleChatRemoveReaction)
	s.mux.HandleFunc("POST /api/v1/chat/messages/{message_id}/pin", s.handleChatPinMessage)
	s.mux.HandleFunc("POST /api/v1/chat/messages/{message_id}/report", s.handleChatReportMessage)
	s.mux.HandleFunc("POST /api/v1/chat/messages/{message_id}/attachments", s.handleChatAddAttachment)
	s.mux.HandleFunc("GET /api/v1/chat/messages/{message_id}/attachments/{attachment_id}", s.handleChatGetAttachment)
	s.mux.HandleFunc("PUT /api/v1/chat/threads/{thread_id}/lock", s.handleChatLockThread)
	s.mux.HandleFunc("PUT /api/v1/chat/servers/{server_id}/moderators", s.handleChatUpdateModerators)
	s.mux.HandleFunc("POST /api/v1/chat/users/{user_id}/mute", s.handleChatMuteUser)
	s.mux.HandleFunc("POST /api/v1/chat/export", s.handleChatExport)

	// M49
	s.mux.HandleFunc("POST /webhooks/chat/message", s.handleCommunityHealthWebhook)
	s.mux.HandleFunc("GET /api/v1/community-health/{server_id}/health-score", s.handleCommunityHealthGetScore)
	s.mux.HandleFunc("GET /api/v1/community-health/{server_id}/user-risk/{user_id}", s.handleCommunityHealthGetUserRisk)

	// M48
	s.mux.HandleFunc("GET /api/v1/reputation/user/{user_id}", s.handleReputationGetUser)
	s.mux.HandleFunc("GET /api/v1/reputation/leaderboard", s.handleReputationLeaderboard)

	// M23
	s.mux.HandleFunc("GET /api/discover/v1/campaigns/browse", s.handleDiscoverBrowse)
	s.mux.HandleFunc("GET /api/discover/v1/campaigns/search", s.handleDiscoverSearch)
	s.mux.HandleFunc("GET /api/discover/v1/campaigns/{campaign_id}", s.handleDiscoverCampaignDetails)
	s.mux.HandleFunc("POST /api/discover/v1/campaigns/{campaign_id}/bookmark", s.handleDiscoverBookmark)

	// M24
	s.mux.HandleFunc("POST /api/clipping/v1/projects", s.handleClippingCreateProject)
	s.mux.HandleFunc("GET /api/clipping/v1/projects/{project_id}", s.handleClippingGetProject)
	s.mux.HandleFunc("PATCH /api/clipping/v1/projects/{project_id}/timeline", s.handleClippingUpdateTimeline)
	s.mux.HandleFunc("POST /api/clipping/v1/projects/{project_id}/timeline/insert", s.handleClippingInsertTimeline)
	s.mux.HandleFunc("GET /api/clipping/v1/projects/{project_id}/suggestions", s.handleClippingGetSuggestions)
	s.mux.HandleFunc("POST /api/clipping/v1/projects/{project_id}/export", s.handleClippingRequestExport)
	s.mux.HandleFunc("GET /api/clipping/v1/projects/{project_id}/export/{export_id}/status", s.handleClippingGetExportStatus)
	s.mux.HandleFunc("POST /api/clipping/v1/projects/{project_id}/submit", s.handleClippingSubmit)

	// M61
	s.mux.HandleFunc("POST /api/v1/subscriptions", s.handleSubscriptionCreate)
	s.mux.HandleFunc("POST /api/v1/subscriptions/{subscription_id}/change-plan", s.handleSubscriptionChangePlan)
	s.mux.HandleFunc("POST /api/v1/subscriptions/{subscription_id}/cancel", s.handleSubscriptionCancel)

	// M22
	s.mux.HandleFunc("GET /api/onboarding/v1/flow", s.handleOnboardingGetFlow)
	s.mux.HandleFunc("POST /api/onboarding/v1/steps/{step_key}/complete", s.handleOnboardingCompleteStep)
	s.mux.HandleFunc("POST /api/onboarding/v1/skip", s.handleOnboardingSkip)
	s.mux.HandleFunc("POST /api/onboarding/v1/resume", s.handleOnboardingResume)
	s.mux.HandleFunc("GET /api/onboarding/v1/admin/flows", s.handleOnboardingAdminFlows)
	s.mux.HandleFunc("POST /api/onboarding/v1/internal/events/user-registered", s.handleOnboardingUserRegisteredEvent)

	// M87
	s.mux.HandleFunc("POST /teams", s.handleTeamCreate)
	s.mux.HandleFunc("POST /teams/{teamId}/invites", s.handleTeamCreateInvite)
	s.mux.HandleFunc("POST /teams/invites/{token}/accept", s.handleTeamAcceptInvite)
	s.mux.HandleFunc("POST /teams/{teamId}/members/{memberId}/role", s.handleTeamUpdateMemberRole)
	s.mux.HandleFunc("DELETE /teams/{teamId}/members/{memberId}", s.handleTeamRemoveMember)
	s.mux.HandleFunc("GET /teams/{teamId}", s.handleTeamGet)
	s.mux.HandleFunc("GET /teams/{teamId}/membership", s.handleTeamMembership)
	s.mux.HandleFunc("GET /teams/{teamId}/audit-logs", s.handleTeamAuditLogs)
	s.mux.HandleFunc("GET /teams/{teamId}/exports/members", s.handleTeamExportMembers)

	// M87 delegated path compatibility
	s.mux.HandleFunc("POST /v1/team", s.handleTeamCreate)
	s.mux.HandleFunc("POST /v1/team/{team_id}/invites", s.handleTeamCreateInviteV1)
	s.mux.HandleFunc("POST /v1/team/invites/{invite_id}/accept", s.handleTeamAcceptInviteV1)
	s.mux.HandleFunc("PUT /v1/team/{team_id}/members/{member_id}/role", s.handleTeamUpdateMemberRoleV1)
	s.mux.HandleFunc("DELETE /v1/team/{team_id}/members/{member_id}", s.handleTeamRemoveMemberV1)

	// M04
	s.mux.HandleFunc("POST /v1/campaigns", s.handleCampaignCreate)
	s.mux.HandleFunc("GET /v1/campaigns", s.handleCampaignList)
	s.mux.HandleFunc("GET /v1/campaigns/{campaign_id}", s.handleCampaignGet)
	s.mux.HandleFunc("PUT /v1/campaigns/{campaign_id}", s.handleCampaignUpdate)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/launch", s.handleCampaignLaunch)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/pause", s.handleCampaignPause)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/resume", s.handleCampaignResume)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/complete", s.handleCampaignComplete)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/media/upload-url", s.handleCampaignMediaUploadURL)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/media/{media_id}/confirm", s.handleCampaignMediaConfirm)
	s.mux.HandleFunc("GET /v1/campaigns/{campaign_id}/media", s.handleCampaignMediaList)
	s.mux.HandleFunc("GET /v1/campaigns/{campaign_id}/analytics", s.handleCampaignAnalytics)
	s.mux.HandleFunc("GET /v1/campaigns/{campaign_id}/analytics/export", s.handleCampaignAnalyticsExport)
	s.mux.HandleFunc("POST /v1/campaigns/{campaign_id}/budget/increase", s.handleCampaignIncreaseBudget)

	// M26
	s.mux.HandleFunc("POST /submissions", s.handleSubmissionCreate)
	s.mux.HandleFunc("GET /submissions/{submission_id}", s.handleSubmissionGet)
	s.mux.HandleFunc("GET /submissions", s.handleSubmissionList)
	s.mux.HandleFunc("POST /submissions/{submission_id}/approve", s.handleSubmissionApprove)
	s.mux.HandleFunc("POST /submissions/{submission_id}/reject", s.handleSubmissionReject)
	s.mux.HandleFunc("POST /submissions/{submission_id}/report", s.handleSubmissionReport)
	s.mux.HandleFunc("POST /submissions/bulk-operations", s.handleSubmissionBulkOperation)
	s.mux.HandleFunc("GET /submissions/{submission_id}/analytics", s.handleSubmissionAnalytics)
	s.mux.HandleFunc("GET /dashboard/creator", s.handleSubmissionCreatorDashboard)
	s.mux.HandleFunc("GET /dashboard/brand", s.handleSubmissionBrandDashboard)
	s.mux.HandleFunc("POST /v1/submissions", s.handleSubmissionCreate)
	s.mux.HandleFunc("GET /v1/submissions/{submission_id}", s.handleSubmissionGet)
	s.mux.HandleFunc("GET /v1/submissions", s.handleSubmissionList)
	s.mux.HandleFunc("POST /v1/submissions/{submission_id}/approve", s.handleSubmissionApprove)
	s.mux.HandleFunc("POST /v1/submissions/{submission_id}/reject", s.handleSubmissionReject)
	s.mux.HandleFunc("POST /v1/submissions/{submission_id}/report", s.handleSubmissionReport)
	s.mux.HandleFunc("POST /v1/submissions/bulk-operations", s.handleSubmissionBulkOperation)
	s.mux.HandleFunc("GET /v1/submissions/{submission_id}/analytics", s.handleSubmissionAnalytics)
	s.mux.HandleFunc("GET /v1/dashboard/creator", s.handleSubmissionCreatorDashboard)
	s.mux.HandleFunc("GET /v1/dashboard/brand", s.handleSubmissionBrandDashboard)

	// M31
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/overlays", s.handleDistributionAddOverlay)
	s.mux.HandleFunc("GET /api/v1/distribution/items/{id}/preview", s.handleDistributionPreview)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/schedule", s.handleDistributionSchedule)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/reschedule", s.handleDistributionReschedule)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/publish", s.handleDistributionPublish)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/download", s.handleDistributionDownload)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/publish-multi", s.handleDistributionPublishMulti)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/retry", s.handleDistributionRetry)

	// M08
	s.mux.HandleFunc("POST /v1/votes", s.handleVotingCreate)
	s.mux.HandleFunc("DELETE /v1/votes/{vote_id}", s.handleVotingRetract)
	s.mux.HandleFunc("GET /v1/votes/submissions/{submission_id}", s.handleVotingSubmissionVotes)
	s.mux.HandleFunc("GET /v1/votes/leaderboard", s.handleVotingLegacyLeaderboard)
	s.mux.HandleFunc("GET /v1/leaderboards/campaign/{campaign_id}", s.handleVotingCampaignLeaderboard)
	s.mux.HandleFunc("GET /v1/leaderboards/round/{round_id}", s.handleVotingRoundLeaderboard)
	s.mux.HandleFunc("GET /v1/leaderboards/trending", s.handleVotingTrendingLeaderboard)
	s.mux.HandleFunc("GET /v1/leaderboards/creator/{user_id}", s.handleVotingCreatorLeaderboard)
	s.mux.HandleFunc("GET /v1/rounds/{round_id}/results", s.handleVotingRoundResults)
	s.mux.HandleFunc("GET /v1/analytics/votes", s.handleVotingAnalytics)
	s.mux.HandleFunc("POST /v1/quarantine/{quarantine_id}/action", s.handleVotingQuarantineAction)
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any, onError func(http.ResponseWriter, int, string, string)) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		onError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return false
	}
	return true
}

func getUserID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func getRequestID(r *http.Request) string {
	if requestID := strings.TrimSpace(r.Header.Get("X-Request-Id")); requestID != "" {
		return requestID
	}
	return strings.TrimSpace(r.Header.Get("Idempotency-Key"))
}

func resolveClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func resolveAuthzUserID(bodyUserID string, r *http.Request) string {
	if strings.TrimSpace(bodyUserID) != "" {
		return bodyUserID
	}
	if fromHeader := strings.TrimSpace(r.Header.Get("X-User-Id")); fromHeader != "" {
		return fromHeader
	}
	return strings.TrimSpace(r.Header.Get("X-Subject-Id"))
}

func requireAuthzAuthorization(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
		writeAuthzError(w, http.StatusUnauthorized, "unauthorized", "Authorization header is required")
		return false
	}
	return true
}

func requireAuthzRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeAuthzError(w, http.StatusBadRequest, "request_id_required", "X-Request-Id header is required")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeMarketplaceError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, marketplacehttp.ErrorResponse{Code: code, Message: message})
}

func writeAuthzError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, authzhttp.ErrorResponse{Code: code, Message: message})
}

func writeCampaignError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, campaignhttp.ErrorResponse{Code: code, Message: message})
}

func writeSubmissionError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, submissionhttp.ErrorResponse{Code: code, Message: message})
}

func writeProductError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, producthttp.ErrorResponse{Code: code, Message: message})
}

func writeChatError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, chathttp.ErrorResponse{Code: code, Message: message})
}

func writeSuperAdminError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, superadminhttp.ErrorResponse{Code: code, Message: message})
}

func getAdminID(r *http.Request) string {
	if adminID := strings.TrimSpace(r.Header.Get("X-Admin-Id")); adminID != "" {
		return adminID
	}
	return strings.TrimSpace(r.Header.Get("X-User-Id"))
}

func requireAdminAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeSuperAdminError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireAdminRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireAdminMFA(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-MFA-Code")) == "" {
		writeSuperAdminError(w, http.StatusUnauthorized, "mfa_required", "X-MFA-Code header is required")
		return false
	}
	return true
}

func requireAdminID(w http.ResponseWriter, r *http.Request) (string, bool) {
	adminID := getAdminID(r)
	if adminID == "" {
		writeSuperAdminError(w, http.StatusUnauthorized, "missing_admin", "X-Admin-Id header is required")
		return "", false
	}
	return adminID, true
}

func requireAdminHeaders(w http.ResponseWriter, r *http.Request) bool {
	if !requireAdminAuthorization(w, r) {
		return false
	}
	if !requireAdminMFA(w, r) {
		return false
	}
	if !requireAdminRequestID(w, r) {
		return false
	}
	return true
}

func requireAdminIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func requireProductAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeProductError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireProductRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeProductError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireProductUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeProductError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireProductIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeProductError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func requireChatAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeChatError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireChatRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeChatError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireChatUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeChatError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireChatIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeChatError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func parseOptionalRFC3339(raw string) (time.Time, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, false, nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC(), true, nil
	}
	if day, err := time.Parse("2006-01-02", raw); err == nil {
		return day.UTC(), true, nil
	}
	return time.Time{}, true, errors.New("invalid_time")
}

func requireSubmissionAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeSubmissionError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	if strings.TrimSpace(parts[1]) == "" {
		writeSubmissionError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireSubmissionUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := getUserID(r)
	if userID == "" {
		writeSubmissionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireSubmissionRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeSubmissionError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireSubmissionIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeSubmissionError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func isCanonicalMarketplaceRoute(r *http.Request) bool {
	return strings.HasPrefix(strings.TrimSpace(r.URL.Path), "/v1/marketplace/")
}

func requireMarketplaceAuthorization(w http.ResponseWriter, r *http.Request, strict bool) bool {
	if !strict {
		return true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireMarketplaceRequestID(w http.ResponseWriter, r *http.Request, strict bool) bool {
	if !strict {
		return true
	}
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeMarketplaceError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireMarketplaceIdempotencyKey(w http.ResponseWriter, r *http.Request, strict bool) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if strict && idempotencyKey == "" {
		writeMarketplaceError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func writeDistributionError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, distributionhttp.ErrorResponse{Code: code, Message: message})
}

func writeVotingError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, votinghttp.ErrorResponse{Code: code, Message: message})
}

func writeMarketplaceDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, marketplacedomainerrors.ErrClipNotFound):
		writeMarketplaceError(w, http.StatusNotFound, "clip_not_found", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrClaimNotFound):
		writeMarketplaceError(w, http.StatusNotFound, "claim_not_found", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrExclusiveClaimConflict):
		writeMarketplaceError(w, http.StatusConflict, "exclusive_claim_conflict", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrClaimLimitReached):
		writeMarketplaceError(w, http.StatusConflict, "claim_limit_reached", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrClipUnavailable):
		writeMarketplaceError(w, http.StatusGone, "clip_unavailable", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrClaimRequired):
		writeMarketplaceError(w, http.StatusForbidden, "claim_required", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrDownloadLimitReached):
		writeMarketplaceError(w, http.StatusTooManyRequests, "download_limit_reached", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrIdempotencyKeyConflict):
		writeMarketplaceError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrInvalidListFilter):
		writeMarketplaceError(w, http.StatusBadRequest, "invalid_list_filter", err.Error())
	case errors.Is(err, marketplacedomainerrors.ErrInvalidClaimRequest):
		writeMarketplaceError(w, http.StatusBadRequest, "invalid_claim_request", err.Error())
	default:
		writeMarketplaceError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeAuthzDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, authzerrors.ErrInvalidPermission):
		writeAuthzError(w, http.StatusUnprocessableEntity, "invalid_permission", err.Error())
	case errors.Is(err, authzerrors.ErrInvalidUserID),
		errors.Is(err, authzerrors.ErrInvalidRoleID),
		errors.Is(err, authzerrors.ErrInvalidAdminID),
		errors.Is(err, authzerrors.ErrInvalidDelegation):
		writeAuthzError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, authzerrors.ErrRoleNotFound),
		errors.Is(err, authzerrors.ErrUserNotFound):
		writeAuthzError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, authzerrors.ErrRoleAlreadyAssigned),
		errors.Is(err, authzerrors.ErrRoleNotAssigned),
		errors.Is(err, authzerrors.ErrIdempotencyConflict):
		writeAuthzError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, authzerrors.ErrIdempotencyKeyRequired):
		writeAuthzError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, authzerrors.ErrForbidden):
		writeAuthzError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeAuthzError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeSuperAdminDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, superadmindomainerrors.ErrUserNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "user_not_found", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrCampaignNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "campaign_not_found", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrSubmissionNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "submission_not_found", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrFlagNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "flag_not_found", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrImpersonationNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "impersonation_not_found", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrImpersonationAlreadyActive):
		writeSuperAdminError(w, http.StatusConflict, "impersonation_already_active", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrImpersonationAlreadyEnded):
		writeSuperAdminError(w, http.StatusConflict, "impersonation_already_ended", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrAlreadyBanned):
		writeSuperAdminError(w, http.StatusConflict, "already_banned", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrNotBanned):
		writeSuperAdminError(w, http.StatusConflict, "not_currently_banned", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrBulkActionConflict):
		writeSuperAdminError(w, http.StatusConflict, "bulk_action_conflict", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrIdempotencyKeyRequired):
		writeSuperAdminError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrIdempotencyConflict):
		writeSuperAdminError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrInvalidRequest):
		writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrUnprocessable):
		writeSuperAdminError(w, http.StatusUnprocessableEntity, "unprocessable", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrForbidden):
		writeSuperAdminError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrConflict):
		writeSuperAdminError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, superadmindomainerrors.ErrNotFound):
		writeSuperAdminError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		writeSuperAdminError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeProductDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, productdomainerrors.ErrProductNotFound),
		errors.Is(err, productdomainerrors.ErrPurchaseNotFound),
		errors.Is(err, productdomainerrors.ErrNotFound):
		writeProductError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, productdomainerrors.ErrPaymentRequired):
		writeProductError(w, http.StatusPaymentRequired, "payment_required", err.Error())
	case errors.Is(err, productdomainerrors.ErrSoldOut):
		writeProductError(w, http.StatusBadRequest, "sold_out", err.Error())
	case errors.Is(err, productdomainerrors.ErrInvalidRequest),
		errors.Is(err, productdomainerrors.ErrIdempotencyKeyRequired):
		writeProductError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, productdomainerrors.ErrIdempotencyConflict),
		errors.Is(err, productdomainerrors.ErrConflict):
		writeProductError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, productdomainerrors.ErrForbidden):
		writeProductError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeProductError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeChatDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, chatdomainerrors.ErrMessageNotFound),
		errors.Is(err, chatdomainerrors.ErrAttachmentNotFound),
		errors.Is(err, chatdomainerrors.ErrNotFound):
		writeChatError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, chatdomainerrors.ErrInvalidRequest),
		errors.Is(err, chatdomainerrors.ErrIdempotencyKeyRequired):
		writeChatError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, chatdomainerrors.ErrIdempotencyConflict),
		errors.Is(err, chatdomainerrors.ErrConflict):
		writeChatError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, chatdomainerrors.ErrRateLimited):
		writeChatError(w, http.StatusTooManyRequests, "rate_limit_exceeded", err.Error())
	case errors.Is(err, chatdomainerrors.ErrForbidden):
		writeChatError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeChatError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeCampaignDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, campaignerrors.ErrCampaignNotFound):
		writeCampaignError(w, http.StatusNotFound, "campaign_not_found", err.Error())
	case errors.Is(err, campaignerrors.ErrMediaFileTooLarge):
		writeCampaignError(w, http.StatusRequestEntityTooLarge, "file_too_large", err.Error())
	case errors.Is(err, campaignerrors.ErrInvalidCampaignInput),
		errors.Is(err, campaignerrors.ErrIdempotencyKeyRequired),
		errors.Is(err, campaignerrors.ErrUnsupportedMediaType),
		errors.Is(err, campaignerrors.ErrDeadlineTooSoon),
		errors.Is(err, campaignerrors.ErrMissingReadyMedia):
		writeCampaignError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, campaignerrors.ErrCampaignNotEditable),
		errors.Is(err, campaignerrors.ErrCampaignEditRestricted),
		errors.Is(err, campaignerrors.ErrInvalidStateTransition),
		errors.Is(err, campaignerrors.ErrInvalidBudgetIncrease),
		errors.Is(err, campaignerrors.ErrMediaAlreadyConfirmed):
		writeCampaignError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, campaignerrors.ErrMediaNotFound):
		writeCampaignError(w, http.StatusNotFound, "media_not_found", err.Error())
	case errors.Is(err, campaignerrors.ErrMediaLimitReached):
		writeCampaignError(w, http.StatusConflict, "media_limit_reached", err.Error())
	case errors.Is(err, campaignerrors.ErrIdempotencyKeyConflict):
		writeCampaignError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	default:
		writeCampaignError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeSubmissionDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, submissionerrors.ErrSubmissionNotFound):
		writeSubmissionError(w, http.StatusNotFound, "submission_not_found", err.Error())
	case errors.Is(err, submissionerrors.ErrInvalidSubmissionInput):
		writeSubmissionError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, submissionerrors.ErrInvalidSubmissionURL):
		writeSubmissionError(w, http.StatusBadRequest, "invalid_submission_url", err.Error())
	case errors.Is(err, submissionerrors.ErrUnsupportedPlatform):
		writeSubmissionError(w, http.StatusBadRequest, "unsupported_platform", err.Error())
	case errors.Is(err, submissionerrors.ErrPlatformNotAllowed):
		writeSubmissionError(w, http.StatusBadRequest, "platform_not_allowed", err.Error())
	case errors.Is(err, submissionerrors.ErrCampaignNotFound):
		writeSubmissionError(w, http.StatusNotFound, "campaign_not_found", err.Error())
	case errors.Is(err, submissionerrors.ErrCampaignNotActive):
		writeSubmissionError(w, http.StatusConflict, "campaign_not_active", err.Error())
	case errors.Is(err, submissionerrors.ErrDuplicateSubmission):
		writeSubmissionError(w, http.StatusConflict, "duplicate_submission", err.Error())
	case errors.Is(err, submissionerrors.ErrAlreadyReported):
		writeSubmissionError(w, http.StatusConflict, "already_reported", err.Error())
	case errors.Is(err, submissionerrors.ErrIdempotencyKeyRequired):
		writeSubmissionError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, submissionerrors.ErrIdempotencyKeyConflict):
		writeSubmissionError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, submissionerrors.ErrInvalidStatusTransition):
		writeSubmissionError(w, http.StatusConflict, "invalid_status_transition", err.Error())
	case errors.Is(err, submissionerrors.ErrUnauthorizedActor):
		writeSubmissionError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeSubmissionError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeDistributionDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, distributionerrors.ErrDistributionItemNotFound):
		writeDistributionError(w, http.StatusNotFound, "distribution_item_not_found", err.Error())
	case errors.Is(err, distributionerrors.ErrDistributionItemExists):
		writeDistributionError(w, http.StatusConflict, "distribution_item_exists", err.Error())
	case errors.Is(err, distributionerrors.ErrInvalidDistributionInput),
		errors.Is(err, distributionerrors.ErrInvalidScheduleWindow),
		errors.Is(err, distributionerrors.ErrInvalidTimezone):
		writeDistributionError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, distributionerrors.ErrUnsupportedPlatform):
		writeDistributionError(w, http.StatusBadRequest, "unsupported_platform", err.Error())
	case errors.Is(err, distributionerrors.ErrUnauthorizedInfluencer):
		writeDistributionError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, distributionerrors.ErrInvalidStateTransition):
		writeDistributionError(w, http.StatusConflict, "invalid_state_transition", err.Error())
	default:
		writeDistributionError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeVotingDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, votingerrors.ErrVoteNotFound):
		writeVotingError(w, http.StatusNotFound, "vote_not_found", err.Error())
	case errors.Is(err, votingerrors.ErrCampaignNotFound):
		writeVotingError(w, http.StatusNotFound, "campaign_not_found", err.Error())
	case errors.Is(err, votingerrors.ErrRoundNotFound):
		writeVotingError(w, http.StatusNotFound, "round_not_found", err.Error())
	case errors.Is(err, votingerrors.ErrQuarantineNotFound):
		writeVotingError(w, http.StatusNotFound, "quarantine_not_found", err.Error())
	case errors.Is(err, votingerrors.ErrInvalidVoteInput):
		writeVotingError(w, http.StatusBadRequest, "invalid_vote_input", err.Error())
	case errors.Is(err, votingerrors.ErrInvalidQuarantineAction):
		writeVotingError(w, http.StatusBadRequest, "invalid_quarantine_action", err.Error())
	case errors.Is(err, votingerrors.ErrIdempotencyKeyRequired):
		writeVotingError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, votingerrors.ErrAlreadyRetracted):
		writeVotingError(w, http.StatusConflict, "already_retracted", err.Error())
	case errors.Is(err, votingerrors.ErrConflict):
		writeVotingError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, votingerrors.ErrIdempotencyConflict):
		writeVotingError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, votingerrors.ErrCampaignNotActive):
		writeVotingError(w, http.StatusConflict, "campaign_not_active", err.Error())
	case errors.Is(err, votingerrors.ErrRoundClosed):
		writeVotingError(w, http.StatusConflict, "round_closed", err.Error())
	case errors.Is(err, votingerrors.ErrQuarantineResolved):
		writeVotingError(w, http.StatusConflict, "quarantine_resolved", err.Error())
	case errors.Is(err, votingerrors.ErrSelfVoteForbidden):
		writeVotingError(w, http.StatusForbidden, "self_vote_forbidden", err.Error())
	case errors.Is(err, votingerrors.ErrSubmissionNotFound):
		writeVotingError(w, http.StatusNotFound, "submission_not_found", err.Error())
	default:
		writeVotingError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (s *Server) handleListClips(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}

	query := r.URL.Query()
	req := marketplacehttp.ListClipsRequest{
		Niche:          query["niche"],
		DurationBucket: query.Get("duration_bucket"),
		PopularitySort: query.Get("popularity_sort"),
		Status:         query.Get("status"),
		Cursor:         query.Get("cursor"),
	}
	if limitRaw := query.Get("limit"); limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil {
			writeMarketplaceError(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
			return
		}
		req.Limit = limit
	}
	resp, err := s.marketplace.Handler.ListClipsHandler(r.Context(), req)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetClip(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}

	resp, err := s.marketplace.Handler.GetClipHandler(r.Context(), r.PathValue("clip_id"))
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetClipPreview(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}

	resp, err := s.marketplace.Handler.GetClipPreviewHandler(r.Context(), r.PathValue("clip_id"))
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClaimClip(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}
	userID := getUserID(r)
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	idempotencyKey, ok := requireMarketplaceIdempotencyKey(w, r, strict)
	if !ok {
		return
	}
	var req marketplacehttp.ClaimClipRequest
	if !s.decodeJSON(w, r, &req, writeMarketplaceError) {
		return
	}
	resp, err := s.marketplace.Handler.ClaimClipHandler(
		r.Context(),
		userID,
		r.PathValue("clip_id"),
		req,
		idempotencyKey,
	)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListClaims(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}
	userID := getUserID(r)
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	resp, err := s.marketplace.Handler.ListClaimsHandler(r.Context(), userID)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDownloadClip(w http.ResponseWriter, r *http.Request) {
	strict := isCanonicalMarketplaceRoute(r)
	if !requireMarketplaceAuthorization(w, r, strict) || !requireMarketplaceRequestID(w, r, strict) {
		return
	}
	userID := getUserID(r)
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	resp, err := s.marketplace.Handler.DownloadClipHandler(
		r.Context(),
		userID,
		r.PathValue("clip_id"),
		strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		resolveClientIP(r),
		r.UserAgent(),
	)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzCheck(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	var req authzhttp.CheckPermissionRequest
	if !s.decodeJSON(w, r, &req, writeAuthzError) {
		return
	}
	userID := resolveAuthzUserID(req.UserID, r)
	resp, err := s.authorization.Handler.CheckPermissionHandler(r.Context(), userID, req)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzCheckBatch(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	var req authzhttp.CheckBatchRequest
	if !s.decodeJSON(w, r, &req, writeAuthzError) {
		return
	}
	userID := resolveAuthzUserID(req.UserID, r)
	resp, err := s.authorization.Handler.CheckBatchHandler(r.Context(), userID, req)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzListUserRoles(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	resp, err := s.authorization.Handler.ListUserRolesHandler(r.Context(), r.PathValue("user_id"))
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzGrantRole(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	userID := r.PathValue("user_id")
	adminID := getUserID(r)
	if strings.TrimSpace(adminID) == "" {
		adminID = r.Header.Get("X-Admin-Id")
	}
	var req authzhttp.GrantRoleRequest
	if !s.decodeJSON(w, r, &req, writeAuthzError) {
		return
	}
	resp, err := s.authorization.Handler.GrantRoleHandler(
		r.Context(),
		userID,
		adminID,
		r.Header.Get("Idempotency-Key"),
		req,
	)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzRevokeRole(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	userID := r.PathValue("user_id")
	adminID := getUserID(r)
	if strings.TrimSpace(adminID) == "" {
		adminID = r.Header.Get("X-Admin-Id")
	}
	var req authzhttp.RevokeRoleRequest
	if !s.decodeJSON(w, r, &req, writeAuthzError) {
		return
	}
	resp, err := s.authorization.Handler.RevokeRoleHandler(
		r.Context(),
		userID,
		adminID,
		r.Header.Get("Idempotency-Key"),
		req,
	)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzCreateDelegation(w http.ResponseWriter, r *http.Request) {
	if !requireAuthzAuthorization(w, r) || !requireAuthzRequestID(w, r) {
		return
	}
	var req authzhttp.CreateDelegationRequest
	if !s.decodeJSON(w, r, &req, writeAuthzError) {
		return
	}
	resp, err := s.authorization.Handler.CreateDelegationHandler(r.Context(), r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminStartImpersonation(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.StartImpersonationRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.StartImpersonationHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminEndImpersonation(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	idempotencyKey, ok := requireAdminIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req superadminhttp.EndImpersonationRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.EndImpersonationHandler(r.Context(), idempotencyKey, req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAdjustWallet(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.WalletAdjustRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.AdjustWalletHandler(r.Context(), adminID, idempotencyKey, r.PathValue("user_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminWalletHistory(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "page_size must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.superAdmin.Handler.WalletHistoryHandler(
		r.Context(),
		r.PathValue("user_id"),
		r.URL.Query().Get("cursor"),
		limit,
	)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminBanUser(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.BanUserRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.BanUserHandler(r.Context(), adminID, idempotencyKey, r.PathValue("user_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminUnbanUser(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.UnbanUserRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.UnbanUserHandler(r.Context(), adminID, idempotencyKey, r.PathValue("user_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminSearchUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	pageSize := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "page_size must be an integer")
			return
		}
		pageSize = parsed
	}
	resp, err := s.superAdmin.Handler.SearchUsersHandler(
		r.Context(),
		r.URL.Query().Get("query"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("cursor"),
		pageSize,
	)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminBulkAction(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.BulkActionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.BulkActionHandler(r.Context(), adminID, idempotencyKey, req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminPauseCampaign(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.PauseCampaignRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.PauseCampaignHandler(r.Context(), adminID, idempotencyKey, r.PathValue("campaign_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAdjustCampaign(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.AdjustCampaignRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.AdjustCampaignHandler(r.Context(), adminID, idempotencyKey, r.PathValue("campaign_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminOverrideSubmission(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.OverrideSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.OverrideSubmissionHandler(r.Context(), adminID, idempotencyKey, r.PathValue("submission_id"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	resp, err := s.superAdmin.Handler.ListFeatureFlagsHandler(r.Context())
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminToggleFeatureFlag(w http.ResponseWriter, r *http.Request) {
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
	var req superadminhttp.ToggleFeatureFlagRequest
	if !s.decodeJSON(w, r, &req, writeSuperAdminError) {
		return
	}
	resp, err := s.superAdmin.Handler.ToggleFeatureFlagHandler(r.Context(), adminID, idempotencyKey, r.PathValue("flag_key"), req)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	query := r.URL.Query()
	startRaw := strings.TrimSpace(query.Get("start"))
	if startRaw == "" {
		startRaw = strings.TrimSpace(query.Get("date_from"))
	}
	endRaw := strings.TrimSpace(query.Get("end"))
	if endRaw == "" {
		endRaw = strings.TrimSpace(query.Get("date_to"))
	}
	start, _, err := parseOptionalRFC3339(startRaw)
	if err != nil {
		writeSuperAdminError(w, http.StatusUnprocessableEntity, "invalid_date_range", "start/date_from must be RFC3339 or YYYY-MM-DD")
		return
	}
	end, _, err := parseOptionalRFC3339(endRaw)
	if err != nil {
		writeSuperAdminError(w, http.StatusUnprocessableEntity, "invalid_date_range", "end/date_to must be RFC3339 or YYYY-MM-DD")
		return
	}
	resp, err := s.superAdmin.Handler.AnalyticsDashboardHandler(r.Context(), start, end)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	pageSize := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "page_size must be an integer")
			return
		}
		pageSize = parsed
	}
	resp, err := s.superAdmin.Handler.AuditLogsHandler(
		r.Context(),
		r.URL.Query().Get("admin_id"),
		r.URL.Query().Get("action_type"),
		r.URL.Query().Get("cursor"),
		pageSize,
	)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAdminAuditLogsExport(w http.ResponseWriter, r *http.Request) {
	if !requireAdminHeaders(w, r) {
		return
	}
	if _, ok := requireAdminID(w, r); !ok {
		return
	}
	query := r.URL.Query()
	format := strings.TrimSpace(query.Get("format"))
	startRaw := strings.TrimSpace(query.Get("start"))
	if startRaw == "" {
		startRaw = strings.TrimSpace(query.Get("date_from"))
	}
	endRaw := strings.TrimSpace(query.Get("end"))
	if endRaw == "" {
		endRaw = strings.TrimSpace(query.Get("date_to"))
	}
	start, _, err := parseOptionalRFC3339(startRaw)
	if err != nil {
		writeSuperAdminError(w, http.StatusUnprocessableEntity, "invalid_date_range", "start/date_from must be RFC3339 or YYYY-MM-DD")
		return
	}
	end, _, err := parseOptionalRFC3339(endRaw)
	if err != nil {
		writeSuperAdminError(w, http.StatusUnprocessableEntity, "invalid_date_range", "end/date_to must be RFC3339 or YYYY-MM-DD")
		return
	}
	includeSignatures := false
	if raw := strings.TrimSpace(query.Get("include_signatures")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			writeSuperAdminError(w, http.StatusBadRequest, "invalid_request", "include_signatures must be boolean")
			return
		}
		includeSignatures = parsed
	}
	resp, err := s.superAdmin.Handler.ExportAuditLogsHandler(r.Context(), format, start, end, includeSignatures)
	if err != nil {
		writeSuperAdminDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductList(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	page := 1
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "invalid_request", "page must be an integer")
			return
		}
		page = parsed
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.product.Handler.ListProductsHandler(r.Context(), producthttp.ListProductsRequest{
		CreatorID:   strings.TrimSpace(r.URL.Query().Get("creator_id")),
		ProductType: strings.TrimSpace(r.URL.Query().Get("type")),
		Visibility:  strings.TrimSpace(r.URL.Query().Get("visibility")),
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductCreate(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	creatorID, ok := requireProductUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req producthttp.CreateProductRequest
	if !s.decodeJSON(w, r, &req, writeProductError) {
		return
	}
	resp, err := s.product.Handler.CreateProductHandler(r.Context(), creatorID, idempotencyKey, req)
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleProductCheckAccess(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		userID = getUserID(r)
	}
	if strings.TrimSpace(userID) == "" {
		writeProductError(w, http.StatusUnauthorized, "missing_user", "user_id query or X-User-Id header is required")
		return
	}
	resp, err := s.product.Handler.CheckAccessHandler(r.Context(), userID, r.PathValue("product_id"))
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	if !resp.Data.HasAccess {
		writeProductError(w, http.StatusPaymentRequired, "payment_required", "You need to purchase this product to access it.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductPurchase(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	userID, ok := requireProductUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.product.Handler.PurchaseProductHandler(r.Context(), userID, idempotencyKey, r.PathValue("id"))
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductFulfill(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	userID, ok := requireProductUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.product.Handler.FulfillProductHandler(r.Context(), userID, idempotencyKey, r.PathValue("id"))
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductAdjustInventory(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	adminID := getAdminID(r)
	if adminID == "" {
		writeProductError(w, http.StatusUnauthorized, "missing_admin", "X-Admin-Id header is required")
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req producthttp.AdjustInventoryRequest
	if !s.decodeJSON(w, r, &req, writeProductError) {
		return
	}
	resp, err := s.product.Handler.AdjustInventoryHandler(r.Context(), adminID, idempotencyKey, r.PathValue("product_id"), req)
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductMediaReorder(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req producthttp.ReorderMediaRequest
	if !s.decodeJSON(w, r, &req, writeProductError) {
		return
	}
	resp, err := s.product.Handler.ReorderMediaHandler(r.Context(), idempotencyKey, r.PathValue("product_id"), req)
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductDiscover(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.product.Handler.DiscoverProductsHandler(r.Context(), limit)
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductSearch(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeProductError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.product.Handler.SearchProductsHandler(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("q")),
		strings.TrimSpace(r.URL.Query().Get("type")),
		limit,
	)
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductUserDataExport(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	resp, err := s.product.Handler.ExportUserDataHandler(r.Context(), r.PathValue("user_id"))
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProductDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if !requireProductAuthorization(w, r) || !requireProductRequestID(w, r) {
		return
	}
	idempotencyKey, ok := requireProductIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.product.Handler.DeleteUserDataHandler(r.Context(), idempotencyKey, r.PathValue("user_id"))
	if err != nil {
		writeProductDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatPostMessage(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.PostMessageRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.PostMessageHandler(r.Context(), userID, userID, idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleChatEditMessage(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.EditMessageRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.EditMessageHandler(r.Context(), userID, chatMessageIDFromPath(r), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatDeleteMessage(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.DeleteMessageRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.DeleteMessageHandler(r.Context(), userID, chatMessageIDFromPath(r), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatListMessages(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	afterSeq := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("after_seq")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "after_seq must be an integer")
			return
		}
		afterSeq = parsed
	}
	resp, err := s.chat.Handler.ListMessagesHandler(
		r.Context(),
		r.PathValue("channel_id"),
		strings.TrimSpace(r.URL.Query().Get("before")),
		afterSeq,
		limit,
	)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatUnreadCount(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	resp, err := s.chat.Handler.UnreadCountHandler(
		r.Context(),
		userID,
		r.PathValue("channel_id"),
		strings.TrimSpace(r.URL.Query().Get("last_read_id")),
	)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatSearch(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.chat.Handler.SearchMessagesHandler(r.Context(), query, strings.TrimSpace(r.URL.Query().Get("channel_id")), limit)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatBackfill(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	channelID := strings.TrimSpace(r.URL.Query().Get("channel_id"))
	if channelID == "" {
		writeChatError(w, http.StatusBadRequest, "invalid_request", "channel_id query parameter is required")
		return
	}
	afterSeq := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("after_seq")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "after_seq must be an integer")
			return
		}
		afterSeq = parsed
	}
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.chat.Handler.ListMessagesHandler(r.Context(), channelID, "", afterSeq, limit)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatPoll(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	channelID := strings.TrimSpace(r.URL.Query().Get("channel_id"))
	if channelID == "" {
		writeChatError(w, http.StatusBadRequest, "invalid_request", "channel_id query parameter is required")
		return
	}
	afterSeq := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("after_seq")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "after_seq must be an integer")
			return
		}
		afterSeq = parsed
	}
	resp, err := s.chat.Handler.ListMessagesHandler(r.Context(), channelID, "", afterSeq, 50)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatAddReaction(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.ReactionRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.AddReactionHandler(r.Context(), userID, r.PathValue("message_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatRemoveReaction(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.chat.Handler.RemoveReactionHandler(
		r.Context(),
		userID,
		r.PathValue("message_id"),
		r.PathValue("emoji"),
		idempotencyKey,
	)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatPinMessage(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.PinMessageRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.PinMessageHandler(r.Context(), userID, r.PathValue("message_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatReportMessage(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.ReportMessageRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.ReportMessageHandler(r.Context(), userID, r.PathValue("message_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatAddAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.AddAttachmentRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.AddAttachmentHandler(r.Context(), userID, r.PathValue("message_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleChatGetAttachment(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	resp, err := s.chat.Handler.GetAttachmentHandler(r.Context(), r.PathValue("message_id"), r.PathValue("attachment_id"))
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatLockThread(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.chat.Handler.LockThreadHandler(r.Context(), userID, r.PathValue("thread_id"), idempotencyKey)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatUpdateModerators(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.UpdateModeratorsRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	resp, err := s.chat.Handler.UpdateModeratorsHandler(r.Context(), userID, r.PathValue("server_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatMuteUser(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	userID, ok := requireChatUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireChatIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req chathttp.MuteUserRequest
	if !s.decodeJSON(w, r, &req, writeChatError) {
		return
	}
	if req.Duration == "" {
		req.Duration = strings.TrimSpace(r.URL.Query().Get("duration"))
	}
	resp, err := s.chat.Handler.MuteUserHandler(r.Context(), userID, r.PathValue("user_id"), idempotencyKey, req)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleChatExport(w http.ResponseWriter, r *http.Request) {
	if !requireChatAuthorization(w, r) || !requireChatRequestID(w, r) {
		return
	}
	limit := 1000
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeChatError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
			return
		}
		limit = parsed
	}
	resp, err := s.chat.Handler.ExportMessagesHandler(
		r.Context(),
		strings.TrimSpace(r.URL.Query().Get("server_id")),
		strings.TrimSpace(r.URL.Query().Get("channel_id")),
		limit,
	)
	if err != nil {
		writeChatDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func chatMessageIDFromPath(r *http.Request) string {
	if messageID := strings.TrimSpace(r.PathValue("message_id")); messageID != "" {
		return messageID
	}
	return strings.TrimSpace(r.PathValue("id"))
}

func (s *Server) handleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	requestID := getRequestID(r)
	if requestID == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.CreateCampaignRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	resp, err := s.campaign.Handler.CreateCampaignHandler(r.Context(), userID, requestID, req)
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleCampaignList(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		userID = strings.TrimSpace(r.URL.Query().Get("brand_id"))
	}
	resp, err := s.campaign.Handler.ListCampaignsHandler(r.Context(), userID, r.URL.Query().Get("status"))
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.campaign.Handler.GetCampaignHandler(r.Context(), r.PathValue("campaign_id"))
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignUpdate(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	if getRequestID(r) == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.UpdateCampaignRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.UpdateCampaignHandler(r.Context(), userID, r.PathValue("campaign_id"), req); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleCampaignLaunch(w http.ResponseWriter, r *http.Request) {
	s.handleCampaignStatus(w, r, func(ctx context.Context, userID string, campaignID string, reason string) error {
		return s.campaign.Handler.LaunchCampaignHandler(ctx, userID, campaignID, reason)
	})
}

func (s *Server) handleCampaignPause(w http.ResponseWriter, r *http.Request) {
	s.handleCampaignStatus(w, r, func(ctx context.Context, userID string, campaignID string, reason string) error {
		return s.campaign.Handler.PauseCampaignHandler(ctx, userID, campaignID, reason)
	})
}

func (s *Server) handleCampaignResume(w http.ResponseWriter, r *http.Request) {
	s.handleCampaignStatus(w, r, func(ctx context.Context, userID string, campaignID string, reason string) error {
		return s.campaign.Handler.ResumeCampaignHandler(ctx, userID, campaignID, reason)
	})
}

func (s *Server) handleCampaignComplete(w http.ResponseWriter, r *http.Request) {
	s.handleCampaignStatus(w, r, func(ctx context.Context, userID string, campaignID string, reason string) error {
		return s.campaign.Handler.CompleteCampaignHandler(ctx, userID, campaignID, reason)
	})
}

func (s *Server) handleCampaignStatus(
	w http.ResponseWriter,
	r *http.Request,
	fn func(context.Context, string, string, string) error,
) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	if getRequestID(r) == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.StatusActionRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := fn(r.Context(), userID, r.PathValue("campaign_id"), req.Reason); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCampaignMediaUploadURL(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	if getRequestID(r) == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.GenerateUploadURLRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	resp, err := s.campaign.Handler.GenerateUploadURLHandler(r.Context(), userID, r.PathValue("campaign_id"), req)
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignMediaConfirm(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	if getRequestID(r) == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.ConfirmMediaRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.ConfirmMediaHandler(
		r.Context(),
		userID,
		r.PathValue("campaign_id"),
		r.PathValue("media_id"),
		req,
	); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "confirmed"})
}

func (s *Server) handleCampaignMediaList(w http.ResponseWriter, r *http.Request) {
	resp, err := s.campaign.Handler.ListMediaHandler(r.Context(), r.PathValue("campaign_id"))
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignAnalytics(w http.ResponseWriter, r *http.Request) {
	resp, err := s.campaign.Handler.GetAnalyticsHandler(r.Context(), r.PathValue("campaign_id"))
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignAnalyticsExport(w http.ResponseWriter, r *http.Request) {
	resp, err := s.campaign.Handler.ExportAnalyticsHandler(r.Context(), r.PathValue("campaign_id"))
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignIncreaseBudget(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	if getRequestID(r) == "" {
		writeCampaignError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return
	}
	var req campaignhttp.IncreaseBudgetRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.IncreaseBudgetHandler(r.Context(), userID, r.PathValue("campaign_id"), req); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "budget_updated"})
}

func (s *Server) handleSubmissionCreate(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubmissionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req submissionhttp.CreateSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	if req.IdempotencyKey != "" && strings.TrimSpace(req.IdempotencyKey) != idempotencyKey {
		writeSubmissionError(w, http.StatusBadRequest, "idempotency_mismatch", "body idempotency_key must match Idempotency-Key header")
		return
	}
	req.IdempotencyKey = idempotencyKey

	resp, err := s.submission.Handler.CreateSubmissionHandler(r.Context(), userID, req)
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleSubmissionGet(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}

	resp, err := s.submission.Handler.GetSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionList(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()
	resp, err := s.submission.Handler.ListSubmissionsHandler(
		r.Context(),
		userID,
		query.Get("campaign_id"),
		query.Get("status"),
	)
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionApprove(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubmissionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req submissionhttp.ApproveSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	req.IdempotencyKey = idempotencyKey
	if err := s.submission.Handler.ApproveSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	item, err := s.submission.Handler.GetSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"submission_id":    item.Submission.SubmissionID,
		"status":           item.Submission.Status,
		"verification_end": item.Submission.VerificationEnd,
	})
}

func (s *Server) handleSubmissionReject(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubmissionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req submissionhttp.RejectSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	req.IdempotencyKey = idempotencyKey
	if err := s.submission.Handler.RejectSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	item, err := s.submission.Handler.GetSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"submission_id": item.Submission.SubmissionID,
		"status":        item.Submission.Status,
	})
}

func (s *Server) handleSubmissionReport(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubmissionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req submissionhttp.ReportSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	req.IdempotencyKey = idempotencyKey
	if err := s.submission.Handler.ReportSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	item, err := s.submission.Handler.GetSubmissionHandler(r.Context(), userID, r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"submission_id": item.Submission.SubmissionID,
		"status":        item.Submission.Status,
	})
}

// handleSubmissionBulkOperation godoc
// @Summary Execute bulk submission operation
// @Description Approves or rejects multiple submissions in a single request.
// @Tags submission-service
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-User-Id header string true "Actor user id"
// @Param request body submissionhttp.BulkOperationRequest true "Bulk operation payload"
// @Success 200 {object} submissionhttp.BulkOperationResponse
// @Failure 400 {object} submissionhttp.ErrorResponse
// @Failure 403 {object} submissionhttp.ErrorResponse
// @Failure 404 {object} submissionhttp.ErrorResponse
// @Failure 409 {object} submissionhttp.ErrorResponse
// @Failure 500 {object} submissionhttp.ErrorResponse
// @Router /submissions/bulk-operations [post]
func (s *Server) handleSubmissionBulkOperation(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubmissionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req submissionhttp.BulkOperationRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	req.IdempotencyKey = idempotencyKey
	if req.ReasonCode == "" {
		req.ReasonCode = strings.TrimSpace(req.Reason)
	}
	resp, err := s.submission.Handler.BulkOperationHandler(r.Context(), userID, req)
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionAnalytics(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	resp, err := s.submission.Handler.AnalyticsHandler(r.Context(), userID, r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionCreatorDashboard(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	userID, ok := requireSubmissionUser(w, r)
	if !ok {
		return
	}
	resp, err := s.submission.Handler.CreatorDashboardHandler(r.Context(), userID)
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionBrandDashboard(w http.ResponseWriter, r *http.Request) {
	if !requireSubmissionAuthorization(w, r) || !requireSubmissionRequestID(w, r) {
		return
	}
	if _, ok := requireSubmissionUser(w, r); !ok {
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("campaign_id")) == "" {
		writeSubmissionError(w, http.StatusBadRequest, "invalid_request", "campaign_id query parameter is required")
		return
	}
	resp, err := s.submission.Handler.BrandDashboardHandler(r.Context(), r.URL.Query().Get("campaign_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionAddOverlay(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	s.logger.Info("distribution add overlay request received",
		"event", "distribution_http_add_overlay_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	var req distributionhttp.AddOverlayRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.AddOverlayHandler(r.Context(), itemID, req); err != nil {
		s.logger.Warn("distribution add overlay request failed",
			"event", "distribution_http_add_overlay_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution add overlay request succeeded",
		"event", "distribution_http_add_overlay_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "overlay_added"})
}

func (s *Server) handleDistributionPreview(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	s.logger.Info("distribution preview request received",
		"event", "distribution_http_preview_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	resp, err := s.distribution.Handler.PreviewHandler(r.Context(), itemID)
	if err != nil {
		s.logger.Warn("distribution preview request failed",
			"event", "distribution_http_preview_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution preview request succeeded",
		"event", "distribution_http_preview_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionSchedule(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	userID := getUserID(r)
	if userID == "" {
		writeDistributionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	s.logger.Info("distribution schedule request received",
		"event", "distribution_http_schedule_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	var req distributionhttp.ScheduleRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.ScheduleHandler(r.Context(), userID, itemID, req); err != nil {
		s.logger.Warn("distribution schedule request failed",
			"event", "distribution_http_schedule_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"influencer_id", userID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution schedule request succeeded",
		"event", "distribution_http_schedule_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled"})
}

func (s *Server) handleDistributionReschedule(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	userID := getUserID(r)
	if userID == "" {
		writeDistributionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	s.logger.Info("distribution reschedule request received",
		"event", "distribution_http_reschedule_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	var req distributionhttp.ScheduleRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.RescheduleHandler(r.Context(), userID, itemID, req); err != nil {
		s.logger.Warn("distribution reschedule request failed",
			"event", "distribution_http_reschedule_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"influencer_id", userID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution reschedule request succeeded",
		"event", "distribution_http_reschedule_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "rescheduled"})
}

func (s *Server) handleDistributionPublish(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	userID := getUserID(r)
	if userID == "" {
		writeDistributionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	s.logger.Info("distribution publish request received",
		"event", "distribution_http_publish_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	var req distributionhttp.PublishRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.PublishHandler(r.Context(), userID, itemID, req); err != nil {
		s.logger.Warn("distribution publish request failed",
			"event", "distribution_http_publish_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"influencer_id", userID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution publish request succeeded",
		"event", "distribution_http_publish_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	writeJSON(w, http.StatusCreated, map[string]string{"status": "published"})
}

func (s *Server) handleDistributionDownload(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	s.logger.Info("distribution download request received",
		"event", "distribution_http_download_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	var req distributionhttp.DownloadRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	resp, err := s.distribution.Handler.DownloadHandler(r.Context(), itemID, req)
	if err != nil {
		s.logger.Warn("distribution download request failed",
			"event", "distribution_http_download_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution download request succeeded",
		"event", "distribution_http_download_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionPublishMulti(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	userID := getUserID(r)
	if userID == "" {
		writeDistributionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	s.logger.Info("distribution publish multi request received",
		"event", "distribution_http_publish_multi_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	var req distributionhttp.PublishMultiRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.PublishMultiHandler(r.Context(), userID, itemID, req); err != nil {
		s.logger.Warn("distribution publish multi request failed",
			"event", "distribution_http_publish_multi_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"influencer_id", userID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution publish multi request succeeded",
		"event", "distribution_http_publish_multi_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) handleDistributionRetry(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	itemID := r.PathValue("id")
	userID := getUserID(r)
	if userID == "" {
		writeDistributionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	s.logger.Info("distribution retry request received",
		"event", "distribution_http_retry_request_received",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	if err := s.distribution.Handler.RetryHandler(r.Context(), userID, itemID); err != nil {
		s.logger.Warn("distribution retry request failed",
			"event", "distribution_http_retry_request_failed",
			"module", "campaign-editorial/distribution-service",
			"layer", "platform",
			"request_id", requestID,
			"item_id", itemID,
			"influencer_id", userID,
			"error", err.Error(),
		)
		writeDistributionDomainError(w, err)
		return
	}
	s.logger.Info("distribution retry request succeeded",
		"event", "distribution_http_retry_request_succeeded",
		"module", "campaign-editorial/distribution-service",
		"layer", "platform",
		"request_id", requestID,
		"item_id", itemID,
		"influencer_id", userID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "retrying"})
}

func (s *Server) handleVotingCreate(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	userID := getUserID(r)
	if strings.TrimSpace(userID) == "" {
		writeVotingError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeVotingError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return
	}

	s.logger.Info("voting create request received",
		"event", "voting_http_create_request_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "platform",
		"request_id", requestID,
		"user_id", userID,
	)

	var req votinghttp.CreateVoteRequest
	if !s.decodeJSON(w, r, &req, writeVotingError) {
		return
	}
	resp, err := s.voting.Handler.CreateVoteHandler(
		r.Context(),
		userID,
		idempotencyKey,
		req,
		resolveClientIP(r),
		strings.TrimSpace(r.UserAgent()),
	)
	if err != nil {
		s.logger.Warn("voting create request failed",
			"event", "voting_http_create_request_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"user_id", userID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	s.logger.Info("voting create request succeeded",
		"event", "voting_http_create_request_succeeded",
		"module", "campaign-editorial/voting-engine",
		"layer", "platform",
		"request_id", requestID,
		"user_id", userID,
		"vote_id", resp.VoteID,
		"submission_id", resp.SubmissionID,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRetract(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	userID := getUserID(r)
	if strings.TrimSpace(userID) == "" {
		writeVotingError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeVotingError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return
	}
	voteID := r.PathValue("vote_id")
	s.logger.Info("voting retract request received",
		"event", "voting_http_retract_request_received",
		"module", "campaign-editorial/voting-engine",
		"layer", "platform",
		"request_id", requestID,
		"user_id", userID,
		"vote_id", voteID,
	)
	if err := s.voting.Handler.RetractVoteHandler(r.Context(), voteID, userID, idempotencyKey); err != nil {
		s.logger.Warn("voting retract request failed",
			"event", "voting_http_retract_request_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"user_id", userID,
			"vote_id", voteID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	s.logger.Info("voting retract request succeeded",
		"event", "voting_http_retract_request_succeeded",
		"module", "campaign-editorial/voting-engine",
		"layer", "platform",
		"request_id", requestID,
		"user_id", userID,
		"vote_id", voteID,
	)
	writeJSON(w, http.StatusOK, map[string]string{"status": "retracted"})
}

func (s *Server) handleVotingSubmissionVotes(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	submissionID := r.PathValue("submission_id")
	resp, err := s.voting.Handler.SubmissionVotesHandler(r.Context(), submissionID)
	if err != nil {
		s.logger.Warn("voting submission score request failed",
			"event", "voting_http_submission_scores_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"submission_id", submissionID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingLegacyLeaderboard(w http.ResponseWriter, r *http.Request) {
	if campaignID := strings.TrimSpace(r.URL.Query().Get("campaign_id")); campaignID != "" {
		resp, err := s.voting.Handler.CampaignLeaderboardHandler(r.Context(), campaignID)
		if err != nil {
			writeVotingDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp, err := s.voting.Handler.TrendingLeaderboardHandler(r.Context())
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingCampaignLeaderboard(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	campaignID := r.PathValue("campaign_id")
	resp, err := s.voting.Handler.CampaignLeaderboardHandler(r.Context(), campaignID)
	if err != nil {
		s.logger.Warn("voting campaign leaderboard request failed",
			"event", "voting_http_campaign_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"campaign_id", campaignID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRoundLeaderboard(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	roundID := r.PathValue("round_id")
	resp, err := s.voting.Handler.RoundLeaderboardHandler(r.Context(), roundID)
	if err != nil {
		s.logger.Warn("voting round leaderboard request failed",
			"event", "voting_http_round_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"round_id", roundID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingTrendingLeaderboard(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	resp, err := s.voting.Handler.TrendingLeaderboardHandler(r.Context())
	if err != nil {
		s.logger.Warn("voting trending leaderboard request failed",
			"event", "voting_http_trending_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingCreatorLeaderboard(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	userID := r.PathValue("user_id")
	resp, err := s.voting.Handler.CreatorLeaderboardHandler(r.Context(), userID)
	if err != nil {
		s.logger.Warn("voting creator leaderboard request failed",
			"event", "voting_http_creator_leaderboard_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"user_id", userID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRoundResults(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	roundID := r.PathValue("round_id")
	resp, err := s.voting.Handler.RoundResultsHandler(r.Context(), roundID)
	if err != nil {
		s.logger.Warn("voting round results request failed",
			"event", "voting_http_round_results_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"round_id", roundID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingAnalytics(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	resp, err := s.voting.Handler.VoteAnalyticsHandler(r.Context())
	if err != nil {
		s.logger.Warn("voting analytics request failed",
			"event", "voting_http_analytics_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingQuarantineAction(w http.ResponseWriter, r *http.Request) {
	requestID := getRequestID(r)
	userID := getUserID(r)
	if strings.TrimSpace(userID) == "" {
		writeVotingError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeVotingError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return
	}
	var req votinghttp.QuarantineActionRequest
	if !s.decodeJSON(w, r, &req, writeVotingError) {
		return
	}
	quarantineID := r.PathValue("quarantine_id")
	if err := s.voting.Handler.QuarantineActionHandler(
		r.Context(),
		quarantineID,
		req.Action,
		userID,
		idempotencyKey,
	); err != nil {
		s.logger.Warn("voting quarantine action request failed",
			"event", "voting_http_quarantine_action_failed",
			"module", "campaign-editorial/voting-engine",
			"layer", "platform",
			"request_id", requestID,
			"user_id", userID,
			"quarantine_id", quarantineID,
			"error", err.Error(),
		)
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "processed"})
}
