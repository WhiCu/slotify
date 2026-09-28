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

var (
	ErrSpaceNotFound = errors.New("application/space: space not found")
	ErrNotAdmin      = errors.New("application/space: only admin can perform this action")
	ErrUserNotFound  = errors.New("application/space: actor not found")
)

type IDGenerator interface {
	NewID() domainspace.SpaceID
}

type Clock interface {
	Now() time.Time
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
	clock      Clock
	users      CreateSpaceUserFinder
	spaces     CreateSpaceSaver
	transactor Transactor
}

func NewCreateSpace(
	log *slog.Logger,
	ids IDGenerator,
	clock Clock,
	users CreateSpaceUserFinder,
	spaces CreateSpaceSaver,
	transactor Transactor,
) *CreateSpace {
	return &CreateSpace{
		log:        log,
		ids:        ids,
		clock:      clock,
		users:      users,
		spaces:     spaces,
		transactor: transactor,
	}
}

type CreateSpaceInput struct {
	Name     string
	Type     string
	Capacity int
	ActorID  domainuser.UserID
}

type CreateSpaceOutput struct {
	ID       domainspace.SpaceID
	Name     string
	Type     string
	Capacity int
}

func (c *CreateSpace) Execute(
	ctx context.Context,
	in CreateSpaceInput,
) (*CreateSpaceOutput, error) {
	c.log.DebugContext(
		ctx,
		"executing create space",
		slog.String("name", in.Name),
		slog.String("actor_id", in.ActorID.String()),
	)

	spaceType, err := domainspace.TypeFromString(in.Type)
	if err != nil {
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
			c.clock.Now(),
		)
		if errNewSpace != nil {
			return errNewSpace
		}

		if errSave := c.spaces.Save(ctx, s); errSave != nil {
			return fmt.Errorf("save space: %w", errSave)
		}

		out = CreateSpaceOutput{
			ID:       s.ID(),
			Name:     s.Name(),
			Type:     s.Type().String(),
			Capacity: s.Capacity(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	c.log.InfoContext(
		ctx,
		"space created",
		slog.String("space_id", out.ID.String()),
		slog.String("actor_id", in.ActorID.String()),
	)

	return &out, nil
}
