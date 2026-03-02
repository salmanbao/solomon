package http

import (
	"context"
	"time"

	"solomon/contexts/internal-ops/admin-dashboard-service/application"
	httptransport "solomon/contexts/internal-ops/admin-dashboard-service/transport/http"
)

type Handler struct {
	Service application.Service
}

func (h Handler) RecordAdminActionHandler(
	ctx context.Context,
	adminID string,
	idempotencyKey string,
	req httptransport.RecordAdminActionRequest,
) (httptransport.RecordAdminActionResponse, error) {
	row, err := h.Service.RecordAdminAction(ctx, idempotencyKey, application.RecordActionInput{
		ActorID:       adminID,
		Action:        req.Action,
		TargetID:      req.TargetID,
		Justification: req.Justification,
		SourceIP:      req.SourceIP,
		CorrelationID: req.CorrelationID,
	})
	if err != nil {
		return httptransport.RecordAdminActionResponse{}, err
	}
	return httptransport.RecordAdminActionResponse{
		AuditID:    row.AuditID,
		OccurredAt: row.OccurredAt.UTC().Format(time.RFC3339),
	}, nil
}
