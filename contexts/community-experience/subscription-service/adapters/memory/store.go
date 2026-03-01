package memory

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/community-experience/subscription-service/domain/errors"
	"solomon/contexts/community-experience/subscription-service/ports"
)

type Store struct {
	mu sync.RWMutex

	plansByID              map[string]ports.SubscriptionPlan
	subscriptionsByID      map[string]ports.Subscription
	trialHistoryByUserPlan map[string]bool

	// DBR:M60-Product-Service read-only projection (no cross-service writes).
	productIDsByPlanID map[string]string
	validProductIDs    map[string]struct{}

	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	plans := map[string]ports.SubscriptionPlan{
		"plan_pro_monthly": {
			PlanID:        "plan_pro_monthly",
			PlanKey:       "pro-monthly",
			PlanName:      "Pro Monthly",
			PriceCents:    4900,
			Currency:      "USD",
			Interval:      "monthly",
			IntervalCount: 1,
			Features: []string{
				"Unlimited campaigns",
				"Priority support",
				"Advanced analytics",
			},
			IsActive:     true,
			TrialEnabled: true,
			TrialDays:    14,
			ProductID:    "prod_002",
			CreatedAt:    now.Add(-60 * 24 * time.Hour),
		},
		"plan_pro_annual": {
			PlanID:        "plan_pro_annual",
			PlanKey:       "pro-annual",
			PlanName:      "Pro Annual",
			PriceCents:    49000,
			Currency:      "USD",
			Interval:      "annual",
			IntervalCount: 1,
			Features: []string{
				"Unlimited campaigns",
				"Priority support",
				"Advanced analytics",
			},
			IsActive:     true,
			TrialEnabled: false,
			TrialDays:    14,
			ProductID:    "prod_001",
			CreatedAt:    now.Add(-60 * 24 * time.Hour),
		},
		"plan_enterprise_monthly": {
			PlanID:        "plan_enterprise_monthly",
			PlanKey:       "enterprise-monthly",
			PlanName:      "Enterprise Monthly",
			PriceCents:    14900,
			Currency:      "USD",
			Interval:      "monthly",
			IntervalCount: 1,
			Features: []string{
				"Unlimited campaigns",
				"Dedicated account manager",
				"SLA support",
				"Advanced analytics",
			},
			IsActive:     true,
			TrialEnabled: true,
			TrialDays:    14,
			ProductID:    "prod_001",
			CreatedAt:    now.Add(-45 * 24 * time.Hour),
		},
	}

	store := &Store{
		plansByID:              make(map[string]ports.SubscriptionPlan, len(plans)),
		subscriptionsByID:      make(map[string]ports.Subscription),
		trialHistoryByUserPlan: make(map[string]bool),
		productIDsByPlanID: map[string]string{
			"plan_pro_monthly":        "prod_002",
			"plan_pro_annual":         "prod_001",
			"plan_enterprise_monthly": "prod_001",
		},
		validProductIDs: map[string]struct{}{
			"prod_001": {},
			"prod_002": {},
		},
		idempotency: make(map[string]ports.IdempotencyRecord),
		sequence:    1,
	}
	for planID, plan := range plans {
		store.plansByID[planID] = plan
	}
	return store
}

