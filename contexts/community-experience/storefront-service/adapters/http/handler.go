package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/community-experience/storefront-service/application"
	domainerrors "solomon/contexts/community-experience/storefront-service/domain/errors"
	"solomon/contexts/community-experience/storefront-service/ports"
	httptransport "solomon/contexts/community-experience/storefront-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) CreateStorefrontHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	req httptransport.CreateStorefrontRequest,
) (httptransport.StorefrontResponse, error) {
	item, err := h.Service.CreateStorefront(ctx, idempotencyKey, actorUserID, ports.CreateStorefrontInput{
		DisplayName: strings.TrimSpace(req.DisplayName),
		Category:    strings.TrimSpace(req.Category),
	})
	if err != nil {
		return httptransport.StorefrontResponse{}, err
	}
	return toStorefrontResponse(item), nil
}

func (h Handler) UpdateStorefrontHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
	req httptransport.UpdateStorefrontRequest,
) (httptransport.StorefrontResponse, error) {
	item, err := h.Service.UpdateStorefront(ctx, idempotencyKey, actorUserID, strings.TrimSpace(storefrontID), ports.UpdateStorefrontInput{
		Headline:       strings.TrimSpace(req.Headline),
		Bio:            strings.TrimSpace(req.Bio),
		VisibilityMode: strings.TrimSpace(req.VisibilityMode),
		Password:       req.Password,
	})
	if err != nil {
		return httptransport.StorefrontResponse{}, err
	}
	return toStorefrontResponse(item), nil
}

func (h Handler) GetStorefrontByIDHandler(
	ctx context.Context,
	storefrontID string,
	actorUserID string,
) (httptransport.StorefrontResponse, error) {
	item, err := h.Service.GetStorefrontByID(ctx, strings.TrimSpace(storefrontID), strings.TrimSpace(actorUserID))
	if err != nil {
		return httptransport.StorefrontResponse{}, err
	}
	return toStorefrontResponse(item), nil
}

func (h Handler) GetStorefrontBySlugHandler(
	ctx context.Context,
	slug string,
) (httptransport.StorefrontResponse, error) {
	item, err := h.Service.GetStorefrontBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return httptransport.StorefrontResponse{}, err
	}
	return toStorefrontResponse(item), nil
}

func (h Handler) PublishStorefrontHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
) (httptransport.StorefrontResponse, error) {
	item, err := h.Service.PublishStorefront(ctx, idempotencyKey, strings.TrimSpace(actorUserID), strings.TrimSpace(storefrontID))
	if err != nil {
		return httptransport.StorefrontResponse{}, err
	}
	return toStorefrontResponse(item), nil
}

func (h Handler) ReportStorefrontHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	storefrontID string,
	req httptransport.ReportStorefrontRequest,
) (httptransport.ReportStorefrontResponse, error) {
	item, err := h.Service.ReportStorefront(ctx, idempotencyKey, strings.TrimSpace(actorUserID), strings.TrimSpace(storefrontID), ports.ReportInput{
		Type:   strings.TrimSpace(req.Type),
		Reason: strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return httptransport.ReportStorefrontResponse{}, err
	}
	resp := httptransport.ReportStorefrontResponse{Status: "success"}
	resp.Data.StorefrontID = item.StorefrontID
	resp.Data.Status = item.Status
	resp.Data.ReportedAt = item.ReportedAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) ConsumeProductPublishedEventHandler(
	ctx context.Context,
	req httptransport.ProductPublishedEventRequest,
) (httptransport.ProductPublishedEventResponse, error) {
	occurredAt := time.Time{}
	if strings.TrimSpace(req.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.OccurredAt))
		if err != nil {
			return httptransport.ProductPublishedEventResponse{}, domainerrors.ErrInvalidRequest
		}
		occurredAt = parsed.UTC()
	}
	item, err := h.Service.ConsumeProductPublishedEvent(ctx, ports.ProductPublishedEvent{
		EventID:      strings.TrimSpace(req.EventID),
		StorefrontID: strings.TrimSpace(req.StorefrontID),
		ProductID:    strings.TrimSpace(req.ProductID),
		OccurredAt:   occurredAt,
	})
	if err != nil {
		return httptransport.ProductPublishedEventResponse{}, err
	}
	resp := httptransport.ProductPublishedEventResponse{Status: "success"}
	resp.Data.StorefrontID = item.StorefrontID
	resp.Data.ProductID = item.ProductID
	resp.Data.Accepted = item.Accepted
	return resp, nil
}

func (h Handler) UpsertSubscriptionProjectionHandler(
	ctx context.Context,
	req httptransport.SubscriptionProjectionRequest,
) (httptransport.GenericAcceptedResponse, error) {
	err := h.Service.UpsertSubscriptionProjection(ctx, ports.SubscriptionProjectionInput{
		UserID: strings.TrimSpace(req.UserID),
		Active: req.Active,
	})
	if err != nil {
		return httptransport.GenericAcceptedResponse{}, err
	}
	return httptransport.GenericAcceptedResponse{Status: "success"}, nil
}

func toStorefrontResponse(item ports.Storefront) httptransport.StorefrontResponse {
	resp := httptransport.StorefrontResponse{Status: "success"}
	resp.Data.StorefrontID = item.StorefrontID
	resp.Data.CreatorUserID = item.CreatorUserID
	resp.Data.Subdomain = item.Subdomain
	resp.Data.DisplayName = item.DisplayName
	resp.Data.Headline = item.Headline
	resp.Data.Bio = item.Bio
	resp.Data.Status = item.Status
	resp.Data.Category = item.Category
	resp.Data.VisibilityMode = item.VisibilityMode
	resp.Data.DiscoverEligible = item.DiscoverEligible
	resp.Data.DiscoverReasons = append([]string(nil), item.DiscoverReasons...)
	resp.Data.CreatedAt = item.CreatedAt.UTC().Format(time.RFC3339)
	resp.Data.UpdatedAt = item.UpdatedAt.UTC().Format(time.RFC3339)
	return resp
}
