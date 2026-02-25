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

	campaignservice "solomon/contexts/campaign-editorial/campaign-service"
	campaignerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	campaignhttp "solomon/contexts/campaign-editorial/campaign-service/transport/http"
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
	authorization "solomon/contexts/identity-access/authorization-service"
	authzerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	authzhttp "solomon/contexts/identity-access/authorization-service/transport/http"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "solomon/internal/platform/httpserver/docs"
)

type Server struct {
	mux           *http.ServeMux
	logger        *slog.Logger
	addr          string
	marketplace   contentlibrarymarketplace.Module
	authorization authorization.Module
	campaign      campaignservice.Module
	submission    submissionservice.Module
	distribution  distributionservice.Module
	voting        votingengine.Module
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
		mux:           http.NewServeMux(),
		logger:        logger,
		addr:          addr,
		marketplace:   marketplace,
		authorization: authorizationModule,
		campaign:      campaignModule,
		submission:    submissionModule,
		distribution:  distributionModule,
		voting:        votingModule,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Start() error {
	s.logger.Info("http server starting",
		"event", "http_server_starting",
		"module", "internal/platform/httpserver",
		"layer", "platform",
		"addr", s.addr,
	)
	return http.ListenAndServe(s.addr, s.mux)
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
	s.mux.HandleFunc("POST /v1/marketplace/clips/{clip_id}/claim", s.handleClaimClip)
	s.mux.HandleFunc("GET /v1/marketplace/claims", s.handleListClaims)

	// M21
	s.mux.HandleFunc("POST /api/authz/v1/check", s.handleAuthzCheck)
	s.mux.HandleFunc("POST /api/authz/v1/check-batch", s.handleAuthzCheckBatch)
	s.mux.HandleFunc("GET /api/authz/v1/users/{user_id}/roles", s.handleAuthzListUserRoles)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/grant", s.handleAuthzGrantRole)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/revoke", s.handleAuthzRevokeRole)
	s.mux.HandleFunc("POST /api/authz/v1/delegations", s.handleAuthzCreateDelegation)

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

	// M31
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/overlays", s.handleDistributionAddOverlay)
	s.mux.HandleFunc("GET /api/v1/distribution/items/{id}/preview", s.handleDistributionPreview)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/schedule", s.handleDistributionSchedule)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/reschedule", s.handleDistributionReschedule)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/download", s.handleDistributionDownload)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/publish-multi", s.handleDistributionPublishMulti)
	s.mux.HandleFunc("POST /api/v1/distribution/items/{id}/retry", s.handleDistributionRetry)

	// M08
	s.mux.HandleFunc("POST /v1/votes", s.handleVotingCreate)
	s.mux.HandleFunc("DELETE /v1/votes/{vote_id}", s.handleVotingRetract)
	s.mux.HandleFunc("GET /v1/votes/submissions/{submission_id}", s.handleVotingSubmissionVotes)
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

func writeCampaignDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, campaignerrors.ErrCampaignNotFound):
		writeCampaignError(w, http.StatusNotFound, "campaign_not_found", err.Error())
	case errors.Is(err, campaignerrors.ErrInvalidCampaignInput),
		errors.Is(err, campaignerrors.ErrIdempotencyKeyRequired):
		writeCampaignError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, campaignerrors.ErrCampaignNotEditable),
		errors.Is(err, campaignerrors.ErrInvalidStateTransition),
		errors.Is(err, campaignerrors.ErrInvalidBudgetIncrease),
		errors.Is(err, campaignerrors.ErrMediaAlreadyConfirmed):
		writeCampaignError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, campaignerrors.ErrMediaNotFound):
		writeCampaignError(w, http.StatusNotFound, "media_not_found", err.Error())
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
	case errors.Is(err, submissionerrors.ErrDuplicateSubmission):
		writeSubmissionError(w, http.StatusConflict, "duplicate_submission", err.Error())
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
	case errors.Is(err, distributionerrors.ErrInvalidDistributionInput),
		errors.Is(err, distributionerrors.ErrInvalidScheduleWindow):
		writeDistributionError(w, http.StatusBadRequest, "invalid_request", err.Error())
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
	case errors.Is(err, votingerrors.ErrInvalidVoteInput):
		writeVotingError(w, http.StatusBadRequest, "invalid_vote_input", err.Error())
	case errors.Is(err, votingerrors.ErrAlreadyRetracted):
		writeVotingError(w, http.StatusConflict, "already_retracted", err.Error())
	case errors.Is(err, votingerrors.ErrConflict):
		writeVotingError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, votingerrors.ErrSubmissionNotFound):
		writeVotingError(w, http.StatusNotFound, "submission_not_found", err.Error())
	default:
		writeVotingError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (s *Server) handleListClips(w http.ResponseWriter, r *http.Request) {
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
	resp, err := s.marketplace.Handler.GetClipHandler(r.Context(), r.PathValue("clip_id"))
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetClipPreview(w http.ResponseWriter, r *http.Request) {
	resp, err := s.marketplace.Handler.GetClipPreviewHandler(r.Context(), r.PathValue("clip_id"))
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClaimClip(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
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
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleListClaims(w http.ResponseWriter, r *http.Request) {
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
	userID := getUserID(r)
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	resp, err := s.marketplace.Handler.DownloadClipHandler(
		r.Context(),
		userID,
		r.PathValue("clip_id"),
		r.Header.Get("Idempotency-Key"),
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
	resp, err := s.authorization.Handler.ListUserRolesHandler(r.Context(), r.PathValue("user_id"))
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzGrantRole(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeCampaignError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}
	var req campaignhttp.CreateCampaignRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	resp, err := s.campaign.Handler.CreateCampaignHandler(r.Context(), userID, r.Header.Get("Idempotency-Key"), req)
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
	var req campaignhttp.UpdateCampaignRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.UpdateCampaignHandler(r.Context(), getUserID(r), r.PathValue("campaign_id"), req); err != nil {
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
	var req campaignhttp.StatusActionRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := fn(r.Context(), getUserID(r), r.PathValue("campaign_id"), req.Reason); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCampaignMediaUploadURL(w http.ResponseWriter, r *http.Request) {
	var req campaignhttp.GenerateUploadURLRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	resp, err := s.campaign.Handler.GenerateUploadURLHandler(r.Context(), getUserID(r), r.PathValue("campaign_id"), req)
	if err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCampaignMediaConfirm(w http.ResponseWriter, r *http.Request) {
	var req campaignhttp.ConfirmMediaRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.ConfirmMediaHandler(
		r.Context(),
		getUserID(r),
		r.PathValue("campaign_id"),
		r.PathValue("media_id"),
		req,
	); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
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
	var req campaignhttp.IncreaseBudgetRequest
	if !s.decodeJSON(w, r, &req, writeCampaignError) {
		return
	}
	if err := s.campaign.Handler.IncreaseBudgetHandler(r.Context(), getUserID(r), r.PathValue("campaign_id"), req); err != nil {
		writeCampaignDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "budget_updated"})
}

func (s *Server) handleSubmissionCreate(w http.ResponseWriter, r *http.Request) {
	var req submissionhttp.CreateSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	resp, err := s.submission.Handler.CreateSubmissionHandler(r.Context(), getUserID(r), req)
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleSubmissionGet(w http.ResponseWriter, r *http.Request) {
	resp, err := s.submission.Handler.GetSubmissionHandler(r.Context(), r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	resp, err := s.submission.Handler.ListSubmissionsHandler(
		r.Context(),
		getUserID(r),
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
	var req submissionhttp.ApproveSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	if err := s.submission.Handler.ApproveSubmissionHandler(r.Context(), getUserID(r), r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (s *Server) handleSubmissionReject(w http.ResponseWriter, r *http.Request) {
	var req submissionhttp.RejectSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	if err := s.submission.Handler.RejectSubmissionHandler(r.Context(), getUserID(r), r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *Server) handleSubmissionReport(w http.ResponseWriter, r *http.Request) {
	var req submissionhttp.ReportSubmissionRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	if err := s.submission.Handler.ReportSubmissionHandler(r.Context(), getUserID(r), r.PathValue("submission_id"), req); err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reported"})
}

func (s *Server) handleSubmissionBulkOperation(w http.ResponseWriter, r *http.Request) {
	var req submissionhttp.BulkOperationRequest
	if !s.decodeJSON(w, r, &req, writeSubmissionError) {
		return
	}
	processed := 0
	actorID := getUserID(r)
	for _, submissionID := range req.SubmissionIDs {
		switch req.OperationType {
		case "bulk_approve":
			_ = s.submission.Handler.ApproveSubmissionHandler(r.Context(), actorID, submissionID, submissionhttp.ApproveSubmissionRequest{Reason: req.Reason})
		case "bulk_reject":
			_ = s.submission.Handler.RejectSubmissionHandler(r.Context(), actorID, submissionID, submissionhttp.RejectSubmissionRequest{Reason: req.Reason})
		default:
			writeSubmissionError(w, http.StatusBadRequest, "invalid_operation", "operation_type must be bulk_approve or bulk_reject")
			return
		}
		processed++
	}
	writeJSON(w, http.StatusOK, map[string]int{"processed": processed})
}

func (s *Server) handleSubmissionAnalytics(w http.ResponseWriter, r *http.Request) {
	resp, err := s.submission.Handler.AnalyticsHandler(r.Context(), r.PathValue("submission_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionCreatorDashboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.submission.Handler.CreatorDashboardHandler(r.Context(), getUserID(r))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmissionBrandDashboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.submission.Handler.BrandDashboardHandler(r.Context(), r.URL.Query().Get("campaign_id"))
	if err != nil {
		writeSubmissionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionAddOverlay(w http.ResponseWriter, r *http.Request) {
	var req distributionhttp.AddOverlayRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.AddOverlayHandler(r.Context(), r.PathValue("id"), req); err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "overlay_added"})
}

func (s *Server) handleDistributionPreview(w http.ResponseWriter, r *http.Request) {
	resp, err := s.distribution.Handler.PreviewHandler(r.Context(), r.PathValue("id"))
	if err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionSchedule(w http.ResponseWriter, r *http.Request) {
	var req distributionhttp.ScheduleRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.ScheduleHandler(r.Context(), getUserID(r), r.PathValue("id"), req); err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "scheduled"})
}

func (s *Server) handleDistributionReschedule(w http.ResponseWriter, r *http.Request) {
	var req distributionhttp.ScheduleRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.RescheduleHandler(r.Context(), getUserID(r), r.PathValue("id"), req); err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rescheduled"})
}

func (s *Server) handleDistributionDownload(w http.ResponseWriter, r *http.Request) {
	var req distributionhttp.DownloadRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	resp, err := s.distribution.Handler.DownloadHandler(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDistributionPublishMulti(w http.ResponseWriter, r *http.Request) {
	var req distributionhttp.PublishMultiRequest
	if !s.decodeJSON(w, r, &req, writeDistributionError) {
		return
	}
	if err := s.distribution.Handler.PublishMultiHandler(r.Context(), getUserID(r), r.PathValue("id"), req); err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) handleDistributionRetry(w http.ResponseWriter, r *http.Request) {
	if err := s.distribution.Handler.RetryHandler(r.Context(), getUserID(r), r.PathValue("id")); err != nil {
		writeDistributionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "retrying"})
}

func (s *Server) handleVotingCreate(w http.ResponseWriter, r *http.Request) {
	var req votinghttp.CreateVoteRequest
	if !s.decodeJSON(w, r, &req, writeVotingError) {
		return
	}
	resp, err := s.voting.Handler.CreateVoteHandler(r.Context(), getUserID(r), r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRetract(w http.ResponseWriter, r *http.Request) {
	if err := s.voting.Handler.RetractVoteHandler(r.Context(), r.PathValue("vote_id"), getUserID(r)); err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "retracted"})
}

func (s *Server) handleVotingSubmissionVotes(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.SubmissionVotesHandler(r.Context(), r.PathValue("submission_id"))
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingCampaignLeaderboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.CampaignLeaderboardHandler(r.Context(), r.PathValue("campaign_id"))
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRoundLeaderboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.CampaignLeaderboardHandler(r.Context(), r.PathValue("round_id"))
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingTrendingLeaderboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.TrendingLeaderboardHandler(r.Context())
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingCreatorLeaderboard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.CreatorLeaderboardHandler(r.Context(), r.PathValue("user_id"))
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingRoundResults(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.RoundResultsHandler(r.Context(), r.PathValue("round_id"))
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingAnalytics(w http.ResponseWriter, r *http.Request) {
	resp, err := s.voting.Handler.VoteAnalyticsHandler(r.Context())
	if err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVotingQuarantineAction(w http.ResponseWriter, r *http.Request) {
	var payload map[string]string
	if !s.decodeJSON(w, r, &payload, writeVotingError) {
		return
	}
	if err := s.voting.Handler.QuarantineActionHandler(r.Context(), r.PathValue("quarantine_id"), payload["action"]); err != nil {
		writeVotingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "processed"})
}
