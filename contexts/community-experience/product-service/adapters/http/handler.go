package httpadapter

import (
	"context"
	"log/slog"
	"time"

	"solomon/contexts/community-experience/product-service/application"
	"solomon/contexts/community-experience/product-service/ports"
	httptransport "solomon/contexts/community-experience/product-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) ListProductsHandler(ctx context.Context, req httptransport.ListProductsRequest) (httptransport.ListProductsResponse, error) {
	items, total, err := h.Service.ListProducts(ctx, ports.ProductFilter{
		CreatorID:   req.CreatorID,
		ProductType: req.ProductType,
		Visibility:  req.Visibility,
		Page:        req.Page,
		Limit:       req.Limit,
	})
	if err != nil {
		return httptransport.ListProductsResponse{}, err
	}

	resp := httptransport.ListProductsResponse{Status: "success"}
	resp.Data.Products = make([]httptransport.ProductDTO, 0, len(items))
	for _, item := range items {
		resp.Data.Products = append(resp.Data.Products, toProductDTO(item))
	}
	resp.Data.Pagination.Page = req.Page
	if resp.Data.Pagination.Page <= 0 {
		resp.Data.Pagination.Page = 1
	}
	resp.Data.Pagination.Limit = req.Limit
	if resp.Data.Pagination.Limit <= 0 {
		resp.Data.Pagination.Limit = 20
	}
	resp.Data.Pagination.Total = total
	resp.Data.Pagination.Pages = total / resp.Data.Pagination.Limit
	if total%resp.Data.Pagination.Limit != 0 {
		resp.Data.Pagination.Pages++
	}
	if resp.Data.Pagination.Pages == 0 {
		resp.Data.Pagination.Pages = 1
	}
	return resp, nil
}

func (h Handler) CreateProductHandler(
	ctx context.Context,
	creatorID string,
	idempotencyKey string,
	req httptransport.CreateProductRequest,
) (httptransport.CreateProductResponse, error) {
	product, err := h.Service.CreateProduct(ctx, idempotencyKey, ports.CreateProductInput{
		CreatorID:           creatorID,
		Name:                req.Name,
		Description:         req.Description,
		ProductType:         req.ProductType,
		PricingModel:        req.PricingModel,
		PriceCents:          req.PriceCents,
		Currency:            req.Currency,
		Category:            req.Category,
		Visibility:          req.Visibility,
		MetaTitle:           req.MetaTitle,
		MetaDescription:     req.MetaDescription,
		FulfillmentMetadata: req.FulfillmentMetadata,
	})
	if err != nil {
		return httptransport.CreateProductResponse{}, err
	}

	resp := httptransport.CreateProductResponse{Status: "success"}
	resp.Data.ProductID = product.ProductID
	resp.Data.Name = product.Name
	resp.Data.Status = product.Status
	resp.Data.CreatedAt = product.CreatedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) CheckAccessHandler(ctx context.Context, userID string, productID string) (httptransport.CheckAccessResponse, error) {
	access, hasAccess, err := h.Service.CheckAccess(ctx, userID, productID)
	if err != nil {
		return httptransport.CheckAccessResponse{}, err
	}
	resp := httptransport.CheckAccessResponse{Status: "success"}
	resp.Data.HasAccess = hasAccess
	if hasAccess {
		resp.Data.AccessType = access.AccessType
		resp.Data.GrantedAt = access.GrantedAt.UTC().Format(time.RFC3339)
		if access.ExpiresAt != nil {
			resp.Data.ExpiresAt = access.ExpiresAt.UTC().Format(time.RFC3339)
		}
	}
	return resp, nil
}

