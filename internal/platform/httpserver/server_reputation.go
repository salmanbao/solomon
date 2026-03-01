package httpserver

import (
	"errors"
	"net/http"
	"strings"

	reputationerrors "solomon/contexts/community-experience/reputation-service/domain/errors"
	reputationhttp "solomon/contexts/community-experience/reputation-service/transport/http"
)

func writeReputationError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, reputationhttp.ErrorResponse{Code: code, Message: message})
}

func writeReputationDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reputationerrors.ErrInvalidRequest):
		writeReputationError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, reputationerrors.ErrNotFound):
		writeReputationError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, reputationerrors.ErrDependencyUnavailable):
		writeReputationError(w, http.StatusFailedDependency, "dependency_unavailable", err.Error())
	default:
		writeReputationError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireReputationAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeReputationError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireReputationRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeReputationError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func (s *Server) handleReputationGetUser(w http.ResponseWriter, r *http.Request) {
	if !requireReputationAuthorization(w, r) || !requireReputationRequestID(w, r) {
		return
	}

	userID := strings.TrimSpace(r.PathValue("user_id"))
	if userID == "" {
		writeReputationError(w, http.StatusBadRequest, "invalid_request", "user_id is required")
		return
	}

	resp, err := s.reputation.Handler.GetUserReputationHandler(r.Context(), userID)
	if err != nil {
		writeReputationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleReputationLeaderboard(w http.ResponseWriter, r *http.Request) {
	if !requireReputationAuthorization(w, r) || !requireReputationRequestID(w, r) {
		return
	}

	req := reputationhttp.LeaderboardRequest{
		Tier:   r.URL.Query().Get("tier"),
		Limit:  r.URL.Query().Get("limit"),
		Offset: r.URL.Query().Get("offset"),
	}
	resp, err := s.reputation.Handler.GetLeaderboardHandler(
		r.Context(),
		req,
		strings.TrimSpace(r.Header.Get("X-User-Id")),
	)
	if err != nil {
		writeReputationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
