package commands

import "context"

// AssignRoleCommand is transport-agnostic input for the use case.
type AssignRoleCommand struct {
	UserID string
	RoleID string
}

// AssignmentRepository is a port consumed by the use case.
type AssignmentRepository interface {
	AssignRole(ctx context.Context, userID string, roleID string) error
}

// EventPublisher is a port for post-commit publication (typically via outbox).
type EventPublisher interface {
	PublishRoleAssigned(ctx context.Context, userID string, roleID string) error
}

// AssignRoleUseCase orchestrates command handling.
type AssignRoleUseCase struct {
	Repo      AssignmentRepository
	Publisher EventPublisher
}

func (u AssignRoleUseCase) Execute(ctx context.Context, cmd AssignRoleCommand) error {
	if err := u.Repo.AssignRole(ctx, cmd.UserID, cmd.RoleID); err != nil {
		return err
	}
	return u.Publisher.PublishRoleAssigned(ctx, cmd.UserID, cmd.RoleID)
}
