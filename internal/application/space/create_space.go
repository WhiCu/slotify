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

type SpaceRepository interface {
	Save(ctx context.Context, s *domainspace.Space) error
	FindByID(ctx context.Context, id domainspace.SpaceID) (*domainspace.Space, error)
	ListAll(ctx context.Context) ([]*domainspace.Space, error)
	ListActive(ctx context.Context) ([]*domainspace.Space, error)
	Delete(ctx context.Context, id domainspace.SpaceID) error
}

type UserRepository interface {
	FindByID(ctx context.Context, id domainuser.UserID) (*domainuser.User, error)
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type CreateSpace struct {
	log        *slog.Logger
	ids        IDGenerator
	spaces     SpaceRepository
	users      UserRepository
	transactor Transactor
}

func NewCreateSpace(
	log *slog.Logger,
	ids IDGenerator,
	spaces SpaceRepository,
	users UserRepository,
	transactor Transactor,
) *CreateSpace {
	return &CreateSpace{
		log:        log,
		ids:        ids,
		spaces:     spaces,
		users:      users,
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

func (c *CreateSpace) Execute(ctx context.Context, in CreateSpaceInput) (*CreateSpaceOutput, error) {
	c.log.DebugContext(ctx, "executing create space",
		slog.String("name", in.Name),
		slog.String("actor_id", in.ActorID.String()),
	)

	spaceType, err := domainspace.TypeFromString(in.Type)
	if err != nil {
		c.log.WarnContext(ctx, "invalid space type",
			slog.String("type", in.Type),
			slog.Any("error", err),
		)
		return nil, domain.ErrInvalidArgument(err)
	}

	var id domainspace.SpaceID

	err = c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, txErr := c.users.FindByID(ctx, in.ActorID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrUserNotFound
			}
			return fmt.Errorf("find actor: %w", txErr)
		}

		if !actor.IsAdmin() {
			c.log.WarnContext(ctx, "non-admin attempted to create space",
				slog.String("actor_id", in.ActorID.String()),
			)
			return ErrNotAdmin
		}

		id = c.ids.NewID()
		now := time.Now()

		s, createErr := domainspace.New(id, in.Name, spaceType, in.Capacity, now)
		if createErr != nil {
			c.log.ErrorContext(ctx, "failed to create space domain entity", slog.Any("error", createErr))
			return createErr
		}

		if saveErr := c.spaces.Save(ctx, s); saveErr != nil {
			c.log.ErrorContext(ctx, "failed to save space",
				slog.String("space_id", s.ID().String()),
				slog.Any("error", saveErr),
			)
			return fmt.Errorf("save space: %w", saveErr)
		}

		return nil
	})

	if err != nil {
		c.log.ErrorContext(ctx, "create space failed", slog.Any("error", err))
		return nil, err
	}

	c.log.InfoContext(ctx, "space created",
		slog.String("space_id", id.String()),
		slog.String("name", in.Name),
	)

	return &CreateSpaceOutput{
		ID:       id,
		Name:     in.Name,
		Type:     in.Type,
		Capacity: in.Capacity,
	}, nil
}
