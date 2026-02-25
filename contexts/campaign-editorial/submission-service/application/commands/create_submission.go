package commands

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	domainerrors "solomon/contexts/campaign-editorial/submission-service/domain/errors"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

type CreateSubmissionCommand struct {
	CreatorID  string
	CampaignID string
	Platform   string
	PostURL    string
}

type CreateSubmissionUseCase struct {
	Repository ports.Repository
	Clock      ports.Clock
	IDGen      ports.IDGenerator
	Logger     *slog.Logger
}

func (uc CreateSubmissionUseCase) Execute(ctx context.Context, cmd CreateSubmissionCommand) (entities.Submission, error) {
	logger := application.ResolveLogger(uc.Logger)
	submissionID, err := uc.IDGen.NewID(ctx)
	if err != nil {
		return entities.Submission{}, err
	}
	now := uc.Clock.Now().UTC()
	submission := entities.Submission{
		SubmissionID: submissionID,
		CampaignID:   strings.TrimSpace(cmd.CampaignID),
		CreatorID:    strings.TrimSpace(cmd.CreatorID),
		Platform:     strings.TrimSpace(cmd.Platform),
		PostURL:      strings.TrimSpace(cmd.PostURL),
		Status:       entities.SubmissionStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if !submission.ValidateCreate() {
		return entities.Submission{}, domainerrors.ErrInvalidSubmissionInput
	}
	if err := uc.Repository.CreateSubmission(ctx, submission); err != nil {
		return entities.Submission{}, err
	}
	logger.Info("submission created",
		"event", "submission_created",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"submission_id", submission.SubmissionID,
		"campaign_id", submission.CampaignID,
		"creator_id", submission.CreatorID,
	)
	return submission, nil
}
