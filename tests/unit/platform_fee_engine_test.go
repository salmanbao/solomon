package unit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	platformfeeengine "solomon/contexts/finance-core/platform-fee-engine"
	"solomon/contexts/finance-core/platform-fee-engine/adapters/memory"
	httptransport "solomon/contexts/finance-core/platform-fee-engine/transport/http"
)

func TestPlatformFeeCalculateIdempotencyReplay(t *testing.T) {
	module := platformfeeengine.NewInMemoryModule(nil)
	ctx := context.Background()

	first, err := module.Handler.CalculateFeeHandler(ctx, "idem-fee-1", httptransport.CalculateFeeRequest{
		SubmissionID: "sub-fee-1",
		UserID:       "user-fee-1",
		CampaignID:   "campaign-fee-1",
		GrossAmount:  10.0,
		FeeRate:      0.2,
	})
	if err != nil {
		t.Fatalf("first fee calculation failed: %v", err)
	}
	second, err := module.Handler.CalculateFeeHandler(ctx, "idem-fee-1", httptransport.CalculateFeeRequest{
		SubmissionID: "sub-fee-1",
		UserID:       "user-fee-1",
		CampaignID:   "campaign-fee-1",
		GrossAmount:  10.0,
		FeeRate:      0.2,
	})
	if err != nil {
		t.Fatalf("second fee calculation failed: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed result on duplicate idempotency key")
	}
	if first.Data.CalculationID != second.Data.CalculationID {
		t.Fatalf("expected same calculation id, got %s and %s", first.Data.CalculationID, second.Data.CalculationID)
	}
}

func TestPlatformFeeConsumeRewardPayoutEligibleProducesCanonicalOutboxEvent(t *testing.T) {
	module := platformfeeengine.NewInMemoryModule(nil)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := module.Handler.ConsumeRewardPayoutEligibleEventHandler(ctx, httptransport.RewardPayoutEligibleEventRequest{
		EventID:      "evt-reward-eligible-1",
		SubmissionID: "sub-fee-2",
		UserID:       "user-fee-2",
		CampaignID:   "campaign-fee-2",
		GrossAmount:  25.0,
		EligibleAt:   now,
	})
	if err != nil {
		t.Fatalf("consume reward.payout_eligible failed: %v", err)
	}

	outbox, err := module.Store.ListPendingOutbox(ctx, 50)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	if len(outbox) == 0 {
		t.Fatalf("expected at least one outbox event")
	}

	found := false
	for _, msg := range outbox {
		if msg.EventType != "fee.calculated" {
			continue
		}
		found = true
		var envelope map[string]any
		if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
			t.Fatalf("decode outbox envelope: %v", err)
		}

		if sourceService, _ := envelope["source_service"].(string); sourceService != "platform-fee-engine" {
			t.Fatalf("unexpected source_service: %s", sourceService)
		}
		if traceID, _ := envelope["trace_id"].(string); strings.TrimSpace(traceID) == "" {
			t.Fatalf("fee.calculated missing trace_id")
		}
		if partitionPath, _ := envelope["partition_key_path"].(string); partitionPath != "submission_id" {
			t.Fatalf("unexpected partition_key_path: %s", partitionPath)
		}

		partitionKey, _ := envelope["partition_key"].(string)
		data, _ := envelope["data"].(map[string]any)
		dataSubmissionID, _ := data["submission_id"].(string)
		if dataSubmissionID != partitionKey {
			t.Fatalf("partition mismatch data.submission_id=%s partition_key=%s", dataSubmissionID, partitionKey)
		}
	}
	if !found {
		t.Fatalf("expected fee.calculated event in outbox")
	}
}

func TestPlatformFeeCanDisableFeeCalculatedEventEmission(t *testing.T) {
	store := memory.NewStore()
	module := platformfeeengine.NewModule(platformfeeengine.Dependencies{
		Repository:                        store,
		Idempotency:                       store,
		EventDedup:                        store,
		Outbox:                            store,
		Clock:                             store,
		IDGenerator:                       store,
		IdempotencyTTL:                    7 * 24 * time.Hour,
		EventDedupTTL:                     7 * 24 * time.Hour,
		DefaultFeeRate:                    0.15,
		DisableFeeCalculatedEventEmission: true,
	})
	module.Store = store

	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := module.Handler.ConsumeRewardPayoutEligibleEventHandler(ctx, httptransport.RewardPayoutEligibleEventRequest{
		EventID:      "evt-reward-eligible-disabled",
		SubmissionID: "sub-fee-disabled",
		UserID:       "user-fee-disabled",
		CampaignID:   "campaign-fee-disabled",
		GrossAmount:  25.0,
		EligibleAt:   now,
	}); err != nil {
		t.Fatalf("consume reward.payout_eligible failed: %v", err)
	}

	outbox, err := module.Store.ListPendingOutbox(ctx, 50)
	if err != nil {
		t.Fatalf("list outbox failed: %v", err)
	}
	if len(outbox) != 0 {
		t.Fatalf("expected no outbox events when fee-calculated emission disabled, got %d", len(outbox))
	}
}
