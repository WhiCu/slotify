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

type DeleteSpace struct {
	log        *slog.Logger
	spaces     SpaceRepository
	users      UserRepository
	transactor Transactor
}

func NewDeleteSpace(
	log *slog.Logger,
	spaces SpaceRepository,
	users UserRepository,
	transactor Transactor,
) *DeleteSpace {
	return &DeleteSpace{
		log:        log,
		spaces:     spaces,
		users:      users,
		transactor: transactor,
	}
}

type DeleteSpaceInput struct {
	SpaceID domainspace.SpaceID
	ActorID domainuser.UserID
}

func (d *DeleteSpace) Execute(ctx context.Context, in DeleteSpaceInput) error {
	d.log.DebugContext(ctx, "executing delete space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	err := d.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, txErr := d.users.FindByID(ctx, in.ActorID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrUserNotFound
			}
			return fmt.Errorf("find actor: %w", txErr)
		}

		if !actor.IsAdmin() {
			d.log.WarnContext(ctx, "non-admin attempted to delete space",
				slog.String("actor_id", in.ActorID.String()),
			)
			return ErrNotAdmin
		}

		_, txErr = d.spaces.FindByID(ctx, in.SpaceID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				d.log.WarnContext(ctx, "space not found", slog.String("space_id", in.SpaceID.String()))
				return ErrSpaceNotFound
			}
			return fmt.Errorf("find space: %w", txErr)
		}

		if txErr = d.spaces.Delete(ctx, in.SpaceID); txErr != nil {
			d.log.ErrorContext(ctx, "failed to delete space",
				slog.String("space_id", in.SpaceID.String()),
				slog.Any("error", txErr),
			)
			return fmt.Errorf("delete space: %w", txErr)
		}

		return nil
	})

	if err != nil {
		d.log.ErrorContext(ctx, "delete space failed", slog.Any("error", err))
		return err
	}

	d.log.InfoContext(ctx, "space deleted",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return nil
}
