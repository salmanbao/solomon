package httptransport

import "time"

// CheckPermissionRequest is the request body for single-permission evaluation.
type CheckPermissionRequest struct {
	UserID       string `json:"user_id,omitempty"`
	Permission   string `json:"permission"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
}

// CheckBatchRequest is the request body for multi-permission evaluation.
type CheckBatchRequest struct {
	UserID      string   `json:"user_id,omitempty"`
	Permissions []string `json:"permissions"`
}

// CheckPermissionResponse describes one permission decision.
type CheckPermissionResponse struct {
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
	CheckedAt  time.Time `json:"checked_at"`
	CacheHit   bool      `json:"cache_hit"`
}

type CheckBatchResponse struct {
	Results []CheckPermissionResponse `json:"results"`
}

type RoleAssignmentDTO struct {
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

type ListUserRolesResponse struct {
	UserID string              `json:"user_id"`
	Roles  []RoleAssignmentDTO `json:"roles"`
}

type GrantRoleRequest struct {
	RoleID    string     `json:"role_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

type GrantRoleResponse struct {
	AssignmentID string     `json:"assignment_id"`
	UserID       string     `json:"user_id"`
	RoleID       string     `json:"role_id"`
	AssignedAt   time.Time  `json:"assigned_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	AuditLogID   string     `json:"audit_log_id"`
	Replayed     bool       `json:"replayed"`
}

type RevokeRoleRequest struct {
	RoleID string `json:"role_id"`
	Reason string `json:"reason,omitempty"`
}

type RevokeRoleResponse struct {
	UserID     string     `json:"user_id"`
	RoleID     string     `json:"role_id"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	AuditLogID string     `json:"audit_log_id"`
	Replayed   bool       `json:"replayed"`
}

type CreateDelegationRequest struct {
	FromAdminID string    `json:"from_admin_id"`
	ToAdminID   string    `json:"to_admin_id"`
	RoleID      string    `json:"role_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	Reason      string    `json:"reason,omitempty"`
}

type CreateDelegationResponse struct {
	DelegationID string    `json:"delegation_id"`
	FromAdminID  string    `json:"from_admin_id"`
	ToAdminID    string    `json:"to_admin_id"`
	RoleID       string    `json:"role_id"`
	DelegatedAt  time.Time `json:"delegated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	AuditLogID   string    `json:"audit_log_id"`
	Replayed     bool      `json:"replayed"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
