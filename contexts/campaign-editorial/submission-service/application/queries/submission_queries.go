package queries

import (
	"context"
	"log/slog"
	"strings"

	application "solomon/contexts/campaign-editorial/submission-service/application"
	"solomon/contexts/campaign-editorial/submission-service/domain/entities"
	"solomon/contexts/campaign-editorial/submission-service/ports"
)

type ListSubmissionsQuery struct {
	CreatorID  string
	CampaignID string
	Status     string
}

type QueryUseCase struct {
	Repository ports.Repository
	Logger     *slog.Logger
}

func (uc QueryUseCase) GetSubmission(ctx context.Context, submissionID string) (entities.Submission, error) {
	return uc.Repository.GetSubmission(ctx, strings.TrimSpace(submissionID))
}

func (uc QueryUseCase) ListSubmissions(ctx context.Context, query ListSubmissionsQuery) ([]entities.Submission, error) {
	filter := ports.SubmissionFilter{
		CreatorID:  strings.TrimSpace(query.CreatorID),
		CampaignID: strings.TrimSpace(query.CampaignID),
	}
	if strings.TrimSpace(query.Status) != "" {
		filter.Status = entities.SubmissionStatus(strings.TrimSpace(query.Status))
	}
	items, err := uc.Repository.ListSubmissions(ctx, filter)
	if err != nil {
		return nil, err
	}
	return items, nil
}

type DashboardSummary struct {
	Total    int
	Pending  int
	Approved int
	Rejected int
	Flagged  int
}

func (uc QueryUseCase) CreatorDashboard(ctx context.Context, creatorID string) (DashboardSummary, error) {
	items, err := uc.Repository.ListSubmissions(ctx, ports.SubmissionFilter{
		CreatorID: strings.TrimSpace(creatorID),
	})
	if err != nil {
		return DashboardSummary{}, err
	}
	return summarize(items), nil
}

func (uc QueryUseCase) BrandDashboard(ctx context.Context, campaignID string) (DashboardSummary, error) {
	items, err := uc.Repository.ListSubmissions(ctx, ports.SubmissionFilter{
		CampaignID: strings.TrimSpace(campaignID),
	})
	if err != nil {
		return DashboardSummary{}, err
	}
	summary := summarize(items)
	application.ResolveLogger(uc.Logger).Debug("brand dashboard computed",
		"event", "submission_brand_dashboard_computed",
		"module", "campaign-editorial/submission-service",
		"layer", "application",
		"campaign_id", campaignID,
		"total", summary.Total,
	)
	return summary, nil
}

func summarize(items []entities.Submission) DashboardSummary {
	summary := DashboardSummary{Total: len(items)}
	for _, item := range items {
		switch item.Status {
		case entities.SubmissionStatusPending:
			summary.Pending++
		case entities.SubmissionStatusApproved:
			summary.Approved++
		case entities.SubmissionStatusRejected:
			summary.Rejected++
		case entities.SubmissionStatusFlagged:
			summary.Flagged++
		}
	}
	return summary
}
