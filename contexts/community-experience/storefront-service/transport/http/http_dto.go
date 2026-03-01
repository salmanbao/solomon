package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateStorefrontRequest struct {
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
}

type StorefrontResponse struct {
	Status string `json:"status"`
	Data   struct {
		StorefrontID     string   `json:"storefront_id"`
		CreatorUserID    string   `json:"creator_user_id"`
		Subdomain        string   `json:"subdomain"`
		DisplayName      string   `json:"display_name"`
		Headline         string   `json:"headline,omitempty"`
		Bio              string   `json:"bio,omitempty"`
		Status           string   `json:"status"`
		Category         string   `json:"category"`
		VisibilityMode   string   `json:"visibility_mode"`
		DiscoverEligible bool     `json:"discover_eligible"`
		DiscoverReasons  []string `json:"discover_reasons,omitempty"`
		CreatedAt        string   `json:"created_at"`
		UpdatedAt        string   `json:"updated_at"`
	} `json:"data"`
}

type UpdateStorefrontRequest struct {
	Headline       string `json:"headline,omitempty"`
	Bio            string `json:"bio,omitempty"`
	VisibilityMode string `json:"visibility_mode,omitempty"`
	Password       string `json:"password,omitempty"`
}

type ReportStorefrontRequest struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

type ReportStorefrontResponse struct {
	Status string `json:"status"`
	Data   struct {
		StorefrontID string `json:"storefront_id"`
		Status       string `json:"status"`
		ReportedAt   string `json:"reported_at"`
	} `json:"data"`
}

type ProductPublishedEventRequest struct {
	EventID      string `json:"event_id"`
	StorefrontID string `json:"storefront_id"`
	ProductID    string `json:"product_id"`
	OccurredAt   string `json:"occurred_at,omitempty"`
}

type ProductPublishedEventResponse struct {
	Status string `json:"status"`
	Data   struct {
		StorefrontID string `json:"storefront_id"`
		ProductID    string `json:"product_id"`
		Accepted     bool   `json:"accepted"`
	} `json:"data"`
}

type SubscriptionProjectionRequest struct {
	UserID string `json:"user_id"`
	Active bool   `json:"active"`
}

type GenericAcceptedResponse struct {
	Status string `json:"status"`
}
