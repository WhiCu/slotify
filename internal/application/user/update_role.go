package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type UpdateRole struct {
	log   *slog.Logger
	users UserRepository
}

func NewUpdateRole(
	log *slog.Logger,
	users UserRepository,
) *UpdateRole {
	return &UpdateRole{
		log:   log,
		users: users,
	}
}

type UpdateRoleInput struct {
	TargetUserID domainuser.UserID
	NewRole      string
	ActorID      domainuser.UserID
}

type UpdateRoleOutput struct {
	ID      domainuser.UserID
	NewRole string
}

func (u *UpdateRole) Execute(ctx context.Context, in UpdateRoleInput) (*UpdateRoleOutput, error) {
	u.log.DebugContext(ctx, "executing update role",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
		slog.String("new_role", in.NewRole),
	)

	newRole, err := domainuser.RoleFromString(in.NewRole)
	if err != nil {
		u.log.WarnContext(ctx, "invalid role provided",
			slog.String("role", in.NewRole),
			slog.Any("error", err),
		)
		return nil, domain.ErrInvalidArgument(err)
	}

	actor, err := u.users.FindByID(ctx, in.ActorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find actor: %w", err)
	}

	if !actor.IsAdmin() {
		u.log.WarnContext(ctx, "non-admin attempted to update role",
			slog.String("actor_id", in.ActorID.String()),
		)
		return nil, ErrNotAdmin
	}

	target, err := u.users.FindByID(ctx, in.TargetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			u.log.WarnContext(ctx, "target user not found",
				slog.String("target_user_id", in.TargetUserID.String()),
			)
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find target user: %w", err)
	}

	if err = target.ChangeRole(newRole, in.ActorID); err != nil {
		u.log.WarnContext(ctx, "domain rejected role change",
			slog.String("target_user_id", in.TargetUserID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	if err = u.users.Save(ctx, target); err != nil {
		u.log.ErrorContext(ctx, "failed to save user after role change",
			slog.String("target_user_id", in.TargetUserID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("save user: %w", err)
	}

	u.log.InfoContext(ctx, "user role updated",
		slog.String("target_user_id", target.ID().String()),
		slog.String("new_role", target.Role().String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return &UpdateRoleOutput{
		ID:      target.ID(),
		NewRole: target.Role().String(),
	}, nil
}
