package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	discoveryerrors "solomon/contexts/campaign-editorial/campaign-discovery-service/domain/errors"
	discoveryhttp "solomon/contexts/campaign-editorial/campaign-discovery-service/transport/http"
)

func writeDiscoverError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, discoveryhttp.ErrorEnvelope{
		Status: "error",
		Error: discoveryhttp.ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func writeDiscoverDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, discoveryerrors.ErrInvalidRequest):
		writeDiscoverError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	case errors.Is(err, discoveryerrors.ErrNotFound):
		writeDiscoverError(w, http.StatusNotFound, "CAMPAIGN_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, discoveryerrors.ErrForbidden):
		writeDiscoverError(w, http.StatusForbidden, "PERMISSION_DENIED", err.Error(), nil)
	case errors.Is(err, discoveryerrors.ErrIdempotencyKeyRequired):
		writeDiscoverError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", err.Error(), nil)
	case errors.Is(err, discoveryerrors.ErrIdempotencyConflict):
		writeDiscoverError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error(), nil)
	case errors.Is(err, discoveryerrors.ErrDependencyUnavailable):
		writeDiscoverError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", err.Error(), nil)
	default:
		writeDiscoverError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireDiscoverAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeDiscoverError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireDiscoverRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeDiscoverError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireDiscoverUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeDiscoverError(w, http.StatusUnauthorized, "USER_REQUIRED", "X-User-Id header is required", nil)
		return "", false
	}
	return userID, true
}

func (s *Server) handleDiscoverBrowse(w http.ResponseWriter, r *http.Request) {
	if !requireDiscoverAuthorization(w, r) || !requireDiscoverRequestID(w, r) {
		return
	}
	userID, ok := requireDiscoverUser(w, r)
	if !ok {
		return
	}
	req := discoveryhttp.BrowseCampaignsRequest{
		PageSize:        r.URL.Query().Get("page_size"),
		Cursor:          r.URL.Query().Get("cursor"),
		SortBy:          r.URL.Query().Get("sort_by"),
		Category:        r.URL.Query().Get("category"),
		BudgetMin:       r.URL.Query().Get("budget_min"),
		BudgetMax:       r.URL.Query().Get("budget_max"),
		DeadlineAfter:   r.URL.Query().Get("deadline_after"),
		DeadlineBefore:  r.URL.Query().Get("deadline_before"),
		Platforms:       r.URL.Query().Get("platforms"),
		State:           r.URL.Query().Get("state"),
		ExcludeFeatured: r.URL.Query().Get("exclude_featured"),
	}
	resp, err := s.campaignDiscovery.Handler.BrowseCampaignsHandler(r.Context(), userID, req)
	if err != nil {
		writeDiscoverDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDiscoverSearch(w http.ResponseWriter, r *http.Request) {
	if !requireDiscoverAuthorization(w, r) || !requireDiscoverRequestID(w, r) {
		return
	}
	userID, ok := requireDiscoverUser(w, r)
	if !ok {
		return
	}
	req := discoveryhttp.SearchCampaignsRequest{
		Query:     r.URL.Query().Get("q"),
		Category:  r.URL.Query().Get("category"),
		BudgetMin: r.URL.Query().Get("budget_min"),
		Limit:     r.URL.Query().Get("limit"),
		Offset:    r.URL.Query().Get("offset"),
	}
	resp, err := s.campaignDiscovery.Handler.SearchCampaignsHandler(r.Context(), userID, req)
	if err != nil {
		writeDiscoverDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDiscoverCampaignDetails(w http.ResponseWriter, r *http.Request) {
	if !requireDiscoverAuthorization(w, r) || !requireDiscoverRequestID(w, r) {
		return
	}
	userID, ok := requireDiscoverUser(w, r)
	if !ok {
		return
	}
	resp, err := s.campaignDiscovery.Handler.GetCampaignDetailsHandler(r.Context(), userID, r.PathValue("campaign_id"))
	if err != nil {
		writeDiscoverDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDiscoverBookmark(w http.ResponseWriter, r *http.Request) {
	if !requireDiscoverAuthorization(w, r) || !requireDiscoverRequestID(w, r) {
		return
	}
	userID, ok := requireDiscoverUser(w, r)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	var req discoveryhttp.SaveBookmarkRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeDiscoverError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	resp, err := s.campaignDiscovery.Handler.SaveBookmarkHandler(
		r.Context(),
		idempotencyKey,
		userID,
		r.PathValue("campaign_id"),
		req,
	)
	if err != nil {
		writeDiscoverDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
