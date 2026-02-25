package entities

import (
	"strings"
	"time"
)

type CampaignStatus string
type CampaignType string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"

	CampaignTypeUGCCreation     CampaignType = "ugc_creation"
	CampaignTypeUGCDistribution CampaignType = "ugc_distribution"
	CampaignTypeHybrid          CampaignType = "hybrid"
)

type Campaign struct {
	CampaignID              string
	BrandID                 string
	Title                   string
	Description             string
	Instructions            string
	Niche                   string
	AllowedPlatforms        []string
	RequiredHashtags        []string
	RequiredTags            []string
	OptionalHashtags        []string
	UsageGuidelines         string
	DosAndDonts             string
	CampaignType            CampaignType
	DeadlineAt              *time.Time
	TargetSubmissions       *int
	BannerImageURL          string
	ExternalURL             string
	BudgetTotal             float64
	BudgetSpent             float64
	BudgetReserved          float64
	BudgetRemaining         float64
	RatePer1KViews          float64
	SubmissionCount         int
	ApprovedSubmissionCount int
	TotalViews              int64
	Status                  CampaignStatus
	CreatedAt               time.Time
	UpdatedAt               time.Time
	LaunchedAt              *time.Time
	CompletedAt             *time.Time
}

func (c Campaign) CanEdit() bool {
	return c.Status == CampaignStatusDraft || c.Status == CampaignStatusPaused
}

func (c Campaign) ValidateBasics(now time.Time) bool {
	title := strings.TrimSpace(c.Title)
	description := strings.TrimSpace(c.Description)
	instructions := strings.TrimSpace(c.Instructions)
	niche := strings.TrimSpace(c.Niche)
	if c.CampaignType == "" {
		c.CampaignType = CampaignTypeUGCCreation
	}

	return title != "" &&
		len(title) >= 3 &&
		len(title) <= 100 &&
		description != "" &&
		len(description) >= 10 &&
		len(description) <= 2000 &&
		instructions != "" &&
		len(instructions) >= 10 &&
		len(instructions) <= 5000 &&
		niche != "" &&
		IsSupportedNiche(niche) &&
		c.BudgetTotal >= 10.0 &&
		c.BudgetTotal <= 1_000_000.0 &&
		c.RatePer1KViews >= 0.1 &&
		c.RatePer1KViews <= 5.0 &&
		len(c.AllowedPlatforms) > 0 &&
		AllSupportedPlatforms(c.AllowedPlatforms) &&
		len(c.RequiredHashtags) <= 10 &&
		len(c.RequiredTags) <= 5 &&
		len(c.OptionalHashtags) <= 10 &&
		IsSupportedCampaignType(c.CampaignType) &&
		DeadlineAtLeastSevenDays(c.DeadlineAt, now)
}

func BudgetAutoPauseThreshold(ratePer1KViews float64) float64 {
	return ratePer1KViews * 0.1
}

func IsSupportedCampaignType(value CampaignType) bool {
	switch value {
	case CampaignTypeUGCCreation, CampaignTypeUGCDistribution, CampaignTypeHybrid:
		return true
	default:
		return false
	}
}

func IsSupportedNiche(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "gaming", "beauty", "fitness", "tech", "comedy":
		return true
	default:
		return false
	}
}

func AllSupportedPlatforms(platforms []string) bool {
	if len(platforms) == 0 {
		return false
	}
	for _, item := range platforms {
		if !IsSupportedPlatform(item) {
			return false
		}
	}
	return true
}

func IsSupportedPlatform(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "tiktok", "instagram", "youtube", "twitter", "facebook", "snapchat", "linkedin":
		return true
	default:
		return false
	}
}

func DeadlineAtLeastSevenDays(deadline *time.Time, now time.Time) bool {
	if deadline == nil {
		return true
	}
	return deadline.UTC().After(now.UTC().Add(7 * 24 * time.Hour))
}
