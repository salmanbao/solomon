package commands

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/campaign-service/application"
	"solomon/contexts/campaign-editorial/campaign-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/campaign-service/domain/errors"
	"solomon/contexts/campaign-editorial/campaign-service/ports"
)

type IncreaseBudgetCommand struct {
	CampaignID string
	ActorID    string
	Amount     float64
	Reason     string
}

type IncreaseBudgetUseCase struct {
	Campaigns ports.CampaignRepository
	History   ports.HistoryRepository
	Outbox    ports.OutboxWriter
	Clock     ports.Clock
	IDGen     ports.IDGenerator
	Logger    *slog.Logger
}

func (uc IncreaseBudgetUseCase) Execute(ctx context.Context, cmd IncreaseBudgetCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	if cmd.Amount <= 0 {
		return domainerrors.ErrInvalidBudgetIncrease
	}
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(cmd.CampaignID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" || campaign.BrandID != strings.TrimSpace(cmd.ActorID) {
		return domainerrors.ErrInvalidCampaignInput
	}
	if campaign.Status != entities.CampaignStatusPaused {
		return domainerrors.ErrInvalidStateTransition
	}

	campaign.BudgetTotal += cmd.Amount
	campaign.BudgetRemaining += cmd.Amount
	campaign.UpdatedAt = uc.Clock.Now().UTC()
	if err := uc.Campaigns.UpdateCampaign(ctx, campaign); err != nil {
		return err
	}
	logID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	if err := uc.History.AppendBudget(ctx, entities.BudgetLog{
		LogID:       logID,
		CampaignID:  campaign.CampaignID,
		AmountDelta: cmd.Amount,
		Reason:      strings.TrimSpace(cmd.Reason),
		CreatedAt:   campaign.UpdatedAt,
	}); err != nil {
		return err
	}
	if uc.Outbox != nil {
		eventID, err := uc.IDGen.NewID(ctx)
		if err != nil {
			return err
		}
		envelope, err := newCampaignEnvelope(
			eventID,
			"campaign.budget_updated",
			campaign.CampaignID,
			campaign.UpdatedAt,
			map[string]any{
				"campaign_id":      campaign.CampaignID,
				"budget_total":     campaign.BudgetTotal,
				"budget_spent":     campaign.BudgetSpent,
				"budget_reserved":  campaign.BudgetReserved,
				"budget_remaining": campaign.BudgetRemaining,
			},
		)
		if err != nil {
			return err
		}
		if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
			return err
		}
	}

	logger.Info("campaign budget increased",
		"event", "campaign_budget_increased",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
		"amount", cmd.Amount,
		"new_budget_total", campaign.BudgetTotal,
	)
	return nil
}
