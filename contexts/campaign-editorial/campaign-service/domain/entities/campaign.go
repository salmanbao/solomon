package entities

import (
	"strings"
	"time"
)

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
)

type Campaign struct {
	CampaignID       string
	BrandID          string
	Title            string
	Description      string
	Instructions     string
	Niche            string
	AllowedPlatforms []string
	RequiredHashtags []string
	BudgetTotal      float64
	BudgetSpent      float64
	BudgetRemaining  float64
	RatePer1KViews   float64
	Status           CampaignStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
	LaunchedAt       *time.Time
	CompletedAt      *time.Time
}

func (c Campaign) CanEdit() bool {
	return c.Status == CampaignStatusDraft || c.Status == CampaignStatusPaused
}

func (c Campaign) ValidateBasics() bool {
	title := strings.TrimSpace(c.Title)
	description := strings.TrimSpace(c.Description)
	instructions := strings.TrimSpace(c.Instructions)
	niche := strings.TrimSpace(c.Niche)
	return title != "" &&
		len(title) >= 3 &&
		description != "" &&
		instructions != "" &&
		niche != "" &&
		c.BudgetTotal >= 10.0 &&
		c.RatePer1KViews >= 0.1 &&
		c.RatePer1KViews <= 5.0
}
