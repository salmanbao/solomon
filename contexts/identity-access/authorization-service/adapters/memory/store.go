package memory

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"solomon/contexts/identity-access/authorization-service/domain/entities"
	domainerrors "solomon/contexts/identity-access/authorization-service/domain/errors"
	"solomon/contexts/identity-access/authorization-service/ports"

	"github.com/google/uuid"
)

// Store is an in-memory adapter implementing repository/cache/idempotency ports.
// It is intended for tests and local development wiring.
type Store struct {
	mu sync.RWMutex

	roles       map[string]entities.Role
	assignments map[string]entities.RoleAssignment
	delegations map[string]entities.Delegation

	idempotency map[string]ports.IdempotencyRecord
	cache       map[string]cacheEntry
	outbox      map[string]outboxRow
	dedup       map[string]dedupEntry
}

type cacheEntry struct {
	Permissions []string
	ExpiresAt   time.Time
}

type outboxRow struct {
	ports.OutboxMessage
	PublishedAt *time.Time
}

type dedupEntry struct {
	PayloadHash string
	ExpiresAt   time.Time
}

// NewStore builds a deterministic in-memory adapter seeded with baseline roles.
func NewStore() *Store {
	roles := map[string]entities.Role{
		"guest": {
			RoleID:      "guest",
			RoleName:    "guest",
			Permissions: []string{"campaign.view", "profile.view"},
		},
		"influencer": {
			RoleID:      "influencer",
			RoleName:    "influencer",
			Permissions: []string{"campaign.view", "content.view", "distribution.post"},
		},
		"editor": {
			RoleID:      "editor",
			RoleName:    "editor",
			Permissions: []string{"campaign.view", "submission.create", "submission.edit"},
		},
		"brand": {
			RoleID:      "brand",
			RoleName:    "brand",
			Permissions: []string{"campaign.create", "campaign.edit", "submission.approve"},
		},
		"admin": {
			RoleID:      "admin",
			RoleName:    "admin",
			Permissions: []string{"user.grant_role", "user.revoke_role", "policy.manage"},
		},
		"super_admin": {
			RoleID:      "super_admin",
			RoleName:    "super_admin",
			Permissions: []string{"user.grant_role", "user.revoke_role", "policy.manage", "role.delegate"},
		},
	}
	return &Store{
		roles:       roles,
		assignments: make(map[string]entities.RoleAssignment),
		delegations: make(map[string]entities.Delegation),
		idempotency: make(map[string]ports.IdempotencyRecord),
		cache:       make(map[string]cacheEntry),
		outbox:      make(map[string]outboxRow),
		dedup:       make(map[string]dedupEntry),
	}
}

// ListEffectivePermissions resolves active assignment and delegation permissions.
func (s *Store) ListEffectivePermissions(_ context.Context, userID string, now time.Time) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	permissions := make(map[string]struct{})
	for _, assignment := range s.assignments {
		if assignment.UserID != userID || !assignment.IsActive {
			continue
		}
		if assignment.ExpiresAt != nil && !assignment.ExpiresAt.After(now) {
			continue
		}
		role, ok := s.roles[assignment.RoleID]
		if !ok {
			continue
		}
		for _, permission := range role.Permissions {
			permissions[permission] = struct{}{}
		}
	}
	for _, delegation := range s.delegations {
		if delegation.ToAdminID != userID || !delegation.IsActive || !delegation.ExpiresAt.After(now) {
			continue
		}
		role, ok := s.roles[delegation.RoleID]
		if !ok {
			continue
		}
		for _, permission := range role.Permissions {
			permissions[permission] = struct{}{}
		}
	}

	items := make([]string, 0, len(permissions))
	for permission := range permissions {
		items = append(items, permission)
	}
	sort.Strings(items)
	return items, nil
}

// ListUserRoles returns role assignments filtered by user identity.
func (s *Store) ListUserRoles(_ context.Context, userID string, now time.Time) ([]entities.RoleAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]entities.RoleAssignment, 0)
	for _, assignment := range s.assignments {
		if assignment.UserID != userID {
			continue
		}
		if assignment.IsActive && assignment.ExpiresAt != nil && !assignment.ExpiresAt.After(now) {
			continue
		}
		items = append(items, assignment)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].AssignedAt.After(items[j].AssignedAt)
	})
	return items, nil
}

func (s *Store) GrantRole(_ context.Context, input ports.GrantRoleInput) (ports.RoleMutationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	role, ok := s.roles[input.RoleID]
	if !ok {
		return ports.RoleMutationResult{}, domainerrors.ErrRoleNotFound
	}
	for _, assignment := range s.assignments {
		if assignment.UserID == input.UserID && assignment.RoleID == input.RoleID && assignment.IsActive {
			if assignment.ExpiresAt == nil || assignment.ExpiresAt.After(input.AssignedAt) {
				return ports.RoleMutationResult{}, domainerrors.ErrRoleAlreadyAssigned
			}
		}
	}

	assignment := entities.RoleAssignment{
		AssignmentID: input.AssignmentID,
		UserID:       input.UserID,
		RoleID:       input.RoleID,
		RoleName:     role.RoleName,
		AssignedBy:   input.AdminID,
		Reason:       input.Reason,
		AssignedAt:   input.AssignedAt.UTC(),
		ExpiresAt:    input.ExpiresAt,
		IsActive:     true,
	}
	s.assignments[assignment.AssignmentID] = assignment

	payload, err := json.Marshal(map[string]string{
		"user_id":     input.UserID,
		"role_id":     input.RoleID,
		"action_type": "role_granted",
	})
	if err != nil {
		return ports.RoleMutationResult{}, err
	}
	if err := s.appendOutbox(input.OutboxID, "authz.policy_changed", payload, input.AssignedAt.UTC()); err != nil {
		return ports.RoleMutationResult{}, err
	}
	return ports.RoleMutationResult{
		Assignment: assignment,
		AuditLogID: input.AuditLogID,
	}, nil
}

