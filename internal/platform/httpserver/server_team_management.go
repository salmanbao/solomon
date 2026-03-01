package httpserver

import (
	"errors"
	"net/http"
	"strings"

	teamerrors "solomon/contexts/internal-ops/team-management-service/domain/errors"
	teamhttp "solomon/contexts/internal-ops/team-management-service/transport/http"
)

func writeTeamError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, teamhttp.ErrorResponse{Code: code, Message: message})
}

func writeTeamDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, teamerrors.ErrTeamNotFound),
		errors.Is(err, teamerrors.ErrMemberNotFound),
		errors.Is(err, teamerrors.ErrInviteNotFound),
		errors.Is(err, teamerrors.ErrNotFound):
		writeTeamError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, teamerrors.ErrInvalidRequest),
		errors.Is(err, teamerrors.ErrIdempotencyKeyRequired):
		writeTeamError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, teamerrors.ErrForbidden):
		writeTeamError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, teamerrors.ErrMFARequired):
		writeTeamError(w, http.StatusUnauthorized, "mfa_required", err.Error())
	case errors.Is(err, teamerrors.ErrConflict),
		errors.Is(err, teamerrors.ErrIdempotencyConflict),
		errors.Is(err, teamerrors.ErrOwnerTransferRequired):
		writeTeamError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, teamerrors.ErrInviteExpired):
		writeTeamError(w, http.StatusGone, "invite_expired", err.Error())
	case errors.Is(err, teamerrors.ErrDependencyUnavailable):
		writeTeamError(w, http.StatusFailedDependency, "dependency_unavailable", err.Error())
	default:
		writeTeamError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireTeamAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeTeamError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireTeamRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeTeamError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireTeamUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeTeamError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireTeamIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeTeamError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
}

func (s *Server) handleTeamCreate(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireTeamIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req teamhttp.CreateTeamRequest
	if !s.decodeJSON(w, r, &req, writeTeamError) {
		return
	}
	resp, err := s.teamManagement.Handler.CreateTeamHandler(r.Context(), idempotencyKey, actorUserID, req)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleTeamCreateInvite(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireTeamIdempotencyKey(w, r)
	if !ok {
		return
	}
	teamID := r.PathValue("teamId")
	if strings.TrimSpace(teamID) == "" {
		writeTeamError(w, http.StatusBadRequest, "invalid_request", "teamId is required")
		return
	}

	var req teamhttp.CreateInviteRequest
	if !s.decodeJSON(w, r, &req, writeTeamError) {
		return
	}
	resp, err := s.teamManagement.Handler.CreateInviteHandler(r.Context(), idempotencyKey, actorUserID, teamID, req)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (s *Server) handleTeamCreateInviteV1(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("teamId", r.PathValue("team_id"))
	s.handleTeamCreateInvite(w, r)
}

func (s *Server) handleTeamAcceptInvite(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireTeamIdempotencyKey(w, r)
	if !ok {
		return
	}

	token := r.PathValue("token")
	if strings.TrimSpace(token) == "" {
		writeTeamError(w, http.StatusBadRequest, "invalid_request", "invite token is required")
		return
	}
	resp, err := s.teamManagement.Handler.AcceptInviteHandler(r.Context(), idempotencyKey, actorUserID, token)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamAcceptInviteV1(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("token", r.PathValue("invite_id"))
	s.handleTeamAcceptInvite(w, r)
}

func (s *Server) handleTeamUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireTeamIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req teamhttp.UpdateMemberRoleRequest
	if !s.decodeJSON(w, r, &req, writeTeamError) {
		return
	}
	resp, err := s.teamManagement.Handler.UpdateMemberRoleHandler(
		r.Context(),
		idempotencyKey,
		actorUserID,
		r.PathValue("teamId"),
		r.PathValue("memberId"),
		strings.TrimSpace(r.Header.Get("X-MFA-Code")),
		req,
	)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamUpdateMemberRoleV1(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("teamId", r.PathValue("team_id"))
	r.SetPathValue("memberId", r.PathValue("member_id"))
	s.handleTeamUpdateMemberRole(w, r)
}

func (s *Server) handleTeamRemoveMember(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireTeamIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.teamManagement.Handler.RemoveMemberHandler(
		r.Context(),
		idempotencyKey,
		actorUserID,
		r.PathValue("teamId"),
		r.PathValue("memberId"),
		strings.TrimSpace(r.Header.Get("X-MFA-Code")),
	)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamRemoveMemberV1(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("teamId", r.PathValue("team_id"))
	r.SetPathValue("memberId", r.PathValue("member_id"))
	s.handleTeamRemoveMember(w, r)
}

func (s *Server) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	resp, err := s.teamManagement.Handler.GetTeamDashboardHandler(r.Context(), actorUserID, r.PathValue("teamId"))
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamMembership(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	_, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		writeTeamError(w, http.StatusBadRequest, "invalid_request", "user_id query parameter is required")
		return
	}
	resp, err := s.teamManagement.Handler.CheckMembershipHandler(r.Context(), r.PathValue("teamId"), userID)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamAuditLogs(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	resp, err := s.teamManagement.Handler.ListAuditLogsHandler(
		r.Context(),
		actorUserID,
		r.PathValue("teamId"),
		r.URL.Query().Get("limit"),
	)
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTeamExportMembers(w http.ResponseWriter, r *http.Request) {
	if !requireTeamAuthorization(w, r) || !requireTeamRequestID(w, r) {
		return
	}
	actorUserID, ok := requireTeamUser(w, r)
	if !ok {
		return
	}
	resp, err := s.teamManagement.Handler.ExportMembersHandler(r.Context(), actorUserID, r.PathValue("teamId"))
	if err != nil {
		writeTeamDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}
