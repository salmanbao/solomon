package httpserver

import (
	"errors"
	"net/http"
	"strings"

	subscriptionerrors "solomon/contexts/community-experience/subscription-service/domain/errors"
	subscriptionhttp "solomon/contexts/community-experience/subscription-service/transport/http"
)

func writeSubscriptionError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, subscriptionhttp.ErrorResponse{Code: code, Message: message})
}

func writeSubscriptionDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, subscriptionerrors.ErrSubscriptionNotFound),
		errors.Is(err, subscriptionerrors.ErrPlanNotFound),
		errors.Is(err, subscriptionerrors.ErrNotFound):
		writeSubscriptionError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, subscriptionerrors.ErrInvalidRequest),
		errors.Is(err, subscriptionerrors.ErrIdempotencyKeyRequired):
		writeSubscriptionError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, subscriptionerrors.ErrIdempotencyConflict),
		errors.Is(err, subscriptionerrors.ErrConflict),
		errors.Is(err, subscriptionerrors.ErrPlanInactive),
		errors.Is(err, subscriptionerrors.ErrTrialAlreadyUsed),
		errors.Is(err, subscriptionerrors.ErrInvalidTransition):
		writeSubscriptionError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, subscriptionerrors.ErrPaymentRequired):
		writeSubscriptionError(w, http.StatusPaymentRequired, "payment_required", err.Error())
	case errors.Is(err, subscriptionerrors.ErrDependencyUnavailable):
		writeSubscriptionError(w, http.StatusFailedDependency, "dependency_unavailable", err.Error())
	case errors.Is(err, subscriptionerrors.ErrForbidden):
		writeSubscriptionError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeSubscriptionError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireSubscriptionAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeSubscriptionError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireSubscriptionRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeSubscriptionError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireSubscriptionUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeSubscriptionError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireSubscriptionIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeSubscriptionError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func (s *Server) handleSubscriptionCreate(w http.ResponseWriter, r *http.Request) {
	if !requireSubscriptionAuthorization(w, r) || !requireSubscriptionRequestID(w, r) {
		return
	}
	userID, ok := requireSubscriptionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubscriptionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req subscriptionhttp.CreateSubscriptionRequest
	if !s.decodeJSON(w, r, &req, writeSubscriptionError) {
		return
	}
	resp, err := s.subscription.Handler.CreateSubscriptionHandler(r.Context(), idempotencyKey, userID, req)
	if err != nil {
		writeSubscriptionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleSubscriptionChangePlan(w http.ResponseWriter, r *http.Request) {
	if !requireSubscriptionAuthorization(w, r) || !requireSubscriptionRequestID(w, r) {
		return
	}
	userID, ok := requireSubscriptionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubscriptionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req subscriptionhttp.ChangePlanRequest
	if !s.decodeJSON(w, r, &req, writeSubscriptionError) {
		return
	}
	resp, err := s.subscription.Handler.ChangePlanHandler(r.Context(), idempotencyKey, userID, r.PathValue("subscription_id"), req)
	if err != nil {
		writeSubscriptionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubscriptionCancel(w http.ResponseWriter, r *http.Request) {
	if !requireSubscriptionAuthorization(w, r) || !requireSubscriptionRequestID(w, r) {
		return
	}
	userID, ok := requireSubscriptionUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireSubscriptionIdempotencyKey(w, r)
	if !ok {
		return
	}

	var req subscriptionhttp.CancelSubscriptionRequest
	if !s.decodeJSON(w, r, &req, writeSubscriptionError) {
		return
	}
	resp, err := s.subscription.Handler.CancelSubscriptionHandler(r.Context(), idempotencyKey, userID, r.PathValue("subscription_id"), req)
	if err != nil {
		writeSubscriptionDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
