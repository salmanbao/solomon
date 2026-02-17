package httpadapter

import (
	"context"

	"solomon/contexts/identity-access/authorization-service/application/commands"
)

// Handler is the transport adapter.
// It maps HTTP DTOs to use-case commands and returns DTO responses.
type Handler struct {
	Assign commands.AssignRoleUseCase
}

func (h Handler) AssignRole(ctx context.Context, userID string, roleID string) error {
	cmd := commands.AssignRoleCommand{UserID: userID, RoleID: roleID}
	return h.Assign.Execute(ctx, cmd)
}
