package ports

import (
	"context"
	"time"
)

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

type SubscriptionPlan struct {
	PlanID        string
	PlanKey       string
	PlanName      string
	PriceCents    int64
	Currency      string
	Interval      string
	IntervalCount int
	Features      []string
	IsActive      bool
	TrialEnabled  bool
	TrialDays     int
	ProductID     string
	DeprecatedAt  *time.Time
	CreatedAt     time.Time
}

type Subscription struct {
	SubscriptionID       string
	UserID               string
	PlanID               string
	PlanName             string
	Status               string
	TrialStart           *time.Time
	TrialEnd             *time.Time
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	NextBillingDate      *time.Time
	AmountCents          int64
	Currency             string
	BillingAnchorDay     int
	CancelAtPeriodEnd    bool
	CanceledAt           *time.Time
	AccessEndsAt         *time.Time
	CancellationFeedback string
	UpdatedAt            time.Time
	CreatedAt            time.Time
}

type CreateSubscriptionInput struct {
	PlanID string
	Trial  bool
}

type PlanChangeResult struct {
	SubscriptionID       string
	OldPlanID            string
	OldPlanName          string
	NewPlanID            string
	NewPlanName          string
	ProrationAmountCents int64
	ProrationDescription string
	NextBillingDate      *time.Time
	ChangedAt            time.Time
}

type CancelSubscriptionResult struct {
	SubscriptionID       string
	Status               string
	CancelAtPeriodEnd    bool
	AccessEndsAt         *time.Time
	CancellationFeedback string
	CanceledAt           *time.Time
}

type Repository interface {
	CreateSubscription(ctx context.Context, userID string, input CreateSubscriptionInput, now time.Time) (Subscription, error)
	ChangePlan(ctx context.Context, userID string, subscriptionID string, newPlanID string, now time.Time) (PlanChangeResult, error)
	CancelSubscription(ctx context.Context, userID string, subscriptionID string, cancelAtPeriodEnd bool, feedback string, now time.Time) (CancelSubscriptionResult, error)
}
