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

type UpdateSpace struct {
	log        *slog.Logger
	spaces     SpaceRepository
	users      UserRepository
	transactor Transactor
}

func NewUpdateSpace(
	log *slog.Logger,
	spaces SpaceRepository,
	users UserRepository,
	transactor Transactor,
) *UpdateSpace {
	return &UpdateSpace{
		log:        log,
		spaces:     spaces,
		users:      users,
		transactor: transactor,
	}
}

type UpdateSpaceInput struct {
	SpaceID  domainspace.SpaceID
	Name     *string
	Capacity *int
	ActorID  domainuser.UserID
}

type UpdateSpaceOutput struct {
	ID       domainspace.SpaceID
	Name     string
	Type     string
	Capacity int
	Active   bool
}

func (u *UpdateSpace) Execute(ctx context.Context, in UpdateSpaceInput) (*UpdateSpaceOutput, error) {
	u.log.DebugContext(ctx, "executing update space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	var out UpdateSpaceOutput

	err := u.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, txErr := u.users.FindByID(ctx, in.ActorID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrUserNotFound
			}
			return fmt.Errorf("find actor: %w", txErr)
		}

		if !actor.IsAdmin() {
			u.log.WarnContext(ctx, "non-admin attempted to update space",
				slog.String("actor_id", in.ActorID.String()),
			)
			return ErrNotAdmin
		}

		s, txErr := u.spaces.FindByID(ctx, in.SpaceID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				u.log.WarnContext(ctx, "space not found", slog.String("space_id", in.SpaceID.String()))
				return ErrSpaceNotFound
			}
			return fmt.Errorf("find space: %w", txErr)
		}

		if in.Name != nil {
			if txErr = s.Rename(*in.Name); txErr != nil {
				u.log.WarnContext(ctx, "domain rejected rename",
					slog.String("space_id", in.SpaceID.String()),
					slog.Any("error", txErr),
				)
				return txErr
			}
		}

		if in.Capacity != nil {
			if txErr = s.ChangeCapacity(*in.Capacity); txErr != nil {
				u.log.WarnContext(ctx, "domain rejected capacity change",
					slog.String("space_id", in.SpaceID.String()),
					slog.Any("error", txErr),
				)
				return txErr
			}
		}

		if txErr = u.spaces.Save(ctx, s); txErr != nil {
			u.log.ErrorContext(ctx, "failed to save space",
				slog.String("space_id", s.ID().String()),
				slog.Any("error", txErr),
			)
			return fmt.Errorf("save space: %w", txErr)
		}

		out = UpdateSpaceOutput{
			ID:       s.ID(),
			Name:     s.Name(),
			Type:     s.Type().String(),
			Capacity: s.Capacity(),
			Active:   s.IsActive(),
		}

		return nil
	})

	if err != nil {
		u.log.ErrorContext(ctx, "update space failed", slog.Any("error", err))
		return nil, err
	}

	u.log.InfoContext(ctx, "space updated",
		slog.String("space_id", out.ID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return &out, nil
}
