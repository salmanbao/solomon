package entities

import "time"

// PermissionDecision is returned by permission check APIs.
type PermissionDecision struct {
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
	CheckedAt  time.Time `json:"checked_at"`
	CacheHit   bool      `json:"cache_hit"`
}
