package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math"
	"strings"
	"time"

	domainerrors "solomon/contexts/finance-core/platform-fee-engine/domain/errors"
	"solomon/contexts/finance-core/platform-fee-engine/ports"
)

type Service struct {
	Repo                              ports.Repository
	Idempotency                       ports.IdempotencyStore
	EventDedup                        ports.EventDedupStore
	Outbox                            ports.OutboxWriter
	Clock                             ports.Clock
	IDGen                             ports.IDGenerator
	IdempotencyTTL                    time.Duration
	EventDedupTTL                     time.Duration
	DefaultFeeRate                    float64
	DisableFeeCalculatedEventEmission bool
	Logger                            *slog.Logger
}

func (s Service) CalculateFee(
	ctx context.Context,
	idempotencyKey string,
	input ports.CalculateFeeInput,
) (ports.FeeCalculation, bool, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return ports.FeeCalculation{}, false, domainerrors.ErrIdempotencyKeyMissing
	}
	if !isValidCalculateInput(input) {
		return ports.FeeCalculation{}, false, domainerrors.ErrInvalidInput
	}

	now := s.now()
	requestHash := hashPayload(map[string]any{
		"submission_id":   strings.TrimSpace(input.SubmissionID),
		"user_id":         strings.TrimSpace(input.UserID),
		"campaign_id":     strings.TrimSpace(input.CampaignID),
		"gross_amount":    round4(input.GrossAmount),
		"fee_rate":        round4(s.resolveFeeRate(input.FeeRate)),
		"source_event_id": strings.TrimSpace(input.SourceEventID),
	})

	record, found, err := s.Idempotency.GetRecord(ctx, strings.TrimSpace(idempotencyKey), now)
	if err != nil {
		return ports.FeeCalculation{}, false, err
	}
	if found {
		if record.RequestHash != requestHash {
			return ports.FeeCalculation{}, false, domainerrors.ErrIdempotencyConflict
		}
		var replayed ports.FeeCalculation
		if err := json.Unmarshal(record.ResponsePayload, &replayed); err != nil {
			return ports.FeeCalculation{}, false, err
		}
		return replayed, true, nil
	}

	calculationID, err := s.IDGen.NewID(ctx)
	if err != nil {
		return ports.FeeCalculation{}, false, err
	}
	calculatedAt := input.CalculatedAt.UTC()
	if calculatedAt.IsZero() {
		calculatedAt = now
	}

	feeRate := s.resolveFeeRate(input.FeeRate)
	gross := round4(input.GrossAmount)
	fee := round4(gross * feeRate)
	net := round4(gross - fee)
	if net < 0 {
		net = 0
	}

	calculation := ports.FeeCalculation{
		CalculationID: strings.TrimSpace(calculationID),
		SubmissionID:  strings.TrimSpace(input.SubmissionID),
		UserID:        strings.TrimSpace(input.UserID),
		CampaignID:    strings.TrimSpace(input.CampaignID),
		GrossAmount:   gross,
		FeeRate:       feeRate,
		FeeAmount:     fee,
		NetAmount:     net,
		CalculatedAt:  calculatedAt,
		SourceEventID: strings.TrimSpace(input.SourceEventID),
	}
	if err := s.Repo.CreateCalculation(ctx, calculation); err != nil {
		return ports.FeeCalculation{}, false, err
	}
	if err := s.appendFeeCalculatedOutbox(ctx, calculation); err != nil {
		return ports.FeeCalculation{}, false, err
	}

	payload, err := json.Marshal(calculation)
	if err != nil {
		return ports.FeeCalculation{}, false, err
	}
	if err := s.Idempotency.PutRecord(ctx, ports.IdempotencyRecord{
		Key:             strings.TrimSpace(idempotencyKey),
		RequestHash:     requestHash,
		ResponsePayload: payload,
		ExpiresAt:       now.Add(s.idempotencyTTL()),
	}); err != nil {
		return ports.FeeCalculation{}, false, err
	}

	resolveLogger(s.Logger).Info("platform fee calculated",
		"event", "platform_fee_calculated",
		"module", "finance-core/platform-fee-engine",
		"layer", "application",
		"calculation_id", calculation.CalculationID,
		"submission_id", calculation.SubmissionID,
		"user_id", calculation.UserID,
		"fee_amount", calculation.FeeAmount,
	)
	return calculation, false, nil
}

