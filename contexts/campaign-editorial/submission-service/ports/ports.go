package ports

import (
	"context"
	"time"

	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
)

type SubmissionFilter struct {
	CreatorID  string
	CampaignID string
	Status     entities.SubmissionStatus
}

type Repository interface {
	CreateSubmission(ctx context.Context, submission entities.Submission) error
	UpdateSubmission(ctx context.Context, submission entities.Submission) error
	GetSubmission(ctx context.Context, submissionID string) (entities.Submission, error)
	ListSubmissions(ctx context.Context, filter SubmissionFilter) ([]entities.Submission, error)
	AddReport(ctx context.Context, report entities.SubmissionReport) error
	AddFlag(ctx context.Context, flag entities.SubmissionFlag) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}
