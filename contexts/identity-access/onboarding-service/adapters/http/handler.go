package httpadapter

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"solomon/contexts/identity-access/onboarding-service/application"
	domainerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	"solomon/contexts/identity-access/onboarding-service/ports"
	httptransport "solomon/contexts/identity-access/onboarding-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) GetFlowHandler(ctx context.Context, userID string) (httptransport.GetFlowResponse, error) {
	item, err := h.Service.GetFlow(ctx, strings.TrimSpace(userID))
	if err != nil {
		return httptransport.GetFlowResponse{}, err
	}
	resp := httptransport.GetFlowResponse{Status: "success"}
	resp.Data.UserID = item.UserID
	resp.Data.Role = item.Role
	resp.Data.FlowID = item.FlowID
	resp.Data.VariantKey = item.VariantKey
	resp.Data.Status = item.Status
	resp.Data.CompletedSteps = item.CompletedSteps
	resp.Data.TotalSteps = item.TotalSteps
	for _, step := range item.Steps {
		resp.Data.Steps = append(resp.Data.Steps, struct {
			StepKey string `json:"step_key"`
			Title   string `json:"title"`
			Status  string `json:"status"`
		}{
			StepKey: step.StepKey,
			Title:   step.Title,
			Status:  step.Status,
		})
	}
	return resp, nil
}

func (h Handler) CompleteStepHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	stepKey string,
	req httptransport.CompleteStepRequest,
) (httptransport.CompleteStepResponse, error) {
	item, err := h.Service.CompleteStep(
		ctx,
		idempotencyKey,
		strings.TrimSpace(userID),
		strings.TrimSpace(stepKey),
		req.Metadata,
	)
	if err != nil {
		return httptransport.CompleteStepResponse{}, err
	}
	resp := httptransport.CompleteStepResponse{Status: "success"}
	resp.Data.StepKey = item.StepKey
	resp.Data.Status = item.Status
	resp.Data.CompletedSteps = item.CompletedSteps
	resp.Data.TotalSteps = item.TotalSteps
	return resp, nil
}

func (h Handler) SkipFlowHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
	req httptransport.SkipFlowRequest,
) (httptransport.SkipFlowResponse, error) {
	item, err := h.Service.SkipFlow(
		ctx,
		idempotencyKey,
		strings.TrimSpace(userID),
		strings.TrimSpace(req.Reason),
	)
	if err != nil {
		return httptransport.SkipFlowResponse{}, err
	}
	resp := httptransport.SkipFlowResponse{Status: "success"}
	resp.Data.Status = item.Status
	resp.Data.ReminderScheduledAt = item.ReminderScheduledAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) ResumeFlowHandler(
	ctx context.Context,
	idempotencyKey string,
	userID string,
) (httptransport.ResumeFlowResponse, error) {
	item, err := h.Service.ResumeFlow(
		ctx,
		idempotencyKey,
		strings.TrimSpace(userID),
	)
	if err != nil {
		return httptransport.ResumeFlowResponse{}, err
	}
	resp := httptransport.ResumeFlowResponse{Status: "success"}
	resp.Data.Status = item.Status
	resp.Data.NextStep = item.NextStep
	return resp, nil
}

func (h Handler) ListAdminFlowsHandler(ctx context.Context) (httptransport.AdminFlowsResponse, error) {
	items, err := h.Service.ListAdminFlows(ctx)
	if err != nil {
		return httptransport.AdminFlowsResponse{}, err
	}
	resp := httptransport.AdminFlowsResponse{Status: "success"}
	for _, item := range items {
		resp.Data.Flows = append(resp.Data.Flows, struct {
			FlowID     string `json:"flow_id"`
			Role       string `json:"role"`
			IsActive   bool   `json:"is_active"`
			StepsCount int    `json:"steps_count"`
		}{
			FlowID:     item.FlowID,
			Role:       item.Role,
			IsActive:   item.IsActive,
			StepsCount: item.StepsCount,
		})
	}
	return resp, nil
}

func (h Handler) ConsumeUserRegisteredEventHandler(
	ctx context.Context,
	req httptransport.UserRegisteredEventRequest,
) (httptransport.UserRegisteredEventResponse, error) {
	occurredAt := time.Time{}
	if strings.TrimSpace(req.OccurredAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.OccurredAt))
		if err != nil {
			return httptransport.UserRegisteredEventResponse{}, domainerrors.ErrInvalidRequest
		}
		occurredAt = parsed.UTC()
	}
	item, err := h.Service.ConsumeUserRegisteredEvent(ctx, ports.UserRegisteredEvent{
		EventID:    strings.TrimSpace(req.EventID),
		UserID:     strings.TrimSpace(req.UserID),
		Role:       strings.ToLower(strings.TrimSpace(req.Role)),
		OccurredAt: occurredAt,
	})
	if err != nil {
		return httptransport.UserRegisteredEventResponse{}, err
	}
	resp := httptransport.UserRegisteredEventResponse{Status: "success"}
	resp.Data.UserID = item.UserID
	resp.Data.Role = item.Role
	resp.Data.FlowID = item.FlowID
	resp.Data.Status = item.Status
	return resp, nil
}
