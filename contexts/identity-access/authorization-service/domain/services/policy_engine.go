package services

// GrantsPermission returns true when the permission exists in the effective set.
func GrantsPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
