package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type DeactivateSpaceUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type DeactivateSpaceFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainspace.SpaceID,
	) (*domainspace.Space, error)
}

type DeactivateSpaceSaver interface {
	Save(ctx context.Context, s *domainspace.Space) error
}

type DeactivateSpace struct {
	log        *slog.Logger
	users      DeactivateSpaceUserFinder
	spaces     DeactivateSpaceFinder
	saver      DeactivateSpaceSaver
	transactor Transactor
}

func NewDeactivateSpace(
	log *slog.Logger,
	users DeactivateSpaceUserFinder,
	spaces DeactivateSpaceFinder,
	saver DeactivateSpaceSaver,
	transactor Transactor,
) *DeactivateSpace {
	return &DeactivateSpace{
		log:        log,
		users:      users,
		spaces:     spaces,
		saver:      saver,
		transactor: transactor,
	}
}

type DeactivateSpaceInput struct {
	SpaceID domainspace.SpaceID
	ActorID domainuser.UserID
}

func (d *DeactivateSpace) Execute(
	ctx context.Context,
	in DeactivateSpaceInput,
) (err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "space", "deactivate")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "DeactivateSpace.Execute",
		trace.WithAttributes(
			attribute.String("domain", "space"),
			attribute.String("usecase", "deactivate"),
			attribute.String("space_id", in.SpaceID.String()),
			attribute.String("actor_id", in.ActorID.String()),
		),
	)
	defer span.End()

	d.log.DebugContext(
		ctx,
		"executing deactivate space",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	err = d.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
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

		s, err := d.spaces.FindByIDForUpdate(ctx, in.SpaceID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrSpaceNotFound
			}

			return fmt.Errorf("find space: %w", err)
		}

		if errDeactivate := s.Deactivate(); errDeactivate != nil {
			return errDeactivate
		}

		if errSave := d.saver.Save(ctx, s); errSave != nil {
			return fmt.Errorf("save space: %w", errSave)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	d.log.InfoContext(
		ctx,
		"space deactivated",
		slog.String("space_id", in.SpaceID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)
	span.SetStatus(codes.Ok, "space successfully deactivated")

	return nil
}
