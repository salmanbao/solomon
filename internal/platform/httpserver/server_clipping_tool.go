package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	clippingerrors "solomon/contexts/campaign-editorial/clipping-tool-service/domain/errors"
	clippinghttp "solomon/contexts/campaign-editorial/clipping-tool-service/transport/http"
)

func writeClippingError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, clippinghttp.ErrorEnvelope{
		Status: "error",
		Error: clippinghttp.ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeClippingDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, clippingerrors.ErrInvalidRequest):
		writeClippingError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrNotFound):
		writeClippingError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrForbidden):
		writeClippingError(w, http.StatusForbidden, "PERMISSION_DENIED", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrConflict):
		writeClippingError(w, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrProjectExporting):
		writeClippingError(w, http.StatusConflict, "EXPORT_IN_PROGRESS", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrIdempotencyKeyRequired):
		writeClippingError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrIdempotencyConflict):
		writeClippingError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, clippingerrors.ErrDependencyUnavailable):
		writeClippingError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", err.Error(), nil)
	default:
		writeClippingError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireClippingAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeClippingError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireClippingRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeClippingError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireClippingUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeClippingError(w, http.StatusUnauthorized, "USER_REQUIRED", "X-User-Id header is required", nil)
		return "", false
	}
	return userID, true
}

func requireClippingIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeClippingError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required", nil)
		return "", false
	}
	return key, true
}

func (s *Server) handleClippingCreateProject(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireClippingIdempotency(w, r)
	if !ok {
		return
	}
	var req clippinghttp.CreateProjectRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeClippingError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.clippingTool.Handler.CreateProjectHandler(r.Context(), idempotencyKey, userID, req)
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleClippingGetProject(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	resp, err := s.clippingTool.Handler.GetProjectHandler(r.Context(), userID, r.PathValue("project_id"))
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClippingUpdateTimeline(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireClippingIdempotency(w, r)
	if !ok {
		return
	}
	var req clippinghttp.UpdateTimelineRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeClippingError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.clippingTool.Handler.UpdateTimelineHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("project_id"),
		req,
	)
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClippingInsertTimeline(w http.ResponseWriter, r *http.Request) {
	s.handleClippingUpdateTimeline(w, r)
}

func (s *Server) handleClippingGetSuggestions(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	resp, err := s.clippingTool.Handler.GetSuggestionsHandler(r.Context(), userID, r.PathValue("project_id"))
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClippingRequestExport(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireClippingIdempotency(w, r)
	if !ok {
		return
	}
	var req clippinghttp.ExportRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeClippingError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.clippingTool.Handler.RequestExportHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("project_id"),
		req,
	)
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (s *Server) handleClippingGetExportStatus(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	resp, err := s.clippingTool.Handler.GetExportStatusHandler(
		r.Context(),
		userID,
		r.PathValue("project_id"),
		r.PathValue("export_id"),
	)
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleClippingSubmit(w http.ResponseWriter, r *http.Request) {
	if !requireClippingAuthorization(w, r) || !requireClippingRequestID(w, r) {
		return
	}
	userID, ok := requireClippingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireClippingIdempotency(w, r)
	if !ok {
		return
	}
	var req clippinghttp.SubmitRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeClippingError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.clippingTool.Handler.SubmitToCampaignHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("project_id"),
		req,
	)
	if err != nil {
		writeClippingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
