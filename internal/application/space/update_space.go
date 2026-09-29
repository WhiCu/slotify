package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type UpdateSpaceUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type UpdateSpaceFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainspace.SpaceID,
	) (*domainspace.Space, error)
}

type UpdateSpaceSaver interface {
	Save(ctx context.Context, s *domainspace.Space) error
}

type UpdateSpace struct {
	log        *slog.Logger
	users      UpdateSpaceUserFinder
	spaces     UpdateSpaceFinder
	saver      UpdateSpaceSaver
	transactor Transactor
}

func NewUpdateSpace(
	log *slog.Logger,
	users UpdateSpaceUserFinder,
	spaces UpdateSpaceFinder,
	saver UpdateSpaceSaver,
	transactor Transactor,
) *UpdateSpace {
	return &UpdateSpace{
		log:        log,
		users:      users,
		spaces:     spaces,
		saver:      saver,
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
	ID        domainspace.SpaceID
	Name      string
	Type      string
	Capacity  int
	Active    bool
	CreatedAt time.Time
}

func (u *UpdateSpace) Execute(
	ctx context.Context,
	in UpdateSpaceInput,
) (*UpdateSpaceOutput, error) {
	u.log.DebugContext(
		ctx,
		"executing update space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	var out UpdateSpaceOutput

	err := u.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, err := u.users.FindByIDForUpdate(ctx, in.ActorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("find actor: %w", err)
		}

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		s, err := u.spaces.FindByIDForUpdate(ctx, in.SpaceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrSpaceNotFound
			}

			return fmt.Errorf("find space: %w", err)
		}

		if in.Name != nil {
			if errRename := s.Rename(*in.Name); errRename != nil {
				return errRename
			}
		}

		if in.Capacity != nil {
			if errChangeCapacity := s.ChangeCapacity(*in.Capacity); errChangeCapacity != nil {
				return errChangeCapacity
			}
		}

		if errSave := u.saver.Save(ctx, s); errSave != nil {
			return fmt.Errorf("save space: %w", errSave)
		}

		out = UpdateSpaceOutput{
			ID:        s.ID(),
			Name:      s.Name(),
			Type:      s.Type().String(),
			Capacity:  s.Capacity(),
			Active:    s.IsActive(),
			CreatedAt: s.CreatedAt(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	u.log.InfoContext(
		ctx,
		"space updated",
		slog.String("space_id", out.ID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return &out, nil
}
