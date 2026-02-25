package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	contentlibrarymarketplace "solomon/contexts/campaign-editorial/content-library-marketplace"
	marketplacedomainerrors "solomon/contexts/campaign-editorial/content-library-marketplace/domain/errors"
	marketplacehttp "solomon/contexts/campaign-editorial/content-library-marketplace/transport/http"
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
}

func New(
	marketplace contentlibrarymarketplace.Module,
	authorizationModule authorization.Module,
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
	s.mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

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

	s.mux.HandleFunc("POST /api/authz/v1/check", s.handleAuthzCheck)
	s.mux.HandleFunc("POST /api/authz/v1/check-batch", s.handleAuthzCheckBatch)
	s.mux.HandleFunc("GET /api/authz/v1/users/{user_id}/roles", s.handleAuthzListUserRoles)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/grant", s.handleAuthzGrantRole)
	s.mux.HandleFunc("POST /api/authz/v1/users/{user_id}/roles/revoke", s.handleAuthzRevokeRole)
	s.mux.HandleFunc("POST /api/authz/v1/delegations", s.handleAuthzCreateDelegation)
}

func (s *Server) handleListClips(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query() // gorm-postgres-enforcer: allow-raw-sql parses HTTP query parameters only
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
	clipID := r.PathValue("clip_id")
	resp, err := s.marketplace.Handler.GetClipHandler(r.Context(), clipID)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetClipPreview(w http.ResponseWriter, r *http.Request) {
	clipID := r.PathValue("clip_id")
	resp, err := s.marketplace.Handler.GetClipPreviewHandler(r.Context(), clipID)
	if err != nil {
		writeMarketplaceDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClaimClip(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}

	var req marketplacehttp.ClaimClipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeMarketplaceError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	clipID := r.PathValue("clip_id")
	idempotencyKey := r.Header.Get("Idempotency-Key")

	resp, err := s.marketplace.Handler.ClaimClipHandler(
		r.Context(),
		userID,
		clipID,
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
	userID := r.Header.Get("X-User-Id")
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
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		writeMarketplaceError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return
	}

	clipID := r.PathValue("clip_id")
	idempotencyKey := r.Header.Get("Idempotency-Key")
	resp, err := s.marketplace.Handler.DownloadClipHandler(
		r.Context(),
		userID,
		clipID,
		idempotencyKey,
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthzError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthzError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
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
	userID := r.PathValue("user_id")
	resp, err := s.authorization.Handler.ListUserRolesHandler(r.Context(), userID)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAuthzGrantRole(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	adminID := r.Header.Get("X-User-Id")
	if strings.TrimSpace(adminID) == "" {
		adminID = r.Header.Get("X-Admin-Id")
	}

	var req authzhttp.GrantRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthzError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
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
	adminID := r.Header.Get("X-User-Id")
	if strings.TrimSpace(adminID) == "" {
		adminID = r.Header.Get("X-Admin-Id")
	}

	var req authzhttp.RevokeRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthzError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthzError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	resp, err := s.authorization.Handler.CreateDelegationHandler(
		r.Context(),
		r.Header.Get("Idempotency-Key"),
		req,
	)
	if err != nil {
		writeAuthzDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
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

func writeMarketplaceError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, marketplacehttp.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func writeAuthzError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, authzhttp.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
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
