package http

type RecordAdminActionRequest struct {
	Action        string `json:"action"`
	TargetID      string `json:"target_id"`
	Justification string `json:"justification"`
	SourceIP      string `json:"source_ip"`
	CorrelationID string `json:"correlation_id"`
}

type RecordAdminActionResponse struct {
	AuditID    string `json:"audit_id"`
	OccurredAt string `json:"occurred_at"`
}
