package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/identity-access/onboarding-service/domain/errors"
	"solomon/contexts/identity-access/onboarding-service/ports"
)

type flowDefinition struct {
	FlowID string
	Role   string
	Steps  []ports.FlowStep
}

type progressRecord struct {
	UserID              string
	Role                string
	FlowID              string
	VariantKey          string
	Status              string
	StepStatusByStepKey map[string]string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ReminderScheduledAt *time.Time
}

type dedupRecord struct {
	RequestHash string
	ExpiresAt   time.Time
}

type Store struct {
	mu sync.RWMutex

	flowsByRole      map[string]flowDefinition
	progressByUserID map[string]progressRecord
	eventDedupByID   map[string]dedupRecord
	idempotency      map[string]ports.IdempotencyRecord
	sequence         uint64
}

func NewStore() *Store {
	return &Store{
		flowsByRole: map[string]flowDefinition{
			ports.RoleBrand: {
				FlowID: "flow_brand_default",
				Role:   ports.RoleBrand,
				Steps: []ports.FlowStep{
					{StepKey: "welcome", Title: "Welcome to ViralForge"},
					{StepKey: "connect_storefront", Title: "Connect your storefront"},
					{StepKey: "create_first_product", Title: "Create your first product"},
				},
			},
			ports.RoleEditor: {
				FlowID: "flow_editor_default",
				Role:   ports.RoleEditor,
				Steps: []ports.FlowStep{
					{StepKey: "welcome", Title: "Welcome to ViralForge"},
					{StepKey: "complete_profile", Title: "Complete your profile"},
					{StepKey: "submit_first_clip", Title: "Submit your first clip"},
				},
			},
			ports.RoleInfluencer: {
				FlowID: "flow_influencer_default",
				Role:   ports.RoleInfluencer,
				Steps: []ports.FlowStep{
					{StepKey: "welcome", Title: "Welcome to ViralForge"},
					{StepKey: "connect_social", Title: "Connect social account"},
					{StepKey: "join_first_campaign", Title: "Join your first campaign"},
				},
			},
		},
		progressByUserID: make(map[string]progressRecord),
		eventDedupByID:   make(map[string]dedupRecord),
		idempotency:      make(map[string]ports.IdempotencyRecord),
		sequence:         1,
	}
}

