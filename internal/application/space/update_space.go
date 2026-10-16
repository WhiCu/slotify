package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
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
) (output *UpdateSpaceOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "space", "update")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "UpdateSpace.Execute",
		trace.WithAttributes(
			attribute.String("domain", "space"),
			attribute.String("usecase", "update"),
			attribute.String("space_id", in.SpaceID.String()),
			attribute.String("actor_id", in.ActorID.String()),
		),
	)
	defer span.End()

	u.log.DebugContext(
		ctx,
		"executing update space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	var out UpdateSpaceOutput

	err = u.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	u.log.InfoContext(
		ctx,
		"space updated",
		slog.String("space_id", out.ID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)
	span.SetAttributes(
		attribute.String("space.name", out.Name),
		attribute.Int("space.capacity", out.Capacity),
		attribute.Bool("space.active", out.Active),
	)
	span.SetStatus(codes.Ok, "space successfully updated")

	return &out, nil
}
