package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	tracer = meta.Tracer
	meter  = meta.Meter
)

var (
	ErrSpaceNotFound = errors.New("application/space: space not found")
	ErrNotAdmin      = errors.New("application/space: only admin can perform this action")
	ErrUserNotFound  = errors.New("application/space: actor not found")
)

type IDGenerator interface {
	NewID() domainspace.SpaceID
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}

type CreateSpaceUserFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type CreateSpaceSaver interface {
	Save(ctx context.Context, s *domainspace.Space) error
}

type CreateSpace struct {
	log        *slog.Logger
	ids        IDGenerator
	users      CreateSpaceUserFinder
	spaces     CreateSpaceSaver
	transactor Transactor

	// metrics
	capacityAdded metric.Int64Counter
}

func NewCreateSpace(
	log *slog.Logger,
	ids IDGenerator,
	users CreateSpaceUserFinder,
	spaces CreateSpaceSaver,
	transactor Transactor,
) *CreateSpace {
	capacityAdded, err := meter.Int64Counter(
		"slotify.space.capacity.added",
		metric.WithDescription("Total capacity of spaces added to the system"),
		metric.WithUnit("{seat}"),
	)
	if err != nil {
		panic(err)
	}

	return &CreateSpace{
		log:           log,
		ids:           ids,
		users:         users,
		spaces:        spaces,
		transactor:    transactor,
		capacityAdded: capacityAdded,
	}
}

type CreateSpaceInput struct {
	Name     string
	Type     string
	Capacity int
	ActorID  domainuser.UserID
}

type CreateSpaceOutput struct {
	ID        domainspace.SpaceID
	Name      string
	Type      string
	Capacity  int
	Active    bool
	CreatedAt time.Time
}

func (c *CreateSpace) Execute(
	ctx context.Context,
	in CreateSpaceInput,
) (output *CreateSpaceOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "space", "create")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "CreateSpace.Execute",
		trace.WithAttributes(
			attribute.String("domain", "space"),
			attribute.String("usecase", "create"),
			attribute.String("name", in.Name),
			attribute.String("actor_id", in.ActorID.String()),
			attribute.String("space_type", in.Type),
			attribute.Int("capacity", in.Capacity),
		),
	)
	defer span.End()
	c.log.DebugContext(
		ctx,
		"executing create space",
		slog.String("name", in.Name),
		slog.String("actor_id", in.ActorID.String()),
	)

	spaceType, err := domainspace.TypeFromString(in.Type)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, domain.ErrInvalidArgument(err)
	}

	var out CreateSpaceOutput

	err = c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, errFind := c.users.FindByIDForUpdate(ctx, in.ActorID)
		if errFind != nil {
			if errors.Is(errFind, domain.ErrNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("find actor: %w", errFind)
		}

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		s, errNewSpace := domainspace.New(
			c.ids.NewID(),
			in.Name,
			spaceType,
			in.Capacity,
			time.Now(),
		)
		if errNewSpace != nil {
			return errNewSpace
		}

		if errSave := c.spaces.Save(ctx, s); errSave != nil {
			return fmt.Errorf("save space: %w", errSave)
		}

		out = CreateSpaceOutput{
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

	c.log.InfoContext(
		ctx,
		"space created",
		slog.String("space_id", out.ID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)
	span.SetAttributes(
		attribute.String("space.id", out.ID.String()),
		attribute.String("space.type", out.Type),
		attribute.Int("space.capacity", out.Capacity),
		attribute.Bool("space.active", out.Active),
	)
	span.SetStatus(codes.Ok, "space successfully created")

	c.capacityAdded.Add(ctx, int64(out.Capacity))

	return &out, nil
}