func (s *Store) ConsumeUserRegisteredEvent(ctx context.Context, event ports.UserRegisteredEvent, now time.Time) (ports.FlowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now = now.UTC()
	eventID := strings.TrimSpace(event.EventID)
	if eventID == "" || strings.TrimSpace(event.UserID) == "" {
		return ports.FlowState{}, domainerrors.ErrSchemaInvalid
	}
	role := strings.ToLower(strings.TrimSpace(event.Role))
	if !ports.IsValidRole(role) {
		return ports.FlowState{}, domainerrors.ErrUnknownRole
	}

	hash := hashEvent(event)
	if dedup, ok := s.eventDedupByID[eventID]; ok {
		if !dedup.ExpiresAt.IsZero() && now.After(dedup.ExpiresAt.UTC()) {
			delete(s.eventDedupByID, eventID)
		} else if dedup.RequestHash != hash {
			return ports.FlowState{}, domainerrors.ErrIdempotencyConflict
		} else {
			return s.getFlowLocked(event.UserID)
		}
	}

	flow, ok := s.flowsByRole[role]
	if !ok {
		return ports.FlowState{}, domainerrors.ErrDependencyUnavailable
	}

	if _, exists := s.progressByUserID[event.UserID]; !exists {
		stepStatus := make(map[string]string, len(flow.Steps))
		for _, step := range flow.Steps {
			stepStatus[step.StepKey] = "pending"
		}
		s.progressByUserID[event.UserID] = progressRecord{
			UserID:              event.UserID,
			Role:                role,
			FlowID:              flow.FlowID,
			VariantKey:          chooseVariant(event.UserID),
			Status:              "in_progress",
			StepStatusByStepKey: stepStatus,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
	}
	s.eventDedupByID[eventID] = dedupRecord{
		RequestHash: hash,
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
	}
	return s.getFlowLocked(event.UserID)
}

func (s *Store) GetFlow(ctx context.Context, userID string) (ports.FlowState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getFlowLocked(userID)
}

func (s *Store) CompleteStep(ctx context.Context, userID string, stepKey string, metadata map[string]any, now time.Time) (ports.StepCompletion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.progressByUserID[strings.TrimSpace(userID)]
	if !ok {
		return ports.StepCompletion{}, domainerrors.ErrProgressNotFound
	}
	flow, ok := s.flowsByRole[progress.Role]
	if !ok {
		return ports.StepCompletion{}, domainerrors.ErrFlowNotFound
	}
	stepKey = strings.TrimSpace(stepKey)
	if !hasStep(flow, stepKey) {
		return ports.StepCompletion{}, domainerrors.ErrStepNotFound
	}

	switch progress.Status {
	case "completed":
		return ports.StepCompletion{}, domainerrors.ErrFlowAlreadyCompleted
	case "skipped":
		return ports.StepCompletion{}, domainerrors.ErrConflict
	}

	if progress.StepStatusByStepKey[stepKey] == "completed" {
		return ports.StepCompletion{}, domainerrors.ErrStepAlreadyCompleted
	}
	progress.StepStatusByStepKey[stepKey] = "completed"
	progress.UpdatedAt = now.UTC()

	completed := countCompleted(progress.StepStatusByStepKey)
	total := len(flow.Steps)
	if completed >= total {
		progress.Status = "completed"
	} else {
		progress.Status = "in_progress"
	}
	s.progressByUserID[userID] = progress
	return ports.StepCompletion{
		StepKey:        stepKey,
		Status:         "completed",
		CompletedSteps: completed,
		TotalSteps:     total,
	}, nil
}

func (s *Store) SkipFlow(ctx context.Context, userID string, reason string, now time.Time) (ports.SkipResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.progressByUserID[strings.TrimSpace(userID)]
	if !ok {
		return ports.SkipResult{}, domainerrors.ErrProgressNotFound
	}
	if progress.Status == "completed" {
		return ports.SkipResult{}, domainerrors.ErrFlowAlreadyCompleted
	}
	if progress.Status == "skipped" {
		if progress.ReminderScheduledAt != nil {
			return ports.SkipResult{
				Status:              "skipped",
				ReminderScheduledAt: progress.ReminderScheduledAt.UTC(),
			}, nil
		}
		return ports.SkipResult{Status: "skipped", ReminderScheduledAt: now.UTC().Add(7 * 24 * time.Hour)}, nil
	}

	reminder := now.UTC().Add(7 * 24 * time.Hour)
	progress.Status = "skipped"
	progress.UpdatedAt = now.UTC()
	progress.ReminderScheduledAt = &reminder
	_ = reason
	s.progressByUserID[userID] = progress
	return ports.SkipResult{
		Status:              "skipped",
		ReminderScheduledAt: reminder,
	}, nil
}

func (s *Store) ResumeFlow(ctx context.Context, userID string, now time.Time) (ports.ResumeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	progress, ok := s.progressByUserID[strings.TrimSpace(userID)]
	if !ok {
		return ports.ResumeResult{}, domainerrors.ErrProgressNotFound
	}
	if progress.Status != "skipped" {
		return ports.ResumeResult{}, domainerrors.ErrResumeNotAllowed
	}

	nextStep := nextPendingStep(progress, s.flowsByRole[progress.Role])
	if nextStep == "" {
		return ports.ResumeResult{}, domainerrors.ErrResumeNotAllowed
	}

	progress.Status = "in_progress"
	progress.UpdatedAt = now.UTC()
	progress.ReminderScheduledAt = nil
	s.progressByUserID[userID] = progress
	return ports.ResumeResult{
		Status:   "in_progress",
		NextStep: nextStep,
	}, nil
}

func (s *Store) ListAdminFlows(ctx context.Context) ([]ports.AdminFlow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]ports.AdminFlow, 0, len(s.flowsByRole))
	for _, flow := range s.flowsByRole {
		items = append(items, ports.AdminFlow{
			FlowID:     flow.FlowID,
			Role:       flow.Role,
			IsActive:   true,
			StepsCount: len(flow.Steps),
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].Role < items[j].Role
	})
	return items, nil
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
	return s.nextID("m22"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func (s *Store) getFlowLocked(userID string) (ports.FlowState, error) {
	progress, ok := s.progressByUserID[strings.TrimSpace(userID)]
	if !ok {
		return ports.FlowState{}, domainerrors.ErrProgressNotFound
	}
	flow, ok := s.flowsByRole[progress.Role]
	if !ok {
		return ports.FlowState{}, domainerrors.ErrFlowNotFound
	}
	steps := make([]ports.FlowStep, 0, len(flow.Steps))
	for _, step := range flow.Steps {
		status := progress.StepStatusByStepKey[step.StepKey]
		if status == "" {
			status = "pending"
		}
		steps = append(steps, ports.FlowStep{
			StepKey: step.StepKey,
			Title:   step.Title,
			Status:  status,
		})
	}
	return ports.FlowState{
		UserID:         progress.UserID,
		Role:           progress.Role,
		FlowID:         progress.FlowID,
		VariantKey:     progress.VariantKey,
		Status:         progress.Status,
		CompletedSteps: countCompleted(progress.StepStatusByStepKey),
		TotalSteps:     len(flow.Steps),
		Steps:          steps,
	}, nil
}

func hasStep(flow flowDefinition, stepKey string) bool {
	for _, step := range flow.Steps {
		if step.StepKey == stepKey {
			return true
		}
	}
	return false
}

func countCompleted(stepStatus map[string]string) int {
	count := 0
	for _, status := range stepStatus {
		if status == "completed" {
			count++
		}
	}
	return count
}

func nextPendingStep(progress progressRecord, flow flowDefinition) string {
	for _, step := range flow.Steps {
		if progress.StepStatusByStepKey[step.StepKey] != "completed" {
			return step.StepKey
		}
	}
	return ""
}

func chooseVariant(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	if sum[0]%2 == 0 {
		return "A"
	}
	return "B"
}

func hashEvent(event ports.UserRegisteredEvent) string {
	raw, _ := json.Marshal(event)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
