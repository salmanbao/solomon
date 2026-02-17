package contracts

// Request/response DTOs for HTTP transport.
// Keep DTOs in contracts so adapters do not leak domain internals.
type AssignRoleRequest struct {
	UserID string `json:"user_id"`
	RoleID string `json:"role_id"`
}

type AssignRoleResponse struct {
	Status string `json:"status"`
}
