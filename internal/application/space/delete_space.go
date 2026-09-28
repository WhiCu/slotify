package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type DeleteSpaceUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type DeleteSpaceFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainspace.SpaceID,
	) (*domainspace.Space, error)
}

type DeleteSpaceDeleter interface {
	Delete(ctx context.Context, id domainspace.SpaceID) error
}

type DeleteSpace struct {
	log        *slog.Logger
	users      DeleteSpaceUserFinder
	spaces     DeleteSpaceFinder
	deleter    DeleteSpaceDeleter
	transactor Transactor
}

func NewDeleteSpace(
	log *slog.Logger,
	users DeleteSpaceUserFinder,
	spaces DeleteSpaceFinder,
	deleter DeleteSpaceDeleter,
	transactor Transactor,
) *DeleteSpace {
	return &DeleteSpace{
		log:        log,
		users:      users,
		spaces:     spaces,
		deleter:    deleter,
		transactor: transactor,
	}
}

type DeleteSpaceInput struct {
	SpaceID domainspace.SpaceID
	ActorID domainuser.UserID
}

func (d *DeleteSpace) Execute(
	ctx context.Context,
	in DeleteSpaceInput,
) error {
	d.log.DebugContext(
		ctx,
		"executing delete space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	err := d.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, err := d.users.FindByIDForUpdate(ctx, in.ActorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("find actor: %w", err)
		}

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		if _, errFind := d.spaces.FindByIDForUpdate(ctx, in.SpaceID); errFind != nil {
			if errors.Is(errFind, domain.ErrNotFound) {
				return ErrSpaceNotFound
			}

			return fmt.Errorf("find space: %w", errFind)
		}

		if errDelete := d.deleter.Delete(ctx, in.SpaceID); errDelete != nil {
			return fmt.Errorf("delete space: %w", errDelete)
		}

		return nil
	})

	if err != nil {
		return err
	}

	d.log.InfoContext(
		ctx,
		"space deleted",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return nil
}
