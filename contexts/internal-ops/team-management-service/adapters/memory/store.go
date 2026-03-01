package memory

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	domainerrors "solomon/contexts/internal-ops/team-management-service/domain/errors"
	"solomon/contexts/internal-ops/team-management-service/ports"
)

var teamNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 -]{1,118}[A-Za-z0-9]$`)

type userProjection struct {
	Email      string
	MFAEnabled bool
}

type Store struct {
	mu sync.RWMutex

	teamsByID                     map[string]ports.Team
	membersByID                   map[string]ports.TeamMember
	memberIDByTeamUser            map[string]string
	invitesByToken                map[string]ports.TeamInvite
	inviteTokenByTeamEmailPending map[string]string
	auditLogsByTeamID             map[string][]ports.TeamAuditLog

	// DBR:M01-Authentication-Service owner_api read-only projection.
	usersByID     map[string]userProjection
	userIDByEmail map[string]string

	idempotency map[string]ports.IdempotencyRecord
	sequence    uint64
}

func NewStore() *Store {
	now := time.Now().UTC()
	users := map[string]userProjection{
		"user_owner_1":   {Email: "owner@example.com", MFAEnabled: true},
		"user_manager_1": {Email: "manager@example.com", MFAEnabled: true},
		"user_editor_1":  {Email: "editor@example.com", MFAEnabled: false},
		"user_viewer_1":  {Email: "viewer@example.com", MFAEnabled: false},
		"user_brand_1":   {Email: "brand@example.com", MFAEnabled: true},
	}

	store := &Store{
		teamsByID:                     make(map[string]ports.Team),
		membersByID:                   make(map[string]ports.TeamMember),
		memberIDByTeamUser:            make(map[string]string),
		invitesByToken:                make(map[string]ports.TeamInvite),
		inviteTokenByTeamEmailPending: make(map[string]string),
		auditLogsByTeamID:             make(map[string][]ports.TeamAuditLog),
		usersByID:                     make(map[string]userProjection, len(users)),
		userIDByEmail:                 make(map[string]string, len(users)),
		idempotency:                   make(map[string]ports.IdempotencyRecord),
		sequence:                      1,
	}

	for id, user := range users {
		store.usersByID[id] = user
		store.userIDByEmail[strings.ToLower(strings.TrimSpace(user.Email))] = id
	}

	teamID := "team_seed_1"
	owner := ports.Team{
		TeamID:       teamID,
		Name:         "Seed Team",
		OrgID:        "org_seed",
		StorefrontID: "store_seed",
		OwnerUserID:  "user_owner_1",
		Status:       "active",
		CreatedAt:    now.Add(-7 * 24 * time.Hour),
		UpdatedAt:    now.Add(-7 * 24 * time.Hour),
	}
	store.teamsByID[teamID] = owner
	memberOwner := ports.TeamMember{
		MemberID: "member_seed_owner",
		TeamID:   teamID,
		UserID:   "user_owner_1",
		Role:     ports.RoleOwner,
		Status:   "active",
		JoinedAt: now.Add(-7 * 24 * time.Hour),
	}
	memberManager := ports.TeamMember{
		MemberID: "member_seed_manager",
		TeamID:   teamID,
		UserID:   "user_manager_1",
		Role:     ports.RoleManager,
		Status:   "active",
		JoinedAt: now.Add(-6 * 24 * time.Hour),
	}
	store.membersByID[memberOwner.MemberID] = memberOwner
	store.membersByID[memberManager.MemberID] = memberManager
	store.memberIDByTeamUser[teamUserKey(teamID, memberOwner.UserID)] = memberOwner.MemberID
	store.memberIDByTeamUser[teamUserKey(teamID, memberManager.UserID)] = memberManager.MemberID
	store.addAuditLocked(teamID, "user_owner_1", "team.created", "team", teamID, now, map[string]string{
		"name": owner.Name,
	})

	return store
}

func (s *Store) CreateTeam(ctx context.Context, actorUserID string, input ports.CreateTeamInput, now time.Time) (ports.Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUserExistsLocked(actorUserID); err != nil {
		return ports.Team{}, err
	}
	if !isValidTeamName(input.Name) || strings.TrimSpace(input.OrgID) == "" {
		return ports.Team{}, domainerrors.ErrInvalidRequest
	}

	teamID := "team_" + s.nextID("m87")
	now = now.UTC()
	item := ports.Team{
		TeamID:       teamID,
		Name:         strings.TrimSpace(input.Name),
		OrgID:        strings.TrimSpace(input.OrgID),
		StorefrontID: strings.TrimSpace(input.StorefrontID),
		OwnerUserID:  actorUserID,
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.teamsByID[teamID] = item

	memberID := "member_" + s.nextID("m87")
	ownerMember := ports.TeamMember{
		MemberID: memberID,
		TeamID:   teamID,
		UserID:   actorUserID,
		Role:     ports.RoleOwner,
		Status:   "active",
		JoinedAt: now,
	}
	s.membersByID[memberID] = ownerMember
	s.memberIDByTeamUser[teamUserKey(teamID, actorUserID)] = memberID
	s.addAuditLocked(teamID, actorUserID, "team.created", "team", teamID, now, map[string]string{
		"name": item.Name,
	})

	return item, nil
}

func (s *Store) CreateInvite(
	ctx context.Context,
	actorUserID string,
	teamID string,
	email string,
	role string,
	now time.Time,
) (ports.TeamInvite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUserExistsLocked(actorUserID); err != nil {
		return ports.TeamInvite{}, err
	}
	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.TeamInvite{}, domainerrors.ErrTeamNotFound
	}
	if _, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID); err != nil {
		return ports.TeamInvite{}, err
	}
	if !ports.IsValidRole(role) || role == ports.RoleOwner {
		return ports.TeamInvite{}, domainerrors.ErrInvalidRequest
	}

	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return ports.TeamInvite{}, err
	}
	pendingKey := teamEmailKey(team.TeamID, normalizedEmail)
	if _, exists := s.inviteTokenByTeamEmailPending[pendingKey]; exists {
		return ports.TeamInvite{}, domainerrors.ErrConflict
	}
	if invitedUserID, exists := s.userIDByEmail[normalizedEmail]; exists {
		if _, memberExists := s.memberIDByTeamUser[teamUserKey(team.TeamID, invitedUserID)]; memberExists {
			return ports.TeamInvite{}, domainerrors.ErrConflict
		}
	}

	now = now.UTC()
	inviteID := "invite_" + s.nextID("m87")
	token := "token_" + s.nextID("m87")
	item := ports.TeamInvite{
		InviteID:  inviteID,
		TeamID:    team.TeamID,
		Email:     normalizedEmail,
		Role:      role,
		Token:     token,
		Status:    "pending",
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		CreatedBy: actorUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.invitesByToken[token] = item
	s.inviteTokenByTeamEmailPending[pendingKey] = token
	s.addAuditLocked(team.TeamID, actorUserID, "team.invite.sent", "invite", inviteID, now, map[string]string{
		"email": normalizedEmail,
		"role":  role,
	})
	return item, nil
}

func (s *Store) AcceptInvite(ctx context.Context, actorUserID string, token string, now time.Time) (ports.Membership, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUserExistsLocked(actorUserID); err != nil {
		return ports.Membership{}, err
	}
	invite, ok := s.invitesByToken[strings.TrimSpace(token)]
	if !ok {
		return ports.Membership{}, domainerrors.ErrInviteNotFound
	}
	if invite.Status != "pending" {
		return ports.Membership{}, domainerrors.ErrInvalidRequest
	}

	now = now.UTC()
	if now.After(invite.ExpiresAt.UTC()) {
		invite.Status = "expired"
		invite.UpdatedAt = now
		s.invitesByToken[invite.Token] = invite
		delete(s.inviteTokenByTeamEmailPending, teamEmailKey(invite.TeamID, invite.Email))
		return ports.Membership{}, domainerrors.ErrInviteExpired
	}

	userProjection := s.usersByID[actorUserID]
	if strings.ToLower(strings.TrimSpace(userProjection.Email)) != strings.ToLower(strings.TrimSpace(invite.Email)) {
		return ports.Membership{}, domainerrors.ErrForbidden
	}

	if _, exists := s.memberIDByTeamUser[teamUserKey(invite.TeamID, actorUserID)]; exists {
		return ports.Membership{}, domainerrors.ErrConflict
	}

	memberID := "member_" + s.nextID("m87")
	member := ports.TeamMember{
		MemberID: memberID,
		TeamID:   invite.TeamID,
		UserID:   actorUserID,
		Role:     invite.Role,
		Status:   "active",
		JoinedAt: now,
	}
	s.membersByID[memberID] = member
	s.memberIDByTeamUser[teamUserKey(invite.TeamID, actorUserID)] = memberID

	invite.Status = "accepted"
	invite.AcceptedBy = actorUserID
	invite.AcceptedAt = &now
	invite.UpdatedAt = now
	s.invitesByToken[invite.Token] = invite
	delete(s.inviteTokenByTeamEmailPending, teamEmailKey(invite.TeamID, invite.Email))

	s.addAuditLocked(invite.TeamID, actorUserID, "team.invite.accepted", "invite", invite.InviteID, now, map[string]string{
		"member_id": memberID,
	})
	s.addAuditLocked(invite.TeamID, actorUserID, "team.member.added", "member", memberID, now, map[string]string{
		"role": member.Role,
	})

	return ports.Membership{
		TeamID:      invite.TeamID,
		UserID:      actorUserID,
		Role:        member.Role,
		Permissions: ports.PermissionsForRole(member.Role),
	}, nil
}

func (s *Store) UpdateMemberRole(
	ctx context.Context,
	actorUserID string,
	teamID string,
	memberID string,
	newRole string,
	mfaCode string,
	now time.Time,
) (ports.TeamMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUserExistsLocked(actorUserID); err != nil {
		return ports.TeamMember{}, err
	}
	if !ports.IsValidRole(newRole) {
		return ports.TeamMember{}, domainerrors.ErrInvalidRequest
	}

	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.TeamMember{}, domainerrors.ErrTeamNotFound
	}
	actorMember, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID)
	if err != nil {
		return ports.TeamMember{}, err
	}
	if actorMember.Role == ports.RoleOwner && strings.TrimSpace(mfaCode) == "" {
		return ports.TeamMember{}, domainerrors.ErrMFARequired
	}

	member, ok := s.membersByID[strings.TrimSpace(memberID)]
	if !ok || member.TeamID != team.TeamID {
		return ports.TeamMember{}, domainerrors.ErrMemberNotFound
	}
	if member.Status != "active" {
		return ports.TeamMember{}, domainerrors.ErrConflict
	}
	if member.Role == ports.RoleOwner && newRole != ports.RoleOwner {
		return ports.TeamMember{}, domainerrors.ErrOwnerTransferRequired
	}
	if actorMember.Role != ports.RoleOwner && member.Role == ports.RoleOwner {
		return ports.TeamMember{}, domainerrors.ErrForbidden
	}

	member.Role = newRole
	s.membersByID[member.MemberID] = member
	now = now.UTC()
	s.addAuditLocked(team.TeamID, actorUserID, "team.role.changed", "member", member.MemberID, now, map[string]string{
		"new_role": newRole,
	})
	return cloneMember(member), nil
}

func (s *Store) RemoveMember(
	ctx context.Context,
	actorUserID string,
	teamID string,
	memberID string,
	mfaCode string,
	now time.Time,
) (ports.TeamMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureUserExistsLocked(actorUserID); err != nil {
		return ports.TeamMember{}, err
	}
	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.TeamMember{}, domainerrors.ErrTeamNotFound
	}
	actorMember, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID)
	if err != nil {
		return ports.TeamMember{}, err
	}
	if actorMember.Role == ports.RoleOwner && strings.TrimSpace(mfaCode) == "" {
		return ports.TeamMember{}, domainerrors.ErrMFARequired
	}

	member, ok := s.membersByID[strings.TrimSpace(memberID)]
	if !ok || member.TeamID != team.TeamID {
		return ports.TeamMember{}, domainerrors.ErrMemberNotFound
	}
	if member.Role == ports.RoleOwner {
		return ports.TeamMember{}, domainerrors.ErrOwnerTransferRequired
	}
	if member.Status != "active" {
		return ports.TeamMember{}, domainerrors.ErrConflict
	}

	now = now.UTC()
	member.Status = "removed"
	member.RemovedAt = &now
	s.membersByID[member.MemberID] = member
	delete(s.memberIDByTeamUser, teamUserKey(member.TeamID, member.UserID))
	s.addAuditLocked(team.TeamID, actorUserID, "team.member.removed", "member", member.MemberID, now, map[string]string{
		"user_id": member.UserID,
	})
	return cloneMember(member), nil
}

func (s *Store) GetTeamDashboard(ctx context.Context, actorUserID string, teamID string) (ports.TeamDashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.TeamDashboard{}, domainerrors.ErrTeamNotFound
	}
	if _, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID); err != nil {
		return ports.TeamDashboard{}, err
	}

	members := make([]ports.TeamMember, 0)
	for _, item := range s.membersByID {
		if item.TeamID == team.TeamID {
			members = append(members, cloneMember(item))
		}
	}
	sort.Slice(members, func(i int, j int) bool {
		return members[i].JoinedAt.Before(members[j].JoinedAt)
	})

	invites := make([]ports.TeamInvite, 0)
	for _, invite := range s.invitesByToken {
		if invite.TeamID == team.TeamID && invite.Status == "pending" {
			invites = append(invites, cloneInvite(invite))
		}
	}
	sort.Slice(invites, func(i int, j int) bool {
		return invites[i].CreatedAt.Before(invites[j].CreatedAt)
	})

	return ports.TeamDashboard{
		Team:           cloneTeam(team),
		Members:        members,
		PendingInvites: invites,
	}, nil
}

func (s *Store) CheckMembership(ctx context.Context, teamID string, userID string) (ports.Membership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.Membership{}, domainerrors.ErrTeamNotFound
	}
	memberID, ok := s.memberIDByTeamUser[teamUserKey(team.TeamID, userID)]
	if !ok {
		return ports.Membership{}, domainerrors.ErrNotFound
	}
	member, ok := s.membersByID[memberID]
	if !ok || member.Status != "active" {
		return ports.Membership{}, domainerrors.ErrNotFound
	}
	return ports.Membership{
		TeamID:      team.TeamID,
		UserID:      member.UserID,
		Role:        member.Role,
		Permissions: ports.PermissionsForRole(member.Role),
	}, nil
}

func (s *Store) ListAuditLogs(ctx context.Context, actorUserID string, teamID string, limit int) ([]ports.TeamAuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return nil, domainerrors.ErrTeamNotFound
	}
	if _, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID); err != nil {
		return nil, err
	}

	items := append([]ports.TeamAuditLog(nil), s.auditLogsByTeamID[team.TeamID]...)
	sort.Slice(items, func(i int, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]ports.TeamAuditLog, 0, len(items))
	for _, item := range items {
		out = append(out, cloneAudit(item))
	}
	return out, nil
}

func (s *Store) CreateMembersExport(ctx context.Context, actorUserID string, teamID string, now time.Time) (ports.MemberExportJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	team, ok := s.teamsByID[strings.TrimSpace(teamID)]
	if !ok {
		return ports.MemberExportJob{}, domainerrors.ErrTeamNotFound
	}
	if _, err := s.requireManagerOrOwnerLocked(team.TeamID, actorUserID); err != nil {
		return ports.MemberExportJob{}, err
	}

	now = now.UTC()
	return ports.MemberExportJob{
		ExportJobID:           "export_" + s.peekID("m87"),
		TeamID:                team.TeamID,
		Status:                "queued",
		CreatedAt:             now,
		EstimatedCompletionAt: now.Add(30 * time.Second),
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
	return s.nextID("m87"), nil
}

func (s *Store) Now() time.Time {
	return time.Now().UTC()
}

func (s *Store) nextID(prefix string) string {
	n := atomic.AddUint64(&s.sequence, 1)
	return fmt.Sprintf("%s_%d", prefix, n)
}

func (s *Store) peekID(prefix string) string {
	n := atomic.LoadUint64(&s.sequence) + 1
	return fmt.Sprintf("%s_%d", prefix, n)
}

func (s *Store) addAuditLocked(teamID string, actorUserID string, action string, targetType string, targetID string, now time.Time, metadata map[string]string) {
	record := ports.TeamAuditLog{
		AuditID:     "audit_" + s.nextID("m87"),
		TeamID:      teamID,
		ActorUserID: actorUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Metadata:    cloneMetadata(metadata),
		CreatedAt:   now.UTC(),
	}
	s.auditLogsByTeamID[teamID] = append(s.auditLogsByTeamID[teamID], record)
}

func (s *Store) requireManagerOrOwnerLocked(teamID string, actorUserID string) (ports.TeamMember, error) {
	memberID, ok := s.memberIDByTeamUser[teamUserKey(teamID, actorUserID)]
	if !ok {
		return ports.TeamMember{}, domainerrors.ErrForbidden
	}
	member, ok := s.membersByID[memberID]
	if !ok || member.Status != "active" {
		return ports.TeamMember{}, domainerrors.ErrForbidden
	}
	if member.Role != ports.RoleOwner && member.Role != ports.RoleManager {
		return ports.TeamMember{}, domainerrors.ErrForbidden
	}
	return member, nil
}

func (s *Store) ensureUserExistsLocked(userID string) error {
	if _, ok := s.usersByID[strings.TrimSpace(userID)]; !ok {
		return domainerrors.ErrDependencyUnavailable
	}
	return nil
}

func normalizeEmail(email string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return "", domainerrors.ErrInvalidRequest
	}
	normalized := strings.ToLower(strings.TrimSpace(addr.Address))
	if normalized == "" {
		return "", domainerrors.ErrInvalidRequest
	}
	return normalized, nil
}

func isValidTeamName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) < 3 || len(trimmed) > 120 {
		return false
	}
	return teamNamePattern.MatchString(trimmed)
}

func teamUserKey(teamID string, userID string) string {
	return strings.TrimSpace(teamID) + "|" + strings.TrimSpace(userID)
}

func teamEmailKey(teamID string, email string) string {
	return strings.TrimSpace(teamID) + "|" + strings.ToLower(strings.TrimSpace(email))
}

func cloneMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneTeam(in ports.Team) ports.Team {
	return in
}

func cloneMember(in ports.TeamMember) ports.TeamMember {
	out := in
	if in.LastActiveAt != nil {
		t := in.LastActiveAt.UTC()
		out.LastActiveAt = &t
	}
	if in.RemovedAt != nil {
		t := in.RemovedAt.UTC()
		out.RemovedAt = &t
	}
	return out
}

func cloneInvite(in ports.TeamInvite) ports.TeamInvite {
	out := in
	if in.AcceptedAt != nil {
		t := in.AcceptedAt.UTC()
		out.AcceptedAt = &t
	}
	return out
}

func cloneAudit(in ports.TeamAuditLog) ports.TeamAuditLog {
	out := in
	out.Metadata = cloneMetadata(in.Metadata)
	return out
}

var _ ports.Repository = (*Store)(nil)
var _ ports.IdempotencyStore = (*Store)(nil)
var _ ports.Clock = (*Store)(nil)
var _ ports.IDGenerator = (*Store)(nil)
