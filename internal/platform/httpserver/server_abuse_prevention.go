package httpserver

import (
	"net/http"
	"strings"
	"time"
)

type abuseErrorEnvelope struct {
	Status    string         `json:"status"`
	Error     abuseErrorBody `json:"error"`
	Timestamp string         `json:"timestamp"`
}

type abuseErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type abuseLoginRequest struct {
	FailedAttempts int  `json:"failed_attempts"`
	KnownDevice    bool `json:"known_device"`
	VPNDetected    bool `json:"vpn_detected"`
}

type abuseChallengeRequest struct {
	Passed bool `json:"passed"`
}

type abuseLoginResponse struct {
	Status    string        `json:"status"`
	Data      abuseDecision `json:"data"`
	Timestamp string        `json:"timestamp"`
}

type abuseChallengeResponse struct {
	Status    string            `json:"status"`
	Data      abuseChallengeDTO `json:"data"`
	Timestamp string            `json:"timestamp"`
}

type abuseThreatsResponse struct {
	Status    string          `json:"status"`
	Data      abuseThreatsDTO `json:"data"`
	Timestamp string          `json:"timestamp"`
}

type abuseDecision struct {
	RiskScore   float64 `json:"risk_score"`
	RiskTier    string  `json:"risk_tier"`
	Decision    string  `json:"decision"`
	ChallengeID string  `json:"challenge_id,omitempty"`
}

type abuseChallengeDTO struct {
	ChallengeID string `json:"challenge_id"`
	Result      string `json:"result"`
}

type abuseThreatsDTO struct {
	ActiveSuspiciousAccounts int `json:"active_suspicious_accounts"`
	FailedLoginsLast24h      int `json:"failed_logins_last_24h"`
	ActiveLockouts           int `json:"active_lockouts"`
}

func writeAbuseError(w http.ResponseWriter, status int, code string, message string, details map[string]any) {
	writeJSON(w, status, abuseErrorEnvelope{
		Status: "error",
		Error: abuseErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func requireAbuseRequestID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Request-Id")) == "" {
		writeAbuseError(w, http.StatusBadRequest, "REQUEST_ID_REQUIRED", "X-Request-Id header is required", nil)
		return false
	}
	return true
}

func requireAbuseAuthorization(w http.ResponseWriter, r *http.Request) bool {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeAbuseError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization bearer token is required", nil)
		return false
	}
	return true
}

func requireAbuseIdempotencyKey(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		writeAbuseError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required", nil)
		return false
	}
	return true
}

func requireAbuseAdminID(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Admin-Id")) == "" {
		writeAbuseError(w, http.StatusUnauthorized, "ADMIN_REQUIRED", "X-Admin-Id header is required", nil)
		return false
	}
	return true
}

func (s *Server) handleAbuseLogin(w http.ResponseWriter, r *http.Request) {
	if !requireAbuseRequestID(w, r) {
		return
	}
	var req abuseLoginRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeAbuseError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	decision := evaluateAbuseRisk(req)
	writeJSON(w, http.StatusOK, abuseLoginResponse{
		Status:    "success",
		Data:      decision,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleAbuseChallenge(w http.ResponseWriter, r *http.Request) {
	if !requireAbuseAuthorization(w, r) || !requireAbuseRequestID(w, r) || !requireAbuseIdempotencyKey(w, r) {
		return
	}
	var req abuseChallengeRequest
	if !s.decodeJSON(w, r, &req, func(w http.ResponseWriter, status int, code string, message string) {
		writeAbuseError(w, status, strings.ToUpper(code), message, nil)
	}) {
		return
	}
	result := "failed"
	if req.Passed {
		result = "passed"
	}
	writeJSON(w, http.StatusOK, abuseChallengeResponse{
		Status: "success",
		Data: abuseChallengeDTO{
			ChallengeID: r.PathValue("id"),
			Result:      result,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleAbuseAdminThreats(w http.ResponseWriter, r *http.Request) {
	if !requireAbuseAuthorization(w, r) || !requireAbuseRequestID(w, r) || !requireAbuseAdminID(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, abuseThreatsResponse{
		Status: "success",
		Data: abuseThreatsDTO{
			ActiveSuspiciousAccounts: 12,
			FailedLoginsLast24h:      97,
			ActiveLockouts:           4,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func evaluateAbuseRisk(req abuseLoginRequest) abuseDecision {
	score := 0.1
	if req.FailedAttempts >= 3 {
		score += 0.4
	}
	if !req.KnownDevice {
		score += 0.2
	}
	if req.VPNDetected {
		score += 0.3
	}
	switch {
	case score >= 0.9:
		return abuseDecision{RiskScore: score, RiskTier: "critical", Decision: "block", ChallengeID: "challenge-critical"}
	case score >= 0.7:
		return abuseDecision{RiskScore: score, RiskTier: "high", Decision: "challenge", ChallengeID: "challenge-high"}
	case score >= 0.3:
		return abuseDecision{RiskScore: score, RiskTier: "medium", Decision: "challenge", ChallengeID: "challenge-medium"}
	default:
		return abuseDecision{RiskScore: score, RiskTier: "low", Decision: "allow"}
	}
}
