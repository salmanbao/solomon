package http

import (
	"context"
	"time"

	"solomon/contexts/moderation-safety/abuse-prevention-service/application"
	httptransport "solomon/contexts/moderation-safety/abuse-prevention-service/transport/http"
)

type Handler struct {
	Service application.Service
}

func (h Handler) ReleaseLockoutHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	userID string,
	req httptransport.ReleaseLockoutRequest,
) (httptransport.ReleaseLockoutResponse, error) {
	result, err := h.Service.ReleaseLockout(ctx, idempotencyKey, application.ReleaseLockoutInput{
		ActorID:       adminID,
		UserID:        userID,
		Reason:        req.Reason,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.ReleaseLockoutResponse{}, err
	}
	return httptransport.ReleaseLockoutResponse{
		ThreatID:        result.ThreatID,
		UserID:          result.UserID,
		Status:          result.Status,
		ReleasedAt:      result.ReleasedAt.UTC().Format(time.RFC3339),
		OwnerAuditLogID: result.OwnerAuditLogID,
	}, nil
}
