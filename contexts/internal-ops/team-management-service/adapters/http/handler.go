package httpadapter

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"solomon/contexts/internal-ops/team-management-service/application"
	"solomon/contexts/internal-ops/team-management-service/ports"
	httptransport "solomon/contexts/internal-ops/team-management-service/transport/http"
)

type Handler struct {
	Service application.Service
	Logger  *slog.Logger
}

func (h Handler) CreateTeamHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	req httptransport.CreateTeamRequest,
) (httptransport.CreateTeamResponse, error) {
	item, err := h.Service.CreateTeam(ctx, idempotencyKey, actorUserID, ports.CreateTeamInput{
		Name:         strings.TrimSpace(req.Name),
		OrgID:        strings.TrimSpace(req.OrgID),
		StorefrontID: strings.TrimSpace(req.StorefrontID),
	})
	if err != nil {
		return httptransport.CreateTeamResponse{}, err
	}

	resp := httptransport.CreateTeamResponse{Status: "success"}
	resp.Data.TeamID = item.TeamID
	resp.Data.Name = item.Name
	resp.Data.OwnerUserID = item.OwnerUserID
	return resp, nil
}

func (h Handler) CreateInviteHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	req httptransport.CreateInviteRequest,
) (httptransport.CreateInviteResponse, error) {
	item, err := h.Service.CreateInvite(ctx, idempotencyKey, actorUserID, strings.TrimSpace(teamID), application.CreateInviteInput{
		Email: strings.TrimSpace(req.Email),
		Role:  strings.TrimSpace(req.Role),
	})
	if err != nil {
		return httptransport.CreateInviteResponse{}, err
	}

	resp := httptransport.CreateInviteResponse{Status: "success"}
	resp.Data.InviteID = item.InviteID
	resp.Data.Email = item.Email
	resp.Data.Role = item.Role
	resp.Data.Status = item.Status
	resp.Data.ExpiresAt = item.ExpiresAt.UTC().Format(time.RFC3339)
	return resp, nil
}

func (h Handler) AcceptInviteHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	token string,
) (httptransport.AcceptInviteResponse, error) {
	item, err := h.Service.AcceptInvite(ctx, idempotencyKey, actorUserID, strings.TrimSpace(token))
	if err != nil {
		return httptransport.AcceptInviteResponse{}, err
	}

	resp := httptransport.AcceptInviteResponse{Status: "success"}
	resp.Data.TeamID = item.TeamID
	resp.Data.UserID = item.UserID
	resp.Data.Role = item.Role
	resp.Data.Permissions = append([]string(nil), item.Permissions...)
	return resp, nil
}

func (h Handler) UpdateMemberRoleHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	memberID string,
	mfaCode string,
	req httptransport.UpdateMemberRoleRequest,
) (httptransport.UpdateMemberRoleResponse, error) {
	item, err := h.Service.UpdateMemberRole(
		ctx,
		idempotencyKey,
		actorUserID,
		strings.TrimSpace(teamID),
		strings.TrimSpace(memberID),
		strings.TrimSpace(req.Role),
		strings.TrimSpace(mfaCode),
	)
	if err != nil {
		return httptransport.UpdateMemberRoleResponse{}, err
	}
	resp := httptransport.UpdateMemberRoleResponse{Status: "success"}
	resp.Data.TeamID = item.TeamID
	resp.Data.MemberID = item.MemberID
	resp.Data.Role = item.Role
	return resp, nil
}