func (s *Store) CreateSubscription(
	ctx context.Context,
	userID string,
	input ports.CreateSubscriptionInput,
	now time.Time,
) (ports.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, ok := s.plansByID[strings.TrimSpace(input.PlanID)]
	if !ok {
		return ports.Subscription{}, domainerrors.ErrPlanNotFound
	}
	if !plan.IsActive {
		return ports.Subscription{}, domainerrors.ErrPlanInactive
	}
	if _, ok := s.validProductIDs[s.productIDsByPlanID[plan.PlanID]]; !ok {
		return ports.Subscription{}, domainerrors.ErrDependencyUnavailable
	}
	for _, item := range s.subscriptionsByID {
		if item.UserID == userID && item.PlanID == plan.PlanID &&
			(item.Status == "active" || item.Status == "trialing" || item.Status == "past_due") {
			return ports.Subscription{}, domainerrors.ErrConflict
		}
	}

	subID := "sub_" + s.nextID("m61")
	now = now.UTC()
	created := ports.Subscription{
		SubscriptionID:    subID,
		UserID:            userID,
		PlanID:            plan.PlanID,
		PlanName:          plan.PlanName,
		Status:            "active",
		AmountCents:       plan.PriceCents,
		Currency:          plan.Currency,
		BillingAnchorDay:  now.Day(),
		CancelAtPeriodEnd: false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	periodStart := now
	periodEnd := addBillingInterval(now, plan.Interval, plan.IntervalCount)

	if input.Trial && plan.TrialEnabled {
		key := trialKey(userID, plan.PlanID)
		if s.trialHistoryByUserPlan[key] {
			return ports.Subscription{}, domainerrors.ErrTrialAlreadyUsed
		}
		s.trialHistoryByUserPlan[key] = true
		trialStart := now
		trialEnd := now.AddDate(0, 0, clampInt(plan.TrialDays, 7, 30))
		nextBillingDate := trialEnd.Add(24 * time.Hour)
		created.Status = "trialing"
		created.TrialStart = &trialStart
		created.TrialEnd = &trialEnd
		created.NextBillingDate = &nextBillingDate
	} else {
		created.CurrentPeriodStart = &periodStart
		created.CurrentPeriodEnd = &periodEnd
		created.NextBillingDate = &periodEnd
	}

	s.subscriptionsByID[subID] = created
	return cloneSubscription(created), nil
}

func (s *Store) ChangePlan(
	ctx context.Context,
	userID string,
	subscriptionID string,
	newPlanID string,
	now time.Time,
) (ports.PlanChangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.subscriptionsByID[subscriptionID]
	if !ok {
		return ports.PlanChangeResult{}, domainerrors.ErrSubscriptionNotFound
	}
	if item.UserID != userID {
		return ports.PlanChangeResult{}, domainerrors.ErrForbidden
	}
	if item.Status != "active" && item.Status != "trialing" && item.Status != "past_due" {
		return ports.PlanChangeResult{}, domainerrors.ErrInvalidTransition
	}
	newPlan, ok := s.plansByID[strings.TrimSpace(newPlanID)]
	if !ok {
		return ports.PlanChangeResult{}, domainerrors.ErrPlanNotFound
	}
	if !newPlan.IsActive {
		return ports.PlanChangeResult{}, domainerrors.ErrPlanInactive
	}
	if item.PlanID == newPlan.PlanID {
		return ports.PlanChangeResult{}, domainerrors.ErrConflict
	}
	if _, ok := s.validProductIDs[s.productIDsByPlanID[newPlan.PlanID]]; !ok {
		return ports.PlanChangeResult{}, domainerrors.ErrDependencyUnavailable
	}

	oldPlan := s.plansByID[item.PlanID]
	now = now.UTC()
	proration := computeProration(item, oldPlan.PriceCents, newPlan.PriceCents, now)

	item.PlanID = newPlan.PlanID
	item.PlanName = newPlan.PlanName
	item.AmountCents = newPlan.PriceCents
	item.Currency = newPlan.Currency
	item.UpdatedAt = now
	s.subscriptionsByID[subscriptionID] = item

	result := ports.PlanChangeResult{
		SubscriptionID:       item.SubscriptionID,
		OldPlanID:            oldPlan.PlanID,
		OldPlanName:          oldPlan.PlanName,
		NewPlanID:            newPlan.PlanID,
		NewPlanName:          newPlan.PlanName,
		ProrationAmountCents: proration,
		ProrationDescription: prorationDescription(proration),
		NextBillingDate:      item.NextBillingDate,
		ChangedAt:            now,
	}
	return result, nil
}

func (s *Store) CancelSubscription(
	ctx context.Context,
	userID string,
	subscriptionID string,
	cancelAtPeriodEnd bool,
	feedback string,
	now time.Time,
) (ports.CancelSubscriptionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.subscriptionsByID[subscriptionID]
	if !ok {
		return ports.CancelSubscriptionResult{}, domainerrors.ErrSubscriptionNotFound
	}
	if item.UserID != userID {
		return ports.CancelSubscriptionResult{}, domainerrors.ErrForbidden
	}
	if item.Status != "active" && item.Status != "trialing" && item.Status != "past_due" {
		return ports.CancelSubscriptionResult{}, domainerrors.ErrInvalidTransition
	}

	now = now.UTC()
	item.Status = "canceled"
	item.CancelAtPeriodEnd = cancelAtPeriodEnd
	item.CancellationFeedback = strings.TrimSpace(feedback)
	item.UpdatedAt = now
	item.CanceledAt = &now

	if cancelAtPeriodEnd {
		if item.CurrentPeriodEnd != nil {
			end := item.CurrentPeriodEnd.UTC()
			item.AccessEndsAt = &end
		} else if item.TrialEnd != nil {
			end := item.TrialEnd.UTC()
			item.AccessEndsAt = &end
		} else if item.NextBillingDate != nil {
			end := item.NextBillingDate.UTC()
			item.AccessEndsAt = &end
		} else {
			end := now
			item.AccessEndsAt = &end
		}
	} else {
		end := now
		item.AccessEndsAt = &end
	}

	s.subscriptionsByID[subscriptionID] = item
	return ports.CancelSubscriptionResult{
		SubscriptionID:       item.SubscriptionID,
		Status:               item.Status,
		CancelAtPeriodEnd:    item.CancelAtPeriodEnd,
		AccessEndsAt:         item.AccessEndsAt,
		CancellationFeedback: item.CancellationFeedback,
		CanceledAt:           item.CanceledAt,
	}, nil
}

func (s *Store) Get(ctx context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.IsZero() && now.UTC().After(record.ExpiresAt.UTC()) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) Put(ctx context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.idempotency[record.Key]; ok {
		if existing.RequestHash != record.RequestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return nil
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) NewID(ctx context.Context) (string, error) {
	return s.nextID("m61"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func addBillingInterval(from time.Time, interval string, count int) time.Time {
	count = clampInt(count, 1, 12)
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "annual":
		return from.AddDate(count, 0, 0)
	default:
		return from.AddDate(0, count, 0)
	}
}

func computeProration(item ports.Subscription, oldPrice int64, newPrice int64, now time.Time) int64 {
	if item.CurrentPeriodStart == nil || item.CurrentPeriodEnd == nil {
		return 0
	}
	start := item.CurrentPeriodStart.UTC()
	end := item.CurrentPeriodEnd.UTC()
	if !end.After(start) {
		return 0
	}
	if !end.After(now) {
		return 0
	}

	daysInCycle := math.Max(1, end.Sub(start).Hours()/24)
	daysRemaining := math.Max(0, end.Sub(now).Hours()/24)
	dailyOld := float64(oldPrice) / daysInCycle
	dailyNew := float64(newPrice) / daysInCycle
	proration := int64(math.Round((dailyNew - dailyOld) * daysRemaining))
	if proration > -10 && proration < 10 {
		return 0
	}
	return proration
}

func prorationDescription(amount int64) string {
	switch {
	case amount > 0:
		return fmt.Sprintf("charged %d cents for remaining cycle", amount)
	case amount < 0:
		return fmt.Sprintf("credited %d cents for remaining cycle", -amount)
	default:
		return "no proration amount applied"
	}
}

func trialKey(userID string, planID string) string {
	return strings.TrimSpace(userID) + "|" + strings.TrimSpace(planID)
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func cloneSubscription(in ports.Subscription) ports.Subscription {
	out := in
	if in.TrialStart != nil {
		v := in.TrialStart.UTC()
		out.TrialStart = &v
	}
	if in.TrialEnd != nil {
		v := in.TrialEnd.UTC()
		out.TrialEnd = &v
	}
	if in.CurrentPeriodStart != nil {
		v := in.CurrentPeriodStart.UTC()
		out.CurrentPeriodStart = &v
	}
	if in.CurrentPeriodEnd != nil {
		v := in.CurrentPeriodEnd.UTC()
		out.CurrentPeriodEnd = &v
	}
	if in.NextBillingDate != nil {
		v := in.NextBillingDate.UTC()
		out.NextBillingDate = &v
	}
	if in.CanceledAt != nil {
		v := in.CanceledAt.UTC()
		out.CanceledAt = &v
	}
	if in.AccessEndsAt != nil {
		v := in.AccessEndsAt.UTC()
		out.AccessEndsAt = &v
	}
	return out
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
