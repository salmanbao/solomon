package services

import "solomon/contexts/identity-access/authorization-service/domain/entities"

// PolicyEngine evaluates whether a role grants a permission.
func PolicyEngine(role entities.Role, permission string) bool {
	for _, p := range role.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
