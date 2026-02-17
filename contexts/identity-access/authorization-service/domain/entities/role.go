package entities

// Role is a core authorization aggregate.
type Role struct {
	RoleID      string   `json:"role_id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}
