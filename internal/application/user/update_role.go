package user

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type UpdateRoleFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type UpdateRoleSaver interface {
	Save(ctx context.Context, u *domainuser.User) error
}

type UpdateRole struct {
	log        *slog.Logger
	users      UpdateRoleFinder
	saver      UpdateRoleSaver
	transactor Transactor
}

func NewUpdateRole(
	log *slog.Logger,
	users UpdateRoleFinder,
	saver UpdateRoleSaver,
	transactor Transactor,
) *UpdateRole {
	return &UpdateRole{
		log:        log,
		users:      users,
		saver:      saver,
		transactor: transactor,
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

func (u *UpdateRole) Execute(
	ctx context.Context,
	in UpdateRoleInput,
) (*UpdateRoleOutput, error) {
	u.log.DebugContext(
		ctx,
		"executing update role",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
		slog.String("new_role", in.NewRole),
	)

	newRole, err := domainuser.RoleFromString(in.NewRole)
	if err != nil {
		return nil, domain.ErrInvalidArgument(err)
	}

	var target *domainuser.User

	err = u.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, t, loadErr := u.loadUsersForUpdate(
			ctx,
			in.ActorID,
			in.TargetUserID,
		)
		if loadErr != nil {
			return loadErr
		}
		target = t

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		if errChangeRole := target.ChangeRole(newRole, in.ActorID); errChangeRole != nil {
			return errChangeRole
		}

		if errSave := u.saver.Save(ctx, target); errSave != nil {
			return fmt.Errorf("save user: %w", errSave)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	u.log.InfoContext(
		ctx,
		"user role updated",
		slog.String("target_user_id", target.ID().String()),
		slog.String("new_role", target.Role().String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return &UpdateRoleOutput{
		ID:      target.ID(),
		NewRole: target.Role().String(),
	}, nil
}

func (u *UpdateRole) loadUsersForUpdate(
	ctx context.Context,
	actorID domainuser.UserID,
	targetID domainuser.UserID,
) (*domainuser.User, *domainuser.User, error) {
	if actorID == targetID {
		actor, err := u.users.FindByIDForUpdate(ctx, actorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, nil, ErrUserNotFound
			}

			return nil, nil, fmt.Errorf("find actor: %w", err)
		}

		return actor, actor, nil
	}

	firstID, secondID := actorID, targetID
	// Всегда берём блокировки в одном порядке
	// Иначе два конкурентных запроса A -> B и B -> A
	if bytes.Compare(firstID[:], secondID[:]) > 0 {
		firstID, secondID = secondID, firstID
	}

	first, err := u.users.FindByIDForUpdate(ctx, firstID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUserNotFound
		}

		return nil, nil, fmt.Errorf("find first user: %w", err)
	}

	second, err := u.users.FindByIDForUpdate(ctx, secondID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUserNotFound
		}

		return nil, nil, fmt.Errorf("find second user: %w", err)
	}

	var actor, target *domainuser.User

	if first.ID() == actorID {
		actor = first
		target = second
	} else {
		actor = second
		target = first
	}

	return actor, target, nil
}
