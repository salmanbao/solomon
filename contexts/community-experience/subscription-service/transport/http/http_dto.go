package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateSubscriptionRequest struct {
	PlanID string `json:"plan_id"`
	Trial  bool   `json:"trial"`
}

type CreateSubscriptionResponse struct {
	Status string `json:"status"`
	Data   struct {
		SubscriptionID  string `json:"subscription_id"`
		PlanName        string `json:"plan_name"`
		Status          string `json:"status"`
		TrialEnd        string `json:"trial_end,omitempty"`
		NextBillingDate string `json:"next_billing_date,omitempty"`
		AmountCents     int64  `json:"amount_cents"`
		Currency        string `json:"currency"`
	} `json:"data"`
}

type ChangePlanRequest struct {
	NewPlanID string `json:"new_plan_id"`
}

type ChangePlanResponse struct {
	Status string `json:"status"`
	Data   struct {
		SubscriptionID       string `json:"subscription_id"`
		OldPlan              string `json:"old_plan"`
		NewPlan              string `json:"new_plan"`
		ProrationAmountCents int64  `json:"proration_amount_cents"`
		ProrationDescription string `json:"proration_description"`
		NextBillingDate      string `json:"next_billing_date,omitempty"`
		ChangedAt            string `json:"changed_at"`
	} `json:"data"`
}

type CancelSubscriptionRequest struct {
	CancelAtPeriodEnd    *bool  `json:"cancel_at_period_end,omitempty"`
	CancellationFeedback string `json:"cancellation_feedback,omitempty"`
}

type CancelSubscriptionResponse struct {
	Status string `json:"status"`
	Data   struct {
		SubscriptionID       string `json:"subscription_id"`
		Status               string `json:"status"`
		CancelAtPeriodEnd    bool   `json:"cancel_at_period_end"`
		AccessEndsAt         string `json:"access_ends_at,omitempty"`
		CancellationFeedback string `json:"cancellation_feedback,omitempty"`
		CanceledAt           string `json:"canceled_at,omitempty"`
	} `json:"data"`
}
