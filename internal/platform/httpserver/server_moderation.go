package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	moderationerrors "solomon/contexts/moderation-safety/moderation-service/domain/errors"
	moderationhttp "solomon/contexts/moderation-safety/moderation-service/transport/http"
)

func writeModerationError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, moderationhttp.ErrorEnvelope{
		Status: "error",
		Error: moderationhttp.ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeModerationDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, moderationerrors.ErrInvalidRequest):
		writeModerationError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, moderationerrors.ErrNotFound):
		writeModerationError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, moderationerrors.ErrForbidden):
		writeModerationError(w, http.StatusForbidden, "PERMISSION_DENIED", err.Error(), nil)
	case errors.Is(err, moderationerrors.ErrIdempotencyKeyRequired):
		writeModerationError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", err.Error(), nil)
	case errors.Is(err, moderationerrors.ErrIdempotencyConflict):
		writeModerationError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, moderationerrors.ErrDependencyUnavailable):
		writeModerationError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", err.Error(), nil)
	default:
		writeModerationError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireModerationAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeModerationError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireModerationRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeModerationError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireModerationUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeModerationError(w, http.StatusUnauthorized, "USER_REQUIRED", "X-User-Id header is required", nil)
		return "", false
	}
	return userID, true
}

func requireModerationIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeModerationError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required", nil)
		return "", false
	}
	return key, true
}

func (s *Server) handleModerationQueue(w http.ResponseWriter, r *http.Request) {
	if !requireModerationAuthorization(w, r) || !requireModerationRequestID(w, r) {
		return
	}
	if _, ok := requireModerationUser(w, r); !ok {
		return
	}
	resp, err := s.moderation.Handler.ListQueueHandler(
		r.Context(),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
	)
	if err != nil {
		writeModerationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleModerationApprove(w http.ResponseWriter, r *http.Request) {
	if !requireModerationAuthorization(w, r) || !requireModerationRequestID(w, r) {
		return
	}
	moderatorID, ok := requireModerationUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireModerationIdempotency(w, r)
	if !ok {
		return
	}
	var req moderationhttp.ApproveRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeModerationError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.moderation.Handler.ApproveHandler(r.Context(), idempotencyKey, moderatorID, req)
	if err != nil {
		writeModerationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleModerationReject(w http.ResponseWriter, r *http.Request) {
	if !requireModerationAuthorization(w, r) || !requireModerationRequestID(w, r) {
		return
	}
	moderatorID, ok := requireModerationUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireModerationIdempotency(w, r)
	if !ok {
		return
	}
	var req moderationhttp.RejectRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeModerationError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.moderation.Handler.RejectHandler(r.Context(), idempotencyKey, moderatorID, req)
	if err != nil {
		writeModerationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleModerationFlag(w http.ResponseWriter, r *http.Request) {
	if !requireModerationAuthorization(w, r) || !requireModerationRequestID(w, r) {
		return
	}
	moderatorID, ok := requireModerationUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireModerationIdempotency(w, r)
	if !ok {
		return
	}
	var req moderationhttp.FlagRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeModerationError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.moderation.Handler.FlagHandler(r.Context(), idempotencyKey, moderatorID, req)
	if err != nil {
		writeModerationDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
