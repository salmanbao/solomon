package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	editordashboarderrors "solomon/contexts/campaign-editorial/editor-dashboard-service/domain/errors"
	editordashboardhttp "solomon/contexts/campaign-editorial/editor-dashboard-service/transport/http"
)

func writeEditorError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, editordashboardhttp.ErrorEnvelope{
		Status: "error",
		Error: editordashboardhttp.ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeEditorDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, editordashboarderrors.ErrInvalidRequest):
		writeEditorError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, editordashboarderrors.ErrNotFound):
		writeEditorError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, editordashboarderrors.ErrForbidden):
		writeEditorError(w, http.StatusForbidden, "PERMISSION_DENIED", err.Error(), nil)
	case errors.Is(err, editordashboarderrors.ErrIdempotencyKeyRequired):
		writeEditorError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", err.Error(), nil)
	case errors.Is(err, editordashboarderrors.ErrIdempotencyConflict):
		writeEditorError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, editordashboarderrors.ErrDependencyUnavailable):
		writeEditorError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", err.Error(), nil)
	default:
		writeEditorError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireEditorAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeEditorError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireEditorRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeEditorError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireEditorUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeEditorError(w, http.StatusUnauthorized, "USER_REQUIRED", "X-User-Id header is required", nil)
		return "", false
	}
	return userID, true
}

func requireEditorIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeEditorError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required", nil)
		return "", false
	}
	return key, true
}

func (s *Server) handleEditorFeed(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.FeedHandler(
		r.Context(),
		userID,
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
		r.URL.Query().Get("sort_by"),
	)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorSubmissions(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.SubmissionsHandler(
		r.Context(),
		userID,
		r.URL.Query().Get("status"),
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
	)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorEarnings(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.EarningsHandler(r.Context(), userID)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorPerformance(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.PerformanceHandler(r.Context(), userID)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorSubmissionsExport(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.ExportSubmissionsHandler(r.Context(), userID, r.URL.Query().Get("status"))
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorSaveCampaign(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireEditorIdempotency(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.SaveCampaignHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("id"),
	)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorRemoveSavedCampaign(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireEditorIdempotency(w, r)
	if !ok {
		return
	}
	resp, err := s.editorDashboard.Handler.RemoveSavedCampaignHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("id"),
	)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleEditorSummary(w http.ResponseWriter, r *http.Request) {
	if !requireEditorAuthorization(w, r) || !requireEditorRequestID(w, r) {
		return
	}
	userID, ok := requireEditorUser(w, r)
	if !ok {
		return
	}
	feed, err := s.editorDashboard.Handler.FeedHandler(r.Context(), userID, "5", "0", r.URL.Query().Get("sort_by"))
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	earnings, err := s.editorDashboard.Handler.EarningsHandler(r.Context(), userID)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	performance, err := s.editorDashboard.Handler.PerformanceHandler(r.Context(), userID)
	if err != nil {
		writeEditorDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"feed":        feed.Data.Items,
			"earnings":    earnings.Data,
			"performance": performance.Data,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
