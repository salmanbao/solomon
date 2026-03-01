package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/community-experience/subscription-service/application"
	"solomon/contexts/community-experience/subscription-service/ports"
	httptransport "solomon/contexts/community-experience/subscription-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) CreateSubscriptionHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	req httptransport.CreateSubscriptionRequest,
) (httptransport.CreateSubscriptionResponse, error) {
	item, err := h.Service.CreateSubscription(ctx, idempotencyKey, userID, toCreateInput(req))
	if err != nil {
		return httptransport.CreateSubscriptionResponse{}, err
	}

	resp := httptransport.CreateSubscriptionResponse{Status: "success"}
	resp.Data.SubscriptionID = item.SubscriptionID
	resp.Data.PlanName = item.PlanName
	resp.Data.Status = item.Status
	resp.Data.AmountCents = item.AmountCents
	resp.Data.Currency = item.Currency
	if item.TrialEnd != nil {
		resp.Data.TrialEnd = item.TrialEnd.UTC().Format(time.RFC3339)
	}
	if item.NextBillingDate != nil {
		resp.Data.NextBillingDate = item.NextBillingDate.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) ChangePlanHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	subscriptionID string,
	req httptransport.ChangePlanRequest,
) (httptransport.ChangePlanResponse, error) {
	item, err := h.Service.ChangePlan(
		ctx,
		idempotencyKey,
		userID,
		strings.TrimSpace(subscriptionID),
		strings.TrimSpace(req.NewPlanID),
	)
	if err != nil {
		return httptransport.ChangePlanResponse{}, err
	}

	resp := httptransport.ChangePlanResponse{Status: "success"}
	resp.Data.SubscriptionID = item.SubscriptionID
	resp.Data.OldPlan = item.OldPlanName
	resp.Data.NewPlan = item.NewPlanName
	resp.Data.ProrationAmountCents = item.ProrationAmountCents
	resp.Data.ProrationDescription = item.ProrationDescription
	resp.Data.ChangedAt = item.ChangedAt.UTC().Format(time.RFC3339)
	if item.NextBillingDate != nil {
		resp.Data.NextBillingDate = item.NextBillingDate.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) CancelSubscriptionHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	subscriptionID string,
	req httptransport.CancelSubscriptionRequest,
) (httptransport.CancelSubscriptionResponse, error) {
	cancelAtPeriodEnd := true
	if req.CancelAtPeriodEnd != nil {
		cancelAtPeriodEnd = *req.CancelAtPeriodEnd
	}

	item, err := h.Service.CancelSubscription(
		ctx,
		idempotencyKey,
		userID,
		strings.TrimSpace(subscriptionID),
		cancelAtPeriodEnd,
		strings.TrimSpace(req.CancellationFeedback),
	)
	if err != nil {
		return httptransport.CancelSubscriptionResponse{}, err
	}

	resp := httptransport.CancelSubscriptionResponse{Status: "success"}
	resp.Data.SubscriptionID = item.SubscriptionID
	resp.Data.Status = item.Status
	resp.Data.CancelAtPeriodEnd = item.CancelAtPeriodEnd
	resp.Data.CancellationFeedback = item.CancellationFeedback
	if item.AccessEndsAt != nil {
		resp.Data.AccessEndsAt = item.AccessEndsAt.UTC().Format(time.RFC3339)
	}
	if item.CanceledAt != nil {
		resp.Data.CanceledAt = item.CanceledAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func toCreateInput(req httptransport.CreateSubscriptionRequest) ports.CreateSubscriptionInput {
	return ports.CreateSubscriptionInput{
		PlanID: strings.TrimSpace(req.PlanID),
		Trial:  req.Trial,
	}
}
