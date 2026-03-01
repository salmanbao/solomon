package ports

import (
	"context"
	"time"
)

const (
	RoleOwner   = "Owner"
	RoleManager = "Manager"
	RoleEditor  = "Editor"
	RoleSupport = "Support"
	RoleViewer  = "Viewer"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleManager, RoleEditor, RoleSupport, RoleViewer:
		return true
	default:
		return false
	}
}

func PermissionsForRole(role string) []string {
	switch role {
	case RoleOwner:
		return []string{
			"team.read", "team.write", "team.delete",
			"campaigns.read", "campaigns.write",
			"products.read", "products.write",
			"payouts.read", "payouts.write",
			"analytics.read",
		}
	case RoleManager:
		return []string{
			"team.read", "team.write",
			"campaigns.read", "campaigns.write",
			"products.read", "products.write",
			"analytics.read",
		}
	case RoleEditor:
		return []string{
			"team.read",
			"campaigns.read", "campaigns.write",
			"products.read",
			"analytics.read",
		}
	case RoleSupport:
		return []string{
			"team.read",
			"campaigns.read",
			"products.read",
			"payouts.read",
		}
	default:
		return []string{
			"team.read",
			"campaigns.read",
			"products.read",
			"analytics.read",
		}
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

type CreateTeamInput struct {
	Name         string
	OrgID        string
	StorefrontID string
}

type Team struct {
	TeamID       string
	Name         string
	OrgID        string
	StorefrontID string
	OwnerUserID  string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TeamMember struct {
	MemberID     string
	TeamID       string
	UserID       string
	Role         string
	Status       string
	LastActiveAt *time.Time
	JoinedAt     time.Time
	RemovedAt    *time.Time
}

type TeamInvite struct {
	InviteID   string
	TeamID     string
	Email      string
	Role       string
	Token      string
	Status     string
	ExpiresAt  time.Time
	CreatedBy  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	AcceptedBy string
	AcceptedAt *time.Time
}

type TeamAuditLog struct {
	AuditID     string
	TeamID      string
	ActorUserID string
	Action      string
	TargetType  string
	TargetID    string
	Metadata    map[string]string
	CreatedAt   time.Time
}

type Membership struct {
	TeamID      string
	UserID      string
	Role        string
	Permissions []string
}

type TeamDashboard struct {
	Team           Team
	Members        []TeamMember
	PendingInvites []TeamInvite
}

type MemberExportJob struct {
	ExportJobID           string
	TeamID                string
	Status                string
	CreatedAt             time.Time
	EstimatedCompletionAt time.Time
}

type Repository interface {
	CreateTeam(ctx context.Context, actorUserID string, input CreateTeamInput, now time.Time) (Team, error)
	CreateInvite(ctx context.Context, actorUserID string, teamID string, email string, role string, now time.Time) (TeamInvite, error)
	AcceptInvite(ctx context.Context, actorUserID string, token string, now time.Time) (Membership, error)
	UpdateMemberRole(ctx context.Context, actorUserID string, teamID string, memberID string, newRole string, mfaCode string, now time.Time) (TeamMember, error)
	RemoveMember(ctx context.Context, actorUserID string, teamID string, memberID string, mfaCode string, now time.Time) (TeamMember, error)
	GetTeamDashboard(ctx context.Context, actorUserID string, teamID string) (TeamDashboard, error)
	CheckMembership(ctx context.Context, teamID string, userID string) (Membership, error)
	ListAuditLogs(ctx context.Context, actorUserID string, teamID string, limit int) ([]TeamAuditLog, error)
	CreateMembersExport(ctx context.Context, actorUserID string, teamID string, now time.Time) (MemberExportJob, error)
}
