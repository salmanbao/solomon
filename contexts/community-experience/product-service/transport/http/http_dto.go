package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ProductDTO struct {
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	ProductType   string `json:"product_type"`
	PricingModel  string `json:"pricing_model"`
	PriceCents    int64  `json:"price_cents"`
	Currency      string `json:"currency"`
	CoverImageURL string `json:"cover_image_url,omitempty"`
	Creator       struct {
		UserID      string `json:"user_id"`
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	SalesCount int64   `json:"sales_count"`
	Rating     float64 `json:"rating"`
	CreatedAt  string  `json:"created_at"`
	Visibility string  `json:"visibility,omitempty"`
	Status     string  `json:"status,omitempty"`
}

type ListProductsRequest struct {
	CreatorID   string
	ProductType string
	Visibility  string
	Page        int
	Limit       int
}

type ListProductsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Products   []ProductDTO `json:"products"`
		Pagination struct {
			Page  int `json:"page"`
			Limit int `json:"limit"`
			Total int `json:"total"`
			Pages int `json:"pages"`
		} `json:"pagination"`
	} `json:"data"`
}

type CreateProductRequest struct {
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	ProductType         string         `json:"product_type"`
	PricingModel        string         `json:"pricing_model"`
	PriceCents          int64          `json:"price_cents"`
	Currency            string         `json:"currency,omitempty"`
	Category            string         `json:"category,omitempty"`
	FulfillmentMetadata map[string]any `json:"fulfillment_metadata,omitempty"`
	MetaTitle           string         `json:"meta_title,omitempty"`
	MetaDescription     string         `json:"meta_description,omitempty"`
	Visibility          string         `json:"visibility,omitempty"`
}

type CreateProductResponse struct {
	Status string `json:"status"`
	Data   struct {
		ProductID string `json:"product_id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	} `json:"data"`
}

type CheckAccessResponse struct {
	Status string `json:"status"`
	Data   struct {
		HasAccess  bool   `json:"has_access"`
		AccessType string `json:"access_type,omitempty"`
		GrantedAt  string `json:"granted_at,omitempty"`
		ExpiresAt  string `json:"expires_at,omitempty"`
	} `json:"data"`
}

type PurchaseProductResponse struct {
	PurchaseID        string `json:"purchase_id"`
	ProductID         string `json:"product_id"`
	Status            string `json:"status"`
	FulfillmentStatus string `json:"fulfillment_status"`
	CreatedAt         string `json:"created_at"`
	Replayed          bool   `json:"replayed,omitempty"`
}

type FulfillProductResponse struct {
	PurchaseID      string `json:"purchase_id"`
	ProductID       string `json:"product_id"`
	Status          string `json:"status"`
	FulfillmentType string `json:"fulfillment_type"`
	ProcessedAt     string `json:"processed_at"`
	Replayed        bool   `json:"replayed,omitempty"`
}

type AdjustInventoryRequest struct {
	NewCount int    `json:"new_count"`
	Reason   string `json:"reason"`
}

type AdjustInventoryResponse struct {
	ProductID string `json:"product_id"`
	OldCount  int    `json:"old_count"`
	NewCount  int    `json:"new_count"`
	ChangedBy string `json:"changed_by"`
	Reason    string `json:"reason"`
	ChangedAt string `json:"changed_at"`
	Replayed  bool   `json:"replayed,omitempty"`
}

type ReorderMediaRequest struct {
	MediaIDs []string `json:"media_ids"`
}

type ReorderMediaResponse struct {
	ProductID  string   `json:"product_id"`
	MediaOrder []string `json:"media_order"`
	UpdatedAt  string   `json:"updated_at"`
	Replayed   bool     `json:"replayed,omitempty"`
}

type DiscoverProductsResponse struct {
	Products []ProductDTO `json:"products"`
}

type SearchProductsResponse struct {
	Products []ProductDTO `json:"products"`
}

type UserDataExportResponse struct {
	UserID        string `json:"user_id"`
	PurchaseCount int    `json:"purchase_count"`
	AccessCount   int    `json:"access_count"`
	GeneratedAt   string `json:"generated_at"`
}

type DeleteAccountResponse struct {
	UserID             string `json:"user_id"`
	RevokedAccessCount int    `json:"revoked_access_count"`
	AnonymizedCount    int    `json:"anonymized_count"`
	ProcessedAt        string `json:"processed_at"`
	Replayed           bool   `json:"replayed,omitempty"`
}
