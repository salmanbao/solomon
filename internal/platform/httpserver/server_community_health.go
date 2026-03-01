package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	communityhealthdomainerrors "solomon/contexts/community-experience/community-health-service/domain/errors"
	communityhealthhttp "solomon/contexts/community-experience/community-health-service/transport/http"
)

func writeCommunityHealthError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, communityhealthhttp.ErrorResponse{Code: code, Message: message})
}

func writeCommunityHealthDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, communityhealthdomainerrors.ErrUnauthorizedWebhook):
		writeCommunityHealthError(w, http.StatusUnauthorized, "unauthorized_webhook", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrNotFound):
		writeCommunityHealthError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrInvalidRequest):
		writeCommunityHealthError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrIdempotencyKeyRequired):
		writeCommunityHealthError(w, http.StatusBadRequest, "idempotency_key_required", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrIdempotencyConflict):
		writeCommunityHealthError(w, http.StatusConflict, "idempotency_conflict", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrConflict):
		writeCommunityHealthError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, communityhealthdomainerrors.ErrForbidden):
		writeCommunityHealthError(w, http.StatusForbidden, "forbidden", err.Error())
	default:
		writeCommunityHealthError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func requireCommunityHealthAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeCommunityHealthError(w, http.StatusUnauthorized, "unauthorized", "Authorization bearer token is required")
		return false
	}
	return true
}

func requireCommunityHealthRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeCommunityHealthError(w, http.StatusBadRequest, "missing_request_id", "X-Request-Id header is required")
		return false
	}
	return true
}

func requireCommunityHealthIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeCommunityHealthError(w, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key header is required")
		return "", false
	}
	return idempotencyKey, true
}

func communityHealthWebhookSecret() string {
	for _, key := range []string{"COMMUNITY_HEALTH_WEBHOOK_SECRET", "CHS_WEBHOOK_SECRET"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func communityHealthWebhookSignature(r *http.Request) string {
	for _, key := range []string{"X-Webhook-Signature", "X-Signature"} {
		if value := strings.TrimSpace(r.Header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func validateCommunityHealthWebhookSignature(signature string, body []byte, secret string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" || strings.TrimSpace(secret) == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(signature), "sha256=") {
		signature = signature[7:]
	}
	provided, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	return hmac.Equal(provided, expected)
}

func (s *Server) handleCommunityHealthWebhook(w http.ResponseWriter, r *http.Request) {
	idempotencyKey, ok := requireCommunityHealthIdempotencyKey(w, r)
	if !ok {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeCommunityHealthError(w, http.StatusBadRequest, "invalid_request", "unable to read request body")
		return
	}

	secret := communityHealthWebhookSecret()
	if secret == "" {
		writeCommunityHealthError(w, http.StatusInternalServerError, "internal_error", "community health webhook secret is not configured")
		return
	}
	if !validateCommunityHealthWebhookSignature(communityHealthWebhookSignature(r), body, secret) {
		writeCommunityHealthDomainError(w, communityhealthdomainerrors.ErrUnauthorizedWebhook)
		return
	}

	var req communityhealthhttp.WebhookIngestRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeCommunityHealthError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	resp, err := s.communityHealth.Handler.IngestWebhookHandler(r.Context(), idempotencyKey, req)
	if err != nil {
		writeCommunityHealthDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCommunityHealthGetScore(w http.ResponseWriter, r *http.Request) {
	if !requireCommunityHealthAuthorization(w, r) || !requireCommunityHealthRequestID(w, r) {
		return
	}
	resp, err := s.communityHealth.Handler.GetCommunityHealthScoreHandler(r.Context(), r.PathValue("server_id"))
	if err != nil {
		writeCommunityHealthDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCommunityHealthGetUserRisk(w http.ResponseWriter, r *http.Request) {
	if !requireCommunityHealthAuthorization(w, r) || !requireCommunityHealthRequestID(w, r) {
		return
	}
	resp, err := s.communityHealth.Handler.GetUserRiskScoreHandler(
		r.Context(),
		r.PathValue("server_id"),
		r.PathValue("user_id"),
	)
	if err != nil {
		writeCommunityHealthDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
