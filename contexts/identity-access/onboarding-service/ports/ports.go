package ports

import (
	"context"
	"time"
)

const (
	RoleBrand      = "brand"
	RoleEditor     = "editor"
	RoleInfluencer = "influencer"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleBrand, RoleEditor, RoleInfluencer:
		return true
	default:
		return false
	}
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Payload     []byte
	ExpiresAt   time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	Put(ctx context.Context, record IdempotencyRecord) error
}

type UserRegisteredEvent struct {
	EventID    string
	UserID     string
	Role       string
	OccurredAt time.Time
}

type FlowStep struct {
	StepKey string
	Title   string
	Status  string
}

type FlowState struct {
	UserID         string
	Role           string
	FlowID         string
	VariantKey     string
	Status         string
	CompletedSteps int
	TotalSteps     int
	Steps          []FlowStep
}

type StepCompletion struct {
	StepKey        string
	Status         string
	CompletedSteps int
	TotalSteps     int
}

type SkipResult struct {
	Status              string
	ReminderScheduledAt time.Time
}

type ResumeResult struct {
	Status   string
	NextStep string
}

type AdminFlow struct {
	FlowID     string
	Role       string
	IsActive   bool
	StepsCount int
}

type Repository interface {
	ConsumeUserRegisteredEvent(ctx context.Context, event UserRegisteredEvent, now time.Time) (FlowState, error)
	GetFlow(ctx context.Context, userID string) (FlowState, error)
	CompleteStep(ctx context.Context, userID string, stepKey string, metadata map[string]any, now time.Time) (StepCompletion, error)
	SkipFlow(ctx context.Context, userID string, reason string, now time.Time) (SkipResult, error)
	ResumeFlow(ctx context.Context, userID string, now time.Time) (ResumeResult, error)
	ListAdminFlows(ctx context.Context) ([]AdminFlow, error)
}
