package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type IDGenerator interface {
	NewID() uuid.UUID
}

type Clock interface {
	Now() time.Time
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}

type CreateUserRootLocker interface {
	LockRootCreation(ctx context.Context) error
}

type CreateUserCounter interface {
	Count(ctx context.Context) (int64, error)
}

type CreateUserSaver interface {
	Save(ctx context.Context, u *domainuser.User) error
}

type CreateUser struct {
	log        *slog.Logger
	ids        IDGenerator
	clock      Clock
	counter    CreateUserCounter
	saver      CreateUserSaver
	rootLocker CreateUserRootLocker
	transactor Transactor
}

func NewCreateUser(
	log *slog.Logger,
	ids IDGenerator,
	clock Clock,
	counter CreateUserCounter,
	saver CreateUserSaver,
	rootLocker CreateUserRootLocker,
	transactor Transactor,
) *CreateUser {
	return &CreateUser{
		log:        log,
		ids:        ids,
		clock:      clock,
		counter:    counter,
		saver:      saver,
		rootLocker: rootLocker,
		transactor: transactor,
	}
}

type CreateUserInput struct {
	Role string
}

type CreateUserOutput struct {
	ID   domainuser.UserID
	Role string
}

func (c *CreateUser) Execute(
	ctx context.Context,
	in CreateUserInput,
) (*CreateUserOutput, error) {
	c.log.DebugContext(ctx, "executing create user")

	var (
		id   domainuser.UserID
		role domainuser.Role
	)

	err := c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := c.rootLocker.LockRootCreation(ctx); err != nil {
			return err
		}

		count, err := c.counter.Count(ctx)
		if err != nil {
			return fmt.Errorf("count users: %w", err)
		}

		id = c.ids.NewID()
		now := c.clock.Now()

		var u *domainuser.User

		if count == 0 {
			u, err = domainuser.NewRoot(id, now)
			if err != nil {
				return fmt.Errorf("create root user: %w", err)
			}
		} else {
			role = domainuser.Member

			if in.Role != "" {
				role, err = domainuser.RoleFromString(in.Role)
				if err != nil {
					return domain.ErrInvalidArgument(err)
				}
			}

			u, err = domainuser.New(id, role, now)
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
		}

		role = u.Role()

		if errSave := c.saver.Save(ctx, u); errSave != nil {
			return fmt.Errorf("save user: %w", errSave)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	c.log.InfoContext(
		ctx,
		"user created",
		slog.String("user_id", id.String()),
		slog.String("role", role.String()),
	)

	return &CreateUserOutput{
		ID:   id,
		Role: role.String(),
	}, nil
}
