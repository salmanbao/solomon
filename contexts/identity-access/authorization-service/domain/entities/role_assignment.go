package entities

import "time"

// RoleAssignment captures an active or historical user-role relation.
type RoleAssignment struct {
	AssignmentID string     `json:"assignment_id"`
	UserID       string     `json:"user_id"`
	RoleID       string     `json:"role_id"`
	RoleName     string     `json:"role_name"`
	AssignedBy   string     `json:"assigned_by"`
	Reason       string     `json:"reason"`
	AssignedAt   time.Time  `json:"assigned_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	IsActive     bool       `json:"is_active"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}
