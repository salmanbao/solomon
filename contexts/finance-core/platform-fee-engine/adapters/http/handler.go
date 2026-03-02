package httpadapter

import (
	"context"
	"log/slog"
	"time"

	"solomon/contexts/finance-core/platform-fee-engine/application"
	"solomon/contexts/finance-core/platform-fee-engine/ports"
	httptransport "solomon/contexts/finance-core/platform-fee-engine/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) CalculateFeeHandler(
	ctx context.Context,
	idempotencyKey string,
	req httptransport.CalculateFeeRequest,
) (httptransport.CalculateFeeResponse, error) {
	calculation, replayed, err := h.Service.CalculateFee(ctx, idempotencyKey, ports.CalculateFeeInput{
		SubmissionID: req.SubmissionID,
		UserID:       req.UserID,
		CampaignID:   req.CampaignID,
		GrossAmount:  req.GrossAmount,
		FeeRate:      req.FeeRate,
	})
	if err != nil {
		return httptransport.CalculateFeeResponse{}, err
	}
	return httptransport.CalculateFeeResponse{
		Status:   "success",
		Replayed: replayed,
		Data:     toDTO(calculation),
	}, nil
}

func (h Handler) ConsumeRewardPayoutEligibleEventHandler(
	ctx context.Context,
	req httptransport.RewardPayoutEligibleEventRequest,
) (httptransport.CalculateFeeResponse, error) {
	eligibleAt, _ := time.Parse(time.RFC3339, req.EligibleAt)
	calculation, replayed, err := h.Service.ConsumeRewardPayoutEligibleEvent(ctx, req.EventID, ports.RewardPayoutEligibleEvent{
		SubmissionID: req.SubmissionID,
		UserID:       req.UserID,
		CampaignID:   req.CampaignID,
		GrossAmount:  req.GrossAmount,
		EligibleAt:   eligibleAt,
	})
	if err != nil {
		return httptransport.CalculateFeeResponse{}, err
	}
	return httptransport.CalculateFeeResponse{
		Status:   "success",
		Replayed: replayed,
		Data:     toDTO(calculation),
	}, nil
}

func (h Handler) ListHistoryHandler(
	ctx context.Context,
	req httptransport.FeeHistoryRequest,
) (httptransport.FeeHistoryResponse, error) {
	items, err := h.Service.ListHistory(ctx, req.UserID, req.Limit, req.Offset)
	if err != nil {
		return httptransport.FeeHistoryResponse{}, err
	}
	resp := httptransport.FeeHistoryResponse{
		Status: "success",
		Data:   make([]httptransport.FeeCalculationDTO, 0, len(items)),
	}
	for _, item := range items {
		resp.Data = append(resp.Data, toDTO(item))
	}
	return resp, nil
}

func (h Handler) MonthlyReportHandler(
	ctx context.Context,
	req httptransport.FeeReportRequest,
) (httptransport.FeeReportResponse, error) {
	report, err := h.Service.MonthlyReport(ctx, req.Month)
	if err != nil {
		return httptransport.FeeReportResponse{}, err
	}
	resp := httptransport.FeeReportResponse{Status: "success"}
	resp.Data.Month = report.Month
	resp.Data.Count = report.Count
	resp.Data.TotalGross = report.TotalGross
	resp.Data.TotalFee = report.TotalFee
	resp.Data.TotalNet = report.TotalNet
	return resp, nil
}

func toDTO(calculation ports.FeeCalculation) httptransport.FeeCalculationDTO {
	return httptransport.FeeCalculationDTO{
		CalculationID: calculation.CalculationID,
		SubmissionID:  calculation.SubmissionID,
		UserID:        calculation.UserID,
		CampaignID:    calculation.CampaignID,
		GrossAmount:   calculation.GrossAmount,
		FeeRate:       calculation.FeeRate,
		FeeAmount:     calculation.FeeAmount,
		NetAmount:     calculation.NetAmount,
		CalculatedAt:  calculation.CalculatedAt.UTC().Format(time.RFC3339),
		SourceEventID: calculation.SourceEventID,
	}
}