func (h Handler) RemoveMemberHandler(
	ctx context.Context,
	idempotencyKey string,
	actorUserID string,
	teamID string,
	memberID string,
	mfaCode string,
) (httptransport.RemoveMemberResponse, error) {
	item, err := h.Service.RemoveMember(
		ctx,
		idempotencyKey,
		actorUserID,
		strings.TrimSpace(teamID),
		strings.TrimSpace(memberID),
		strings.TrimSpace(mfaCode),
	)
	if err != nil {
		return httptransport.RemoveMemberResponse{}, err
	}
	resp := httptransport.RemoveMemberResponse{Status: "success"}
	resp.Data.TeamID = item.TeamID
	resp.Data.MemberID = item.MemberID
	resp.Data.Status = item.Status
	if item.RemovedAt != nil {
		resp.Data.RemovedAt = item.RemovedAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}

func (h Handler) GetTeamDashboardHandler(
	ctx context.Context,
	actorUserID string,
	teamID string,
) (httptransport.GetTeamResponse, error) {
	item, err := h.Service.GetTeamDashboard(ctx, actorUserID, strings.TrimSpace(teamID))
	if err != nil {
		return httptransport.GetTeamResponse{}, err
	}
	resp := httptransport.GetTeamResponse{Status: "success"}
	resp.Data.TeamID = item.Team.TeamID
	resp.Data.Name = item.Team.Name
	resp.Data.OrgID = item.Team.OrgID
	resp.Data.StorefrontID = item.Team.StorefrontID
	resp.Data.OwnerUserID = item.Team.OwnerUserID
	resp.Data.Status = item.Team.Status
	resp.Data.CreatedAt = item.Team.CreatedAt.UTC().Format(time.RFC3339)
	resp.Data.UpdatedAt = item.Team.UpdatedAt.UTC().Format(time.RFC3339)

	for _, member := range item.Members {
		row := struct {
			MemberID string `json:"member_id"`
			UserID   string `json:"user_id"`
			Role     string `json:"role"`
			Status   string `json:"status"`
			JoinedAt string `json:"joined_at"`
		}{
			MemberID: member.MemberID,
			UserID:   member.UserID,
			Role:     member.Role,
			Status:   member.Status,
			JoinedAt: member.JoinedAt.UTC().Format(time.RFC3339),
		}
		resp.Data.Members = append(resp.Data.Members, row)
	}
	for _, invite := range item.PendingInvites {
		row := struct {
			InviteID  string `json:"invite_id"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			Status    string `json:"status"`
			ExpiresAt string `json:"expires_at"`
		}{
			InviteID:  invite.InviteID,
			Email:     invite.Email,
			Role:      invite.Role,
			Status:    invite.Status,
			ExpiresAt: invite.ExpiresAt.UTC().Format(time.RFC3339),
		}
		resp.Data.PendingInvites = append(resp.Data.PendingInvites, row)
	}
	return resp, nil
}

func (h Handler) CheckMembershipHandler(
	ctx context.Context,
	teamID string,
	userID string,
) (httptransport.MembershipResponse, error) {
	item, err := h.Service.CheckMembership(ctx, strings.TrimSpace(teamID), strings.TrimSpace(userID))
	if err != nil {
		return httptransport.MembershipResponse{}, err
	}
	resp := httptransport.MembershipResponse{Status: "success"}
	resp.Data.TeamID = item.TeamID
	resp.Data.UserID = item.UserID
	resp.Data.Role = item.Role
	resp.Data.Permissions = append([]string(nil), item.Permissions...)
	return resp, nil
}

func (h Handler) ListAuditLogsHandler(
	ctx context.Context,
	actorUserID string,
	teamID string,
	limit string,
) (httptransport.AuditLogsResponse, error) {
	pageSize := 50
	if strings.TrimSpace(limit) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(limit)); err == nil {
			pageSize = parsed
		}
	}
	items, err := h.Service.ListAuditLogs(ctx, actorUserID, strings.TrimSpace(teamID), pageSize)
	if err != nil {
		return httptransport.AuditLogsResponse{}, err
	}
	resp := httptransport.AuditLogsResponse{Status: "success"}
	for _, item := range items {
		row := struct {
			AuditID     string            `json:"audit_id"`
			ActorUserID string            `json:"actor_user_id"`
			Action      string            `json:"action"`
			TargetType  string            `json:"target_type"`
			TargetID    string            `json:"target_id,omitempty"`
			Metadata    map[string]string `json:"metadata,omitempty"`
			CreatedAt   string            `json:"created_at"`
		}{
			AuditID:     item.AuditID,
			ActorUserID: item.ActorUserID,
			Action:      item.Action,
			TargetType:  item.TargetType,
			TargetID:    item.TargetID,
			Metadata:    item.Metadata,
			CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		}
		resp.Data.Items = append(resp.Data.Items, row)
	}
	return resp, nil
}

func (h Handler) ExportMembersHandler(
	ctx context.Context,
	actorUserID string,
	teamID string,
) (httptransport.ExportMembersResponse, error) {
	item, err := h.Service.CreateMembersExport(ctx, actorUserID, strings.TrimSpace(teamID))
	if err != nil {
		return httptransport.ExportMembersResponse{}, err
	}
	resp := httptransport.ExportMembersResponse{Status: "success"}
	resp.Data.ExportJobID = item.ExportJobID
	resp.Data.TeamID = item.TeamID
	resp.Data.Status = item.Status
	resp.Data.CreatedAt = item.CreatedAt.UTC().Format(time.RFC3339)
	resp.Data.EstimatedCompletionAt = item.EstimatedCompletionAt.UTC().Format(time.RFC3339)
	return resp, nil
}
