package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	influencerdashboarderrors "solomon/contexts/campaign-editorial/influencer-dashboard-service/domain/errors"
	influencerdashboardhttp "solomon/contexts/campaign-editorial/influencer-dashboard-service/transport/http"
)

func writeInfluencerError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, influencerdashboardhttp.ErrorEnvelope{
		Status: "error",
		Error: influencerdashboardhttp.ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeInfluencerDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, influencerdashboarderrors.ErrInvalidRequest):
		writeInfluencerError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, influencerdashboarderrors.ErrNotFound):
		writeInfluencerError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, influencerdashboarderrors.ErrForbidden):
		writeInfluencerError(w, http.StatusForbidden, "PERMISSION_DENIED", err.Error(), nil)
	case errors.Is(err, influencerdashboarderrors.ErrIdempotencyKeyRequired):
		writeInfluencerError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", err.Error(), nil)
	case errors.Is(err, influencerdashboarderrors.ErrIdempotencyConflict):
		writeInfluencerError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, influencerdashboarderrors.ErrDependencyUnavailable):
		writeInfluencerError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", err.Error(), nil)
	default:
		writeInfluencerError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireInfluencerAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeInfluencerError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireInfluencerRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeInfluencerError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireInfluencerUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeInfluencerError(w, http.StatusUnauthorized, "USER_REQUIRED", "X-User-Id header is required", nil)
		return "", false
	}
	return userID, true
}

func requireInfluencerIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeInfluencerError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required", nil)
		return "", false
	}
	return key, true
}

func (s *Server) handleInfluencerSummary(w http.ResponseWriter, r *http.Request) {
	if !requireInfluencerAuthorization(w, r) || !requireInfluencerRequestID(w, r) {
		return
	}
	userID, ok := requireInfluencerUser(w, r)
	if !ok {
		return
	}
	resp, err := s.influencerDashboard.Handler.SummaryHandler(r.Context(), userID)
	if err != nil {
		writeInfluencerDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleInfluencerContent(w http.ResponseWriter, r *http.Request) {
	if !requireInfluencerAuthorization(w, r) || !requireInfluencerRequestID(w, r) {
		return
	}
	userID, ok := requireInfluencerUser(w, r)
	if !ok {
		return
	}
	resp, err := s.influencerDashboard.Handler.ContentHandler(
		r.Context(),
		userID,
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
		r.URL.Query().Get("view"),
		r.URL.Query().Get("sort_by"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("date_from"),
		r.URL.Query().Get("date_to"),
	)
	if err != nil {
		writeInfluencerDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleInfluencerCreateGoal(w http.ResponseWriter, r *http.Request) {
	if !requireInfluencerAuthorization(w, r) || !requireInfluencerRequestID(w, r) {
		return
	}
	userID, ok := requireInfluencerUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireInfluencerIdempotency(w, r)
	if !ok {
		return
	}
	var req influencerdashboardhttp.CreateGoalRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeInfluencerError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.influencerDashboard.Handler.CreateGoalHandler(r.Context(), idempotencyKey, userID, req)
	if err != nil {
		writeInfluencerDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
