package entities

import "time"

// Delegation is a temporary grant from one admin to another.
type Delegation struct {
	DelegationID string    `json:"delegation_id"`
	FromAdminID  string    `json:"from_admin_id"`
	ToAdminID    string    `json:"to_admin_id"`
	RoleID       string    `json:"role_id"`
	Reason       string    `json:"reason"`
	DelegatedAt  time.Time `json:"delegated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `json:"is_active"`
}
