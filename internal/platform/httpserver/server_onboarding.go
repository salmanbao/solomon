package httpserver

import (
	"errors"
	"net/http"
	"strings"

	onboardingerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	onboardinghttp "solomon/contexts/identity-access/onboarding-service/transport/http"
)

func writeOnboardingError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, onboardinghttp.ErrorResponse{Code: code, Message: message})
}

func writeOnboardingDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, onboardingerrors.ErrFlowNotFound),
		errors.Is(err, onboardingerrors.ErrProgressNotFound),
		errors.Is(err, onboardingerrors.ErrStepNotFound),
		errors.Is(err, onboardingerrors.ErrNotFound):
		writeOnboardingError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, onboardingerrors.ErrInvalidRequest),
		errors.Is(err, onboardingerrors.ErrIdempotencyKeyRequired):
		writeOnboardingError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, onboardingerrors.ErrSchemaInvalid),
		errors.Is(err, onboardingerrors.ErrUnknownRole):
		writeOnboardingError(w, http.StatusUnprocessableEntity, "unprocessable_entity", err.Error())
	case errors.Is(err, onboardingerrors.ErrStepAlreadyCompleted),
		errors.Is(err, onboardingerrors.ErrFlowAlreadyCompleted),
		errors.Is(err, onboardingerrors.ErrResumeNotAllowed),
		errors.Is(err, onboardingerrors.ErrIdempotencyConflict),
		errors.Is(err, onboardingerrors.ErrConflict):
		writeOnboardingError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, onboardingerrors.ErrDependencyUnavailable):
		writeOnboardingError(w, http.StatusServiceUnavailable, "dependency_unavailable", err.Error())
	default:
		writeOnboardingError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireOnboardingAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeOnboardingError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireOnboardingRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeOnboardingError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireOnboardingUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeOnboardingError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireOnboardingIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeOnboardingError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return key, true
}

func (s *Server) handleOnboardingGetFlow(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	userID, ok := requireOnboardingUser(w, r)
	if !ok {
		return
	}
	resp, err := s.onboarding.Handler.GetFlowHandler(r.Context(), userID)
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOnboardingCompleteStep(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	userID, ok := requireOnboardingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireOnboardingIdempotencyKey(w, r)
	if !ok {
		return
	}
	stepKey := strings.TrimSpace(r.PathValue("step_key"))
	if stepKey == "" {
		writeOnboardingError(w, http.StatusBadRequest, "invalid_request", "step_key is required")
		return
	}
	var req onboardinghttp.CompleteStepRequest
	if !s.decodeJSON(w, r, &req, writeOnboardingError) {
		return
	}
	resp, err := s.onboarding.Handler.CompleteStepHandler(r.Context(), idempotencyKey, userID, stepKey, req)
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOnboardingSkip(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	userID, ok := requireOnboardingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireOnboardingIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req onboardinghttp.SkipFlowRequest
	if !s.decodeJSON(w, r, &req, writeOnboardingError) {
		return
	}
	resp, err := s.onboarding.Handler.SkipFlowHandler(r.Context(), idempotencyKey, userID, req)
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOnboardingResume(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	userID, ok := requireOnboardingUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireOnboardingIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.onboarding.Handler.ResumeFlowHandler(r.Context(), idempotencyKey, userID)
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOnboardingAdminFlows(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	if _, ok := requireOnboardingUser(w, r); !ok {
		return
	}
	resp, err := s.onboarding.Handler.ListAdminFlowsHandler(r.Context())
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOnboardingUserRegisteredEvent(w http.ResponseWriter, r *http.Request) {
	if !requireOnboardingAuthorization(w, r) || !requireOnboardingRequestID(w, r) {
		return
	}
	var req onboardinghttp.UserRegisteredEventRequest
	if !s.decodeJSON(w, r, &req, writeOnboardingError) {
		return
	}
	resp, err := s.onboarding.Handler.ConsumeUserRegisteredEventHandler(r.Context(), req)
	if err != nil {
		writeOnboardingDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}
