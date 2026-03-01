package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GetFlowResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID         string `json:"user_id"`
		Role           string `json:"role"`
		FlowID         string `json:"flow_id"`
		VariantKey     string `json:"variant_key"`
		Status         string `json:"status"`
		CompletedSteps int    `json:"completed_steps"`
		TotalSteps     int    `json:"total_steps"`
		Steps          []struct {
			StepKey string `json:"step_key"`
			Title   string `json:"title"`
			Status  string `json:"status"`
		} `json:"steps"`
	} `json:"data"`
}

type CompleteStepRequest struct {
	Metadata map[string]any `json:"metadata,omitempty"`
}

type CompleteStepResponse struct {
	Status string `json:"status"`
	Data   struct {
		StepKey        string `json:"step_key"`
		Status         string `json:"status"`
		CompletedSteps int    `json:"completed_steps"`
		TotalSteps     int    `json:"total_steps"`
	} `json:"data"`
}

type SkipFlowRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SkipFlowResponse struct {
	Status string `json:"status"`
	Data   struct {
		Status              string `json:"status"`
		ReminderScheduledAt string `json:"reminder_scheduled_at"`
	} `json:"data"`
}

type ResumeFlowResponse struct {
	Status string `json:"status"`
	Data   struct {
		Status   string `json:"status"`
		NextStep string `json:"next_step"`
	} `json:"data"`
}

type AdminFlowsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Flows []struct {
			FlowID     string `json:"flow_id"`
			Role       string `json:"role"`
			IsActive   bool   `json:"is_active"`
			StepsCount int    `json:"steps_count"`
		} `json:"flows"`
	} `json:"data"`
}

type UserRegisteredEventRequest struct {
	EventID    string `json:"event_id"`
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type UserRegisteredEventResponse struct {
	Status string `json:"status"`
	Data   struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
		FlowID string `json:"flow_id"`
		Status string `json:"status"`
	} `json:"data"`
}
