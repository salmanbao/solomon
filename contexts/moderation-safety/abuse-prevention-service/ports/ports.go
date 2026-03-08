package ports

import (
	"context"
	"time"
)

type IdempotencyRecord struct {
	Key          string
	RequestHash  string
	ResponseBody []byte
	ExpiresAt    time.Time
}

type IdempotencyStore interface {
	Get(ctx context.Context, key string, now time.Time) (*IdempotencyRecord, error)
	Reserve(ctx context.Context, key, requestHash string, expiresAt time.Time) error
	Complete(ctx context.Context, key string, responseBody []byte, at time.Time) error
}

type LockoutRelease struct {
	ThreatID   string
	UserID     string
	Status     string
	ReleasedAt time.Time
}

type AuditLog struct {
	AuditID       string
	ActorID       string
	Action        string
	TargetID      string
	Justification string
	OccurredAt    time.Time
	SourceIP      string
	CorrelationID string
}

type Repository interface {
	ReleaseLockout(ctx context.Context, userID string, releasedAt time.Time) (LockoutRelease, error)
	AppendAuditLog(ctx context.Context, row AuditLog) error
	ListRecentAuditLogs(ctx context.Context, limit int) ([]AuditLog, error)
}

type Clock interface {
	Now() time.Time
}
