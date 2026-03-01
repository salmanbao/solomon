package httpserver

import (
	"errors"
	"net/http"
	"strings"

	storefronterrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	storefronthttp "solomon/contexts/community-experience/storefront-service/transport/http"
)

func writeStorefrontError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, storefronthttp.ErrorResponse{Code: code, Message: message})
}

func writeStorefrontDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storefronterrors.ErrStorefrontNotFound),
		errors.Is(err, storefronterrors.ErrNotFound):
		writeStorefrontError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, storefronterrors.ErrInvalidRequest),
		errors.Is(err, storefronterrors.ErrIdempotencyKeyRequired):
		writeStorefrontError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, storefronterrors.ErrPrivateAccessDenied),
		errors.Is(err, storefronterrors.ErrUnauthorized):
		writeStorefrontError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, storefronterrors.ErrForbidden):
		writeStorefrontError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, storefronterrors.ErrConflict),
		errors.Is(err, storefronterrors.ErrAlreadyPublished),
		errors.Is(err, storefronterrors.ErrIdempotencyConflict):
		writeStorefrontError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, storefronterrors.ErrDependencyUnavailable):
		writeStorefrontError(w, http.StatusFailedDependency, "dependency_unavailable", err.Error())
	default:
		writeStorefrontError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireStorefrontAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeStorefrontError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireStorefrontRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeStorefrontError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireStorefrontUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeStorefrontError(w, http.StatusUnauthorized, "missing_user", "X-User-Id header is required")
		return "", false
	}
	return userID, true
}

func requireStorefrontIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeStorefrontError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func isStorefrontID(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "storefront_")
}

func (s *Server) handleStorefrontCreate(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
		return
	}
	actorUserID, ok := requireStorefrontUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireStorefrontIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req storefronthttp.CreateStorefrontRequest
	if !s.decodeJSON(w, r, &req, writeStorefrontError) {
		return
	}
	resp, err := s.storefront.Handler.CreateStorefrontHandler(r.Context(), idempotencyKey, actorUserID, req)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleStorefrontUpdate(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
		return
	}
	actorUserID, ok := requireStorefrontUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireStorefrontIdempotencyKey(w, r)
	if !ok {
		return
	}
	var req storefronthttp.UpdateStorefrontRequest
	if !s.decodeJSON(w, r, &req, writeStorefrontError) {
		return
	}
	resp, err := s.storefront.Handler.UpdateStorefrontHandler(r.Context(), idempotencyKey, actorUserID, r.PathValue("storefrontId"), req)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStorefrontGet(w http.ResponseWriter, r *http.Request) {
	identifier := strings.TrimSpace(r.PathValue("identifier"))
	if identifier == "" {
		writeStorefrontError(w, http.StatusBadRequest, "invalid_request", "identifier is required")
		return
	}
	if isStorefrontID(identifier) {
		if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
			return
		}
		actorUserID, ok := requireStorefrontUser(w, r)
		if !ok {
			return
		}
		resp, err := s.storefront.Handler.GetStorefrontByIDHandler(r.Context(), identifier, actorUserID)
		if err != nil {
			writeStorefrontDomainError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp, err := s.storefront.Handler.GetStorefrontBySlugHandler(r.Context(), identifier)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStorefrontPublish(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
		return
	}
	actorUserID, ok := requireStorefrontUser(w, r)
	if !ok {
		return
	}
	idempotencyKey, ok := requireStorefrontIdempotencyKey(w, r)
	if !ok {
		return
	}
	resp, err := s.storefront.Handler.PublishStorefrontHandler(r.Context(), idempotencyKey, actorUserID, r.PathValue("storefrontId"))
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStorefrontReport(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontRequestID(w, r) {
		return
	}
	idempotencyKey, ok := requireStorefrontIdempotencyKey(w, r)
	if !ok {
		return
	}
	actorUserID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if actorUserID == "" {
		actorUserID = "anonymous"
	}
	var req storefronthttp.ReportStorefrontRequest
	if !s.decodeJSON(w, r, &req, writeStorefrontError) {
		return
	}
	resp, err := s.storefront.Handler.ReportStorefrontHandler(r.Context(), idempotencyKey, actorUserID, r.PathValue("storefrontId"), req)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (s *Server) handleStorefrontProductPublishedEvent(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
		return
	}
	var req storefronthttp.ProductPublishedEventRequest
	if !s.decodeJSON(w, r, &req, writeStorefrontError) {
		return
	}
	resp, err := s.storefront.Handler.ConsumeProductPublishedEventHandler(r.Context(), req)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func (s *Server) handleStorefrontSubscriptionProjection(w http.ResponseWriter, r *http.Request) {
	if !requireStorefrontAuthorization(w, r) || !requireStorefrontRequestID(w, r) {
		return
	}
	var req storefronthttp.SubscriptionProjectionRequest
	if !s.decodeJSON(w, r, &req, writeStorefrontError) {
		return
	}
	resp, err := s.storefront.Handler.UpsertSubscriptionProjectionHandler(r.Context(), req)
	if err != nil {
		writeStorefrontDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}
