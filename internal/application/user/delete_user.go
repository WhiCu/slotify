package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type DeleteUser struct {
	log          *slog.Logger
	users        UserRepository
	reservations ReservationRepository
	transactor   Transactor
}

func NewDeleteUser(
	log *slog.Logger,
	users UserRepository,
	reservations ReservationRepository,
	transactor Transactor,
) *DeleteUser {
	return &DeleteUser{
		log:          log,
		users:        users,
		reservations: reservations,
		transactor:   transactor,
	}
}

type DeleteUserInput struct {
	TargetUserID domainuser.UserID
	ActorID      domainuser.UserID
}

func (d *DeleteUser) Execute(ctx context.Context, in DeleteUserInput) error {
	d.log.DebugContext(ctx, "executing delete user",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	if in.TargetUserID == in.ActorID {
		d.log.WarnContext(ctx, "actor attempted to delete themselves",
			slog.String("actor_id", in.ActorID.String()),
		)
		return ErrCannotDeleteSelf
	}

	err := d.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, err := d.users.FindByID(ctx, in.ActorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrUserNotFound
			}
			return fmt.Errorf("find actor: %w", err)
		}

		if !actor.IsAdmin() {
			d.log.WarnContext(ctx, "non-admin attempted to delete user",
				slog.String("actor_id", in.ActorID.String()),
			)
			return ErrNotAdmin
		}

		target, err := d.users.FindByID(ctx, in.TargetUserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				d.log.WarnContext(ctx, "target user not found",
					slog.String("target_user_id", in.TargetUserID.String()),
				)
				return ErrUserNotFound
			}
			return fmt.Errorf("find target user: %w", err)
		}

		if cancelErr := d.reservations.CancelAllByUserID(ctx, target.ID()); cancelErr != nil {
			d.log.ErrorContext(ctx, "failed to cancel reservations for user",
				slog.String("user_id", target.ID().String()),
				slog.Any("error", cancelErr),
			)
			return fmt.Errorf("cancel reservations: %w", cancelErr)
		}

		if delErr := d.users.Delete(ctx, target.ID()); delErr != nil {
			d.log.ErrorContext(ctx, "failed to delete user",
				slog.String("user_id", target.ID().String()),
				slog.Any("error", delErr),
			)
			return fmt.Errorf("delete user: %w", delErr)
		}

		return nil
	})

	if err != nil {
		d.log.ErrorContext(ctx, "delete user transaction failed",
			slog.String("target_user_id", in.TargetUserID.String()),
			slog.Any("error", err),
		)
		return err
	}

	d.log.InfoContext(ctx, "user deleted with reservations cancelled",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return nil
}
