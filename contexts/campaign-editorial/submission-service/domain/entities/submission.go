package entities

import (
	"strings"
	"time"
)

type SubmissionStatus string

const (
	SubmissionStatusPending   SubmissionStatus = "pending"
	SubmissionStatusApproved  SubmissionStatus = "approved"
	SubmissionStatusRejected  SubmissionStatus = "rejected"
	SubmissionStatusFlagged   SubmissionStatus = "flagged"
	SubmissionStatusCancelled SubmissionStatus = "cancelled"
)

type Submission struct {
	SubmissionID          string
	CampaignID            string
	CreatorID             string
	Platform              string
	PostURL               string
	Status                SubmissionStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ApprovedAt            *time.Time
	ApprovedByUserID      string
	ApprovalReason        string
	RejectedAt            *time.Time
	RejectionReason       string
	RejectionNotes        string
	ReportedCount         int
	VerificationWindowEnd *time.Time
}

func (s Submission) ValidateCreate() bool {
	return strings.TrimSpace(s.CampaignID) != "" &&
		strings.TrimSpace(s.CreatorID) != "" &&
		strings.TrimSpace(s.Platform) != "" &&
		strings.TrimSpace(s.PostURL) != ""
}

type SubmissionReport struct {
	ReportID     string
	SubmissionID string
	ReportedByID string
	Reason       string
	Description  string
	ReportedAt   time.Time
}

type SubmissionFlag struct {
	FlagID       string
	SubmissionID string
	FlagType     string
	Severity     string
	Details      string
	CreatedAt    time.Time
}
