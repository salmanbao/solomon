package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CalculateFeeRequest struct {
	SubmissionID string  `json:"submission_id"`
	UserID       string  `json:"user_id"`
	CampaignID   string  `json:"campaign_id"`
	GrossAmount  float64 `json:"gross_amount"`
	FeeRate      float64 `json:"fee_rate,omitempty"`
}

type FeeCalculationDTO struct {
	CalculationID string  `json:"calculation_id"`
	SubmissionID  string  `json:"submission_id"`
	UserID        string  `json:"user_id"`
	CampaignID    string  `json:"campaign_id"`
	GrossAmount   float64 `json:"gross_amount"`
	FeeRate       float64 `json:"fee_rate"`
	FeeAmount     float64 `json:"fee_amount"`
	NetAmount     float64 `json:"net_amount"`
	CalculatedAt  string  `json:"calculated_at"`
	SourceEventID string  `json:"source_event_id,omitempty"`
}

type CalculateFeeResponse struct {
	Status   string            `json:"status"`
	Replayed bool              `json:"replayed,omitempty"`
	Data     FeeCalculationDTO `json:"data"`
}

type FeeHistoryRequest struct {
	UserID string
	Limit  int
	Offset int
}

type FeeHistoryResponse struct {
	Status string              `json:"status"`
	Data   []FeeCalculationDTO `json:"data"`
}

type FeeReportRequest struct {
	Month string
}

type FeeReportResponse struct {
	Status string `json:"status"`
	Data   struct {
		Month      string  `json:"month"`
		Count      int     `json:"count"`
		TotalGross float64 `json:"total_gross"`
		TotalFee   float64 `json:"total_fee"`
		TotalNet   float64 `json:"total_net"`
	} `json:"data"`
}

type RewardPayoutEligibleEventRequest struct {
	EventID      string  `json:"event_id"`
	SubmissionID string  `json:"submission_id"`
	UserID       string  `json:"user_id"`
	CampaignID   string  `json:"campaign_id"`
	GrossAmount  float64 `json:"gross_amount"`
	EligibleAt   string  `json:"eligible_at"`
}