func (s *Store) RevokeRole(_ context.Context, input ports.RevokeRoleInput) (ports.RoleMutationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target entities.RoleAssignment
	found := false
	for id, assignment := range s.assignments {
		if assignment.UserID == input.UserID && assignment.RoleID == input.RoleID && assignment.IsActive {
			target = assignment
			target.IsActive = false
			revokedAt := input.RevokedAt.UTC()
			target.RevokedAt = &revokedAt
			s.assignments[id] = target
			found = true
			break
		}
	}
	if !found {
		return ports.RoleMutationResult{}, domainerrors.ErrRoleNotAssigned
	}

	payload, err := json.Marshal(map[string]string{
		"user_id":     input.UserID,
		"role_id":     input.RoleID,
		"action_type": "role_revoked",
	})
	if err != nil {
		return ports.RoleMutationResult{}, err
	}
	if err := s.appendOutbox(input.OutboxID, "authz.policy_changed", payload, input.RevokedAt.UTC()); err != nil {
		return ports.RoleMutationResult{}, err
	}
	return ports.RoleMutationResult{
		Assignment: target,
		AuditLogID: input.AuditLogID,
	}, nil
}

func (s *Store) CreateDelegation(_ context.Context, input ports.DelegationInput) (ports.DelegationMutationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.FromAdminID == input.ToAdminID || !input.ExpiresAt.After(input.DelegatedAt) {
		return ports.DelegationMutationResult{}, domainerrors.ErrInvalidDelegation
	}
	if _, ok := s.roles[input.RoleID]; !ok {
		return ports.DelegationMutationResult{}, domainerrors.ErrRoleNotFound
	}

	delegation := entities.Delegation{
		DelegationID: input.DelegationID,
		FromAdminID:  input.FromAdminID,
		ToAdminID:    input.ToAdminID,
		RoleID:       input.RoleID,
		Reason:       input.Reason,
		DelegatedAt:  input.DelegatedAt.UTC(),
		ExpiresAt:    input.ExpiresAt.UTC(),
		IsActive:     true,
	}
	s.delegations[delegation.DelegationID] = delegation

	payload, err := json.Marshal(map[string]string{
		"user_id":     input.ToAdminID,
		"role_id":     input.RoleID,
		"action_type": "delegated",
	})
	if err != nil {
		return ports.DelegationMutationResult{}, err
	}
	if err := s.appendOutbox(input.OutboxID, "authz.policy_changed", payload, input.DelegatedAt.UTC()); err != nil {
		return ports.DelegationMutationResult{}, err
	}
	return ports.DelegationMutationResult{
		Delegation: delegation,
		AuditLogID: input.AuditLogID,
	}, nil
}

func (s *Store) GetRecord(_ context.Context, key string, now time.Time) (ports.IdempotencyRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.idempotency[key]
	if !ok {
		return ports.IdempotencyRecord{}, false, nil
	}
	if !record.ExpiresAt.After(now) {
		delete(s.idempotency, key)
		return ports.IdempotencyRecord{}, false, nil
	}
	return record, true, nil
}

func (s *Store) PutRecord(_ context.Context, record ports.IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.idempotency[record.Key]
	if exists && existing.RequestHash != record.RequestHash {
		return domainerrors.ErrIdempotencyConflict
	}
	s.idempotency[record.Key] = record
	return nil
}

func (s *Store) Get(_ context.Context, userID string, now time.Time) ([]string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.cache[userID]
	if !ok {
		return nil, false, nil
	}
	if !entry.ExpiresAt.After(now) {
		delete(s.cache, userID)
		return nil, false, nil
	}
	return append([]string(nil), entry.Permissions...), true, nil
}

func (s *Store) Set(_ context.Context, userID string, permissions []string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[userID] = cacheEntry{
		Permissions: append([]string(nil), permissions...),
		ExpiresAt:   expiresAt.UTC(),
	}
	return nil
}

func (s *Store) Invalidate(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.cache, userID)
	return nil
}

func (s *Store) ListPendingOutbox(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	rows := make([]ports.OutboxMessage, 0, len(s.outbox))
	for _, row := range s.outbox {
		if row.PublishedAt == nil {
			rows = append(rows, row.OutboxMessage)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (s *Store) MarkOutboxPublished(_ context.Context, outboxID string, publishedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	row, ok := s.outbox[outboxID]
	if !ok {
		return errors.New("outbox record not found")
	}
	value := publishedAt.UTC()
	row.PublishedAt = &value
	s.outbox[outboxID] = row
	return nil
}

func (s *Store) ReserveEvent(_ context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.dedup[eventID]
	if !ok {
		s.dedup[eventID] = dedupEntry{
			PayloadHash: payloadHash,
			ExpiresAt:   expiresAt.UTC(),
		}
		return false, nil
	}
	if existing.PayloadHash != payloadHash {
		return false, domainerrors.ErrIdempotencyConflict
	}
	return true, nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) NewID(_ context.Context) (string, error) {
	return uuid.NewString(), nil
}

func (s *Store) appendOutbox(outboxID string, eventType string, payload []byte, createdAt time.Time) error {
	if _, exists := s.outbox[outboxID]; exists {
		return domainerrors.ErrIdempotencyConflict
	}
	s.outbox[outboxID] = outboxRow{
		OutboxMessage: ports.OutboxMessage{
			OutboxID:  outboxID,
			EventType: eventType,
			Payload:   append([]byte(nil), payload...),
			CreatedAt: createdAt,
		},
	}
	return nil
}
