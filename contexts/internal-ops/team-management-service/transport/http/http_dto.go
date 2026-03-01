package http

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CreateTeamRequest struct {
	Name         string `json:"name"`
	OrgID        string `json:"org_id"`
	StorefrontID string `json:"storefront_id,omitempty"`
}

type CreateTeamResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID      string `json:"team_id"`
		Name        string `json:"name"`
		OwnerUserID string `json:"owner_user_id"`
	} `json:"data"`
}

type CreateInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type CreateInviteResponse struct {
	Status string `json:"status"`
	Data   struct {
		InviteID  string `json:"invite_id"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		Status    string `json:"status"`
		ExpiresAt string `json:"expires_at"`
	} `json:"data"`
}

type AcceptInviteResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID      string   `json:"team_id"`
		UserID      string   `json:"user_id"`
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type UpdateMemberRoleResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID   string `json:"team_id"`
		MemberID string `json:"member_id"`
		Role     string `json:"role"`
	} `json:"data"`
}

type RemoveMemberResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID    string `json:"team_id"`
		MemberID  string `json:"member_id"`
		Status    string `json:"status"`
		RemovedAt string `json:"removed_at,omitempty"`
	} `json:"data"`
}

type GetTeamResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID       string `json:"team_id"`
		Name         string `json:"name"`
		OrgID        string `json:"org_id"`
		StorefrontID string `json:"storefront_id,omitempty"`
		OwnerUserID  string `json:"owner_user_id"`
		Status       string `json:"status"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		Members      []struct {
			MemberID string `json:"member_id"`
			UserID   string `json:"user_id"`
			Role     string `json:"role"`
			Status   string `json:"status"`
			JoinedAt string `json:"joined_at"`
		} `json:"members"`
		PendingInvites []struct {
			InviteID  string `json:"invite_id"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			Status    string `json:"status"`
			ExpiresAt string `json:"expires_at"`
		} `json:"pending_invites"`
	} `json:"data"`
}

type MembershipResponse struct {
	Status string `json:"status"`
	Data   struct {
		TeamID      string   `json:"team_id"`
		UserID      string   `json:"user_id"`
		Role        string   `json:"role"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

type AuditLogsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Items []struct {
			AuditID     string            `json:"audit_id"`
			ActorUserID string            `json:"actor_user_id"`
			Action      string            `json:"action"`
			TargetType  string            `json:"target_type"`
			TargetID    string            `json:"target_id,omitempty"`
			Metadata    map[string]string `json:"metadata,omitempty"`
			CreatedAt   string            `json:"created_at"`
		} `json:"items"`
	} `json:"data"`
}

type ExportMembersResponse struct {
	Status string `json:"status"`
	Data   struct {
		ExportJobID           string `json:"export_job_id"`
		TeamID                string `json:"team_id"`
		Status                string `json:"status"`
		CreatedAt             string `json:"created_at"`
		EstimatedCompletionAt string `json:"estimated_completion_at"`
	} `json:"data"`
}
