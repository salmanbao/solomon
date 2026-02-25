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

type ChangeStatusAction string

const (
	StatusActionLaunch   ChangeStatusAction = "launch"
	StatusActionPause    ChangeStatusAction = "pause"
	StatusActionResume   ChangeStatusAction = "resume"
	StatusActionComplete ChangeStatusAction = "complete"
)

type ChangeStatusCommand struct {
	CampaignID string
	ActorID    string
	Action     ChangeStatusAction
	Reason     string
}

type ChangeStatusUseCase struct {
	Campaigns ports.CampaignRepository
	Media     ports.MediaRepository
	History   ports.HistoryRepository
	Outbox    ports.OutboxWriter
	Clock     ports.Clock
	IDGen     ports.IDGenerator
	Logger    *slog.Logger
}

func (uc ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) error {
	logger := application.ResolveLogger(uc.Logger)
	campaign, err := uc.Campaigns.GetCampaign(ctx, strings.TrimSpace(cmd.CampaignID))
	if err != nil {
		return err
	}
	if strings.TrimSpace(cmd.ActorID) == "" || campaign.BrandID != strings.TrimSpace(cmd.ActorID) {
		return domainerrors.ErrInvalidCampaignInput
	}

	now := uc.Clock.Now().UTC()
	from := campaign.Status
	to := from
	switch cmd.Action {
	case StatusActionLaunch:
		if campaign.Status != entities.CampaignStatusDraft {
			return domainerrors.ErrInvalidStateTransition
		}
		if !campaign.ValidateBasics(now) {
			return domainerrors.ErrInvalidCampaignInput
		}
		if !entities.DeadlineAtLeastSevenDays(campaign.DeadlineAt, now) {
			return domainerrors.ErrDeadlineTooSoon
		}
		if uc.Media == nil {
			return domainerrors.ErrMissingReadyMedia
		}
		mediaItems, err := uc.Media.ListMediaByCampaign(ctx, campaign.CampaignID)
		if err != nil {
			return err
		}
		ready := 0
		for _, item := range mediaItems {
			if item.Status == entities.MediaStatusReady {
				ready++
			}
		}
		if ready == 0 {
			return domainerrors.ErrMissingReadyMedia
		}
		to = entities.CampaignStatusActive
		campaign.LaunchedAt = &now
	case StatusActionPause:
		if campaign.Status == entities.CampaignStatusPaused {
			return nil
		}
		if campaign.Status != entities.CampaignStatusActive {
			return domainerrors.ErrInvalidStateTransition
		}
		to = entities.CampaignStatusPaused
	case StatusActionResume:
		if campaign.Status == entities.CampaignStatusActive {
			return nil
		}
		if campaign.Status != entities.CampaignStatusPaused {
			return domainerrors.ErrInvalidStateTransition
		}
		if campaign.BudgetRemaining <= entities.BudgetAutoPauseThreshold(campaign.RatePer1KViews) {
			return domainerrors.ErrInvalidStateTransition
		}
		to = entities.CampaignStatusActive
	case StatusActionComplete:
		if campaign.Status != entities.CampaignStatusActive && campaign.Status != entities.CampaignStatusPaused {
			return domainerrors.ErrInvalidStateTransition
		}
		to = entities.CampaignStatusCompleted
		campaign.CompletedAt = &now
	default:
		return domainerrors.ErrInvalidStateTransition
	}

	campaign.Status = to
	campaign.UpdatedAt = now
	if err := uc.Campaigns.UpdateCampaign(ctx, campaign); err != nil {
		return err
	}
	historyID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return err
	}
	if err := uc.History.AppendState(ctx, entities.StateHistory{
		HistoryID:    historyID,
		CampaignID:   campaign.CampaignID,
		FromState:    from,
		ToState:      to,
		ChangedBy:    strings.TrimSpace(cmd.ActorID),
		ChangeReason: strings.TrimSpace(cmd.Reason),
		CreatedAt:    now,
	}); err != nil {
		return err
	}
	if uc.Outbox != nil {
		eventType := ""
		switch to {
		case entities.CampaignStatusActive:
			if from == entities.CampaignStatusDraft {
				eventType = "campaign.launched"
			}
			if from == entities.CampaignStatusPaused {
				eventType = "campaign.resumed"
			}
		case entities.CampaignStatusPaused:
			eventType = "campaign.paused"
		case entities.CampaignStatusCompleted:
			eventType = "campaign.completed"
		}
		if strings.TrimSpace(eventType) != "" {
			eventID, err := uc.IDGen.NewID(ctx)
			if err != nil {
				return err
			}
			envelope, err := newCampaignEnvelope(
				eventID,
				eventType,
				campaign.CampaignID,
				now,
				map[string]any{
					"campaign_id": campaign.CampaignID,
					"brand_id":    campaign.BrandID,
					"from_status": string(from),
					"to_status":   string(to),
					"reason":      strings.TrimSpace(cmd.Reason),
				},
			)
			if err != nil {
				return err
			}
			if err := uc.Outbox.AppendOutbox(ctx, envelope); err != nil {
				return err
			}
		}
	}

	logger.Info("campaign state changed",
		"event", "campaign_state_changed",
		"module", "campaign-editorial/campaign-service",
		"layer", "application",
		"campaign_id", campaign.CampaignID,
		"from_status", string(from),
		"to_status", string(to),
	)
	return nil
}
