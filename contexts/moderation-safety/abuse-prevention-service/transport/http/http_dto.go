package http

type ReleaseLockoutRequest struct {
	Reason        string `json:"reason"`
	SourceIP      string `json:"source_ip,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

type ReleaseLockoutResponse struct {
	ThreatID        string `json:"threat_id"`
	UserID          string `json:"user_id"`
	Status          string `json:"status"`
	ReleasedAt      string `json:"released_at"`
	OwnerAuditLogID string `json:"owner_audit_log_id"`
}
