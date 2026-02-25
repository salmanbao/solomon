package entities

// Role models a permission bundle that can be assigned to users.
type Role struct {
	RoleID      string   `json:"role_id"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}