func (h Handler) PurchaseProductHandler(
	ctx context.Context,
	userID string,
	idempotencyKey string,
	productID string,
) (httptransport.PurchaseProductResponse, error) {
	result, err := h.Service.PurchaseProduct(ctx, idempotencyKey, userID, productID)
	if err != nil {
		return httptransport.PurchaseProductResponse{}, err
	}
	return httptransport.PurchaseProductResponse{
		PurchaseID:        result.PurchaseID,
		ProductID:         result.ProductID,
		Status:            result.Status,
		FulfillmentStatus: result.FulfillmentStatus,
		CreatedAt:         result.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) FulfillProductHandler(
	ctx context.Context,
	userID string,
	idempotencyKey string,
	productID string,
) (httptransport.FulfillProductResponse, error) {
	result, err := h.Service.FulfillProduct(ctx, idempotencyKey, userID, productID)
	if err != nil {
		return httptransport.FulfillProductResponse{}, err
	}
	return httptransport.FulfillProductResponse{
		PurchaseID:      result.PurchaseID,
		ProductID:       result.ProductID,
		Status:          result.Status,
		FulfillmentType: result.FulfillmentType,
		ProcessedAt:     result.ProcessedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) AdjustInventoryHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	productID string,
	req httptransport.AdjustInventoryRequest,
) (httptransport.AdjustInventoryResponse, error) {
	result, err := h.Service.AdjustInventory(ctx, idempotencyKey, adminID, productID, req.NewCount, req.Reason)
	if err != nil {
		return httptransport.AdjustInventoryResponse{}, err
	}
	return httptransport.AdjustInventoryResponse{
		ProductID: result.ProductID,
		OldCount:  result.OldCount,
		NewCount:  result.NewCount,
		ChangedBy: result.ChangedBy,
		Reason:    result.Reason,
		ChangedAt: result.ChangedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) ReorderMediaHandler(
	ctx context.Context,
	idempotencyKey string,
	productID string,
	req httptransport.ReorderMediaRequest,
) (httptransport.ReorderMediaResponse, error) {
	result, err := h.Service.ReorderMedia(ctx, idempotencyKey, productID, req.MediaIDs)
	if err != nil {
		return httptransport.ReorderMediaResponse{}, err
	}
	return httptransport.ReorderMediaResponse{
		ProductID:  result.ProductID,
		MediaOrder: append([]string(nil), result.MediaOrder...),
		UpdatedAt:  result.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) DiscoverProductsHandler(ctx context.Context, limit int) (httptransport.DiscoverProductsResponse, error) {
	items, err := h.Service.DiscoverProducts(ctx, limit)
	if err != nil {
		return httptransport.DiscoverProductsResponse{}, err
	}
	resp := httptransport.DiscoverProductsResponse{
		Products: make([]httptransport.ProductDTO, 0, len(items)),
	}
	for _, item := range items {
		resp.Products = append(resp.Products, toProductDTO(item))
	}
	return resp, nil
}

func (h Handler) SearchProductsHandler(ctx context.Context, query string, productType string, limit int) (httptransport.SearchProductsResponse, error) {
	items, err := h.Service.SearchProducts(ctx, query, productType, limit)
	if err != nil {
		return httptransport.SearchProductsResponse{}, err
	}
	resp := httptransport.SearchProductsResponse{
		Products: make([]httptransport.ProductDTO, 0, len(items)),
	}
	for _, item := range items {
		resp.Products = append(resp.Products, toProductDTO(item))
	}
	return resp, nil
}

func (h Handler) ExportUserDataHandler(ctx context.Context, userID string) (httptransport.UserDataExportResponse, error) {
	result, err := h.Service.ExportUserData(ctx, userID)
	if err != nil {
		return httptransport.UserDataExportResponse{}, err
	}
	return httptransport.UserDataExportResponse{
		UserID:        result.UserID,
		PurchaseCount: len(result.Purchases),
		AccessCount:   len(result.AccessRecords),
		GeneratedAt:   result.GeneratedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h Handler) DeleteUserDataHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
) (httptransport.DeleteAccountResponse, error) {
	result, err := h.Service.DeleteUserData(ctx, idempotencyKey, userID)
	if err != nil {
		return httptransport.DeleteAccountResponse{}, err
	}
	return httptransport.DeleteAccountResponse{
		UserID:             result.UserID,
		RevokedAccessCount: result.RevokedAccessCount,
		AnonymizedCount:    result.AnonymizedCount,
		ProcessedAt:        result.ProcessedAt.UTC().Format(time.RFC3339),
	}, nil
}

func toProductDTO(item ports.Product) httptransport.ProductDTO {
	dto := httptransport.ProductDTO{
		ProductID:     item.ProductID,
		Name:          item.Name,
		Description:   item.Description,
		ProductType:   item.ProductType,
		PricingModel:  item.PricingModel,
		PriceCents:    item.PriceCents,
		Currency:      item.Currency,
		CoverImageURL: item.CoverImageURL,
		SalesCount:    item.SalesCount,
		Rating:        item.Rating,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		Visibility:    item.Visibility,
		Status:        item.Status,
	}
	dto.Creator.UserID = item.CreatorID
	dto.Creator.DisplayName = item.CreatorID
	return dto
}
