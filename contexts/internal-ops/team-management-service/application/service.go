package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	domainerrors "solomon/contexts/internal-ops/team-management-service/domain/errors"
	"solomon/contexts/internal-ops/team-management-service/ports"
)

type Service struct {
	Repo           ports.Repository
	Idempotency    ports.IdempotencyStore
	Clock          ports.Clock
	Logger         *slog.Logger
	IdempotencyTTL time.Duration
}

type CreateInviteInput struct {
	Email string
	Role  string
}

func (s Service) CreateTeam(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	input ports.CreateTeamInput,
) (ports.Team, error) {
	var out ports.Team
	if strings.TrimSpace(actorUserID) == "" ||
		strings.TrimSpace(input.Name) == "" ||
		strings.TrimSpace(input.OrgID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m87_create_team", actorUserID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.CreateTeam(ctx, actorUserID, input, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) CreateInvite(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	input CreateInviteInput,
) (ports.TeamInvite, error) {
	var out ports.TeamInvite
	if strings.TrimSpace(actorUserID) == "" ||
		strings.TrimSpace(teamID) == "" ||
		strings.TrimSpace(input.Email) == "" ||
		strings.TrimSpace(input.Role) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(input)
	requestHash := hashStrings("m87_create_invite", actorUserID, teamID, string(payload))
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.CreateInvite(ctx, actorUserID, teamID, input.Email, input.Role, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) AcceptInvite(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	token string,
) (ports.Membership, error) {
	var out ports.Membership
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(token) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m87_accept_invite", actorUserID, token)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.AcceptInvite(ctx, actorUserID, token, s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) UpdateMemberRole(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	memberID string,
	newRole string,
	mfaCode string,
) (ports.TeamMember, error) {
	var out ports.TeamMember
	if strings.TrimSpace(actorUserID) == "" ||
		strings.TrimSpace(teamID) == "" ||
		strings.TrimSpace(memberID) == "" ||
		strings.TrimSpace(newRole) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m87_update_member_role", actorUserID, teamID, memberID, newRole)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.UpdateMemberRole(ctx, actorUserID, teamID, memberID, newRole, strings.TrimSpace(mfaCode), s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) RemoveMember(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	memberID string,
	mfaCode string,
) (ports.TeamMember, error) {
	var out ports.TeamMember
	if strings.TrimSpace(actorUserID) == "" ||
		strings.TrimSpace(teamID) == "" ||
		strings.TrimSpace(memberID) == "" {
		return out, domainerrors.ErrInvalidRequest
	}
	if err := s.requireIdempotency(idempotencyKey); err != nil {
		return out, err
	}
	requestHash := hashStrings("m87_remove_member", actorUserID, teamID, memberID)
	err := s.runIdempotent(
		ctx,
		strings.TrimSpace(idempotencyKey),
		requestHash,
		func(raw []byte) error { return json.Unmarshal(raw, &out) },
		func() ([]byte, error) {
			item, err := s.Repo.RemoveMember(ctx, actorUserID, teamID, memberID, strings.TrimSpace(mfaCode), s.now())
			if err != nil {
				return nil, err
			}
			return json.Marshal(item)
		},
	)
	return out, err
}

func (s Service) GetTeamDashboard(ctx context.Context, actorUserID string, teamID string) (ports.TeamDashboard, error) {
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(teamID) == "" {
		return ports.TeamDashboard{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.GetTeamDashboard(ctx, actorUserID, teamID)
}

func (s Service) CheckMembership(ctx context.Context, teamID string, userID string) (ports.Membership, error) {
	if strings.TrimSpace(teamID) == "" || strings.TrimSpace(userID) == "" {
		return ports.Membership{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.CheckMembership(ctx, teamID, userID)
}

func (s Service) ListAuditLogs(ctx context.Context, actorUserID string, teamID string, limit int) ([]ports.TeamAuditLog, error) {
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(teamID) == "" {
		return nil, domainerrors.ErrInvalidRequest
	}
	return s.Repo.ListAuditLogs(ctx, actorUserID, teamID, limit)
}

func (s Service) CreateMembersExport(ctx context.Context, actorUserID string, teamID string) (ports.MemberExportJob, error) {
	if strings.TrimSpace(actorUserID) == "" || strings.TrimSpace(teamID) == "" {
		return ports.MemberExportJob{}, domainerrors.ErrInvalidRequest
	}
	return s.Repo.CreateMembersExport(ctx, actorUserID, teamID, s.now())
}

func (s Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func (s Service) idempotencyTTL() time.Duration {
	if s.IdempotencyTTL <= 0 {
		return 7 * 24 * time.Hour
	}
	return s.IdempotencyTTL
}

func (s Service) requireIdempotency(key string) error {
	if strings.TrimSpace(key) == "" {
		return domainerrors.ErrIdempotencyKeyRequired
	}
	return nil
}

func (s Service) runIdempotent(
	ctx context.Context,
	key string,
	requestHash string,
	decode func([]byte) error,
	exec func() ([]byte, error),
) error {
	now := s.now()
	record, found, err := s.Idempotency.Get(ctx, key, now)
	if err != nil {
		return err
	}
	if found {
		if record.RequestHash != requestHash {
			return domainerrors.ErrIdempotencyConflict
		}
		return decode(record.Payload)
	}

	payload, err := exec()
	if err != nil {
		return err
	}
	if err := s.Idempotency.Put(ctx, ports.IdempotencyRecord{
		Key:         key,
		RequestHash: requestHash,
		Payload:     payload,
		ExpiresAt:   now.Add(s.idempotencyTTL()),
	}); err != nil {
		return err
	}

	resolveLogger(s.Logger).Debug("team management idempotent operation committed",
		"event", "team_mgmt_idempotent_operation_committed",
		"module", "internal-ops/team-management-service",
		"layer", "application",
		"idempotency_key", key,
	)
	return decode(payload)
}

func hashStrings(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])
}
