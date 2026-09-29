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

var (
	ErrCannotDeleteSelf = errors.New(
		"application/user: cannot delete yourself",
	)

	ErrCannotDeleteRoot = errors.New(
		"application/user: cannot delete root user",
	)
)

type DeleteUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type DeleteUserDeleter interface {
	Delete(ctx context.Context, id domainuser.UserID) error
}

type DeleteUserReservationCanceller interface {
	CancelAllByUserID(
		ctx context.Context,
		userID domainuser.UserID,
	) error
}

type DeleteUser struct {
	log          *slog.Logger
	users        DeleteUserFinder
	deleter      DeleteUserDeleter
	reservations DeleteUserReservationCanceller
	transactor   Transactor
}

func NewDeleteUser(
	log *slog.Logger,
	users DeleteUserFinder,
	deleter DeleteUserDeleter,
	reservations DeleteUserReservationCanceller,
	transactor Transactor,
) *DeleteUser {
	return &DeleteUser{
		log:          log,
		users:        users,
		deleter:      deleter,
		reservations: reservations,
		transactor:   transactor,
	}
}

type DeleteUserInput struct {
	TargetUserID domainuser.UserID
	ActorID      domainuser.UserID
}

func (d *DeleteUser) Execute(
	ctx context.Context,
	in DeleteUserInput,
) error {
	d.log.DebugContext(
		ctx,
		"executing delete user",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	if in.TargetUserID == in.ActorID {
		return ErrCannotDeleteSelf
	}

	err := d.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		d.log.DebugContext(
			ctx,
			"start of delete user transaction",
			slog.String("target_user_id", in.TargetUserID.String()),
			slog.String("actor_id", in.ActorID.String()),
		)
		actor, target, err := d.loadUsersForUpdate(
			ctx,
			in.ActorID,
			in.TargetUserID,
		)
		if err != nil {
			return err
		}

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		if errCancel := d.reservations.CancelAllByUserID(ctx, target.ID()); errCancel != nil {
			return fmt.Errorf("cancel reservations: %w", errCancel)
		}

		if errDelete := d.deleter.Delete(ctx, target.ID()); errDelete != nil {
			return fmt.Errorf("delete user: %w", errDelete)
		}

		d.log.DebugContext(
			ctx,
			"end of delete user transaction",
			slog.String("target_user_id", target.ID().String()),
			slog.String("actor_id", actor.ID().String()),
		)
		return nil
	})

	if err != nil {
		d.log.WarnContext(
			ctx,
			"delete user failed",
			slog.String("target_user_id", in.TargetUserID.String()),
			slog.Any("error", err),
		)

		return err
	}

	d.log.InfoContext(
		ctx,
		"user deleted",
		slog.String("target_user_id", in.TargetUserID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return nil
}

func (d *DeleteUser) loadUsersForUpdate(
	ctx context.Context,
	actorID domainuser.UserID,
	targetID domainuser.UserID,
) (*domainuser.User, *domainuser.User, error) {
	firstID, secondID := actorID, targetID

	if bytes.Compare(firstID[:], secondID[:]) > 0 {
		firstID, secondID = secondID, firstID
	}

	first, err := d.users.FindByIDForUpdate(ctx, firstID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrUserNotFound
		}

		return nil, nil, fmt.Errorf("find first user: %w", err)
	}

	second, err := d.users.FindByIDForUpdate(ctx, secondID)
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