func (s Service) ConsumeRewardPayoutEligibleEvent(
	ctx context.Context,
	eventID string,
	event ports.RewardPayoutEligibleEvent,
) (ports.FeeCalculation, bool, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" || !isValidEventInput(event) {
		return ports.FeeCalculation{}, false, domainerrors.ErrInvalidInput
	}

	payloadHash := hashPayload(map[string]any{
		"submission_id": event.SubmissionID,
		"user_id":       event.UserID,
		"campaign_id":   event.CampaignID,
		"gross_amount":  round4(event.GrossAmount),
		"eligible_at":   event.EligibleAt.UTC().Format(time.RFC3339Nano),
	})

	if s.EventDedup != nil {
		alreadyProcessed, err := s.EventDedup.ReserveEvent(ctx, eventID, payloadHash, s.now().Add(s.eventDedupTTL()))
		if err != nil {
			return ports.FeeCalculation{}, false, err
		}
		if alreadyProcessed {
			// Idempotency store replays the original calculation response.
			return s.CalculateFee(ctx, "event:"+eventID, ports.CalculateFeeInput{
				SubmissionID:  event.SubmissionID,
				UserID:        event.UserID,
				CampaignID:    event.CampaignID,
				GrossAmount:   event.GrossAmount,
				CalculatedAt:  event.EligibleAt,
				SourceEventID: eventID,
			})
		}
	}

	return s.CalculateFee(ctx, "event:"+eventID, ports.CalculateFeeInput{
		SubmissionID:  event.SubmissionID,
		UserID:        event.UserID,
		CampaignID:    event.CampaignID,
		GrossAmount:   event.GrossAmount,
		CalculatedAt:  event.EligibleAt,
		SourceEventID: eventID,
	})
}

func (s Service) ListHistory(
	ctx context.Context,
	userID string,
	limit int,
	offset int,
) ([]ports.FeeCalculation, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, domainerrors.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.Repo.ListCalculationsByUser(ctx, strings.TrimSpace(userID), limit, offset)
}

func (s Service) MonthlyReport(ctx context.Context, month string) (ports.FeeReport, error) {
	month = strings.TrimSpace(month)
	if len(month) != len("2006-01") {
		return ports.FeeReport{}, domainerrors.ErrInvalidInput
	}
	return s.Repo.BuildMonthlyReport(ctx, month)
}

func (s Service) appendFeeCalculatedOutbox(ctx context.Context, calculation ports.FeeCalculation) error {
	if s.Outbox == nil || s.DisableFeeCalculatedEventEmission {
		return nil
	}
	eventID, err := s.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(map[string]any{
		"calculation_id":  calculation.CalculationID,
		"submission_id":   calculation.SubmissionID,
		"user_id":         calculation.UserID,
		"campaign_id":     calculation.CampaignID,
		"gross_amount":    calculation.GrossAmount,
		"fee_rate":        calculation.FeeRate,
		"fee_amount":      calculation.FeeAmount,
		"net_amount":      calculation.NetAmount,
		"calculated_at":   calculation.CalculatedAt.UTC().Format(time.RFC3339),
		"source_event_id": calculation.SourceEventID,
	})
	if err != nil {
		return err
	}
	return s.Outbox.AppendOutbox(ctx, ports.EventEnvelope{
		EventID:          strings.TrimSpace(eventID),
		EventType:        "fee.calculated",
		OccurredAt:       calculation.CalculatedAt.UTC(),
		SourceService:    "platform-fee-engine",
		TraceID:          strings.TrimSpace(eventID),
		SchemaVersion:    1,
		PartitionKeyPath: "submission_id",
		PartitionKey:     calculation.SubmissionID,
		Data:             data,
	})
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) eventDedupTTL() time.Duration {
	if s.EventDedupTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.EventDedupTTL
}

func (s Service) resolveFeeRate(requested float64) float64 {
	rate := requested
	if rate <= 0 {
		rate = s.DefaultFeeRate
	}
	if rate <= 0 {
		rate = 0.15
	}
	if rate > 1 {
		rate = 1
	}
	return round4(rate)
}

func isValidCalculateInput(input ports.CalculateFeeInput) bool {
	return strings.TrimSpace(input.SubmissionID) != "" &&
		strings.TrimSpace(input.UserID) != "" &&
		strings.TrimSpace(input.CampaignID) != "" &&
		input.GrossAmount > 0
}

func isValidEventInput(input ports.RewardPayoutEligibleEvent) bool {
	return strings.TrimSpace(input.SubmissionID) != "" &&
		strings.TrimSpace(input.UserID) != "" &&
		strings.TrimSpace(input.CampaignID) != "" &&
		input.GrossAmount > 0
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func hashPayload(payload map[string]any) string {
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
