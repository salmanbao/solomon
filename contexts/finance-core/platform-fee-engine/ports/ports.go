package ports

import (
	"context"
	"time"

	contractsv1 "solomon/contracts/gen/events/v1"
)

type FeeCalculation struct {
	CalculationID string
	SubmissionID  string
	UserID        string
	CampaignID    string
	GrossAmount   float64
	FeeRate       float64
	FeeAmount     float64
	NetAmount     float64
	CalculatedAt  time.Time
	SourceEventID string
}

type CalculateFeeInput struct {
	SubmissionID  string
	UserID        string
	CampaignID    string
	GrossAmount   float64
	FeeRate       float64
	CalculatedAt  time.Time
	SourceEventID string
}

type RewardPayoutEligibleEvent struct {
	SubmissionID string
	UserID       string
	CampaignID   string
	GrossAmount  float64
	EligibleAt   time.Time
}

type Repository interface {
	CreateCalculation(ctx context.Context, calculation FeeCalculation) error
	GetCalculation(ctx context.Context, calculationID string) (FeeCalculation, error)
	ListCalculationsByUser(ctx context.Context, userID string, limit int, offset int) ([]FeeCalculation, error)
	BuildMonthlyReport(ctx context.Context, month string) (FeeReport, error)
}

type FeeReport struct {
	Month      string
	TotalGross float64
	TotalFee   float64
	TotalNet   float64
	Count      int
}

type IdempotencyRecord struct {
	Key             string
	RequestHash     string
	ResponsePayload []byte
	ExpiresAt       time.Time
}

type IdempotencyStore interface {
	GetRecord(ctx context.Context, key string, now time.Time) (IdempotencyRecord, bool, error)
	PutRecord(ctx context.Context, record IdempotencyRecord) error
}

type EventDedupStore interface {
	ReserveEvent(ctx context.Context, eventID string, payloadHash string, expiresAt time.Time) (bool, error)
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(ctx context.Context) (string, error)
}

type EventEnvelope = contractsv1.Envelope

type OutboxMessage struct {
	OutboxID     string
	EventType    string
	PartitionKey string
	Payload      []byte
	CreatedAt    time.Time
}

type OutboxWriter interface {
	AppendOutbox(ctx context.Context, envelope EventEnvelope) error
}

type OutboxRepository interface {
	ListPendingOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, outboxID string, publishedAt time.Time) error
}
