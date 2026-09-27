package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	ErrUserNotFound     = errors.New("application/user: user not found")
	ErrNotAdmin         = errors.New("application/user: only admin can perform this action")
	ErrHasReservations  = errors.New("application/user: user has active reservations")
	ErrCannotDeleteRoot = errors.New("application/user: cannot delete root user")
	ErrCannotDeleteSelf = errors.New("application/user: cannot delete yourself")
)

type IDGenerator interface {
	NewID() uuid.UUID
}

type UserRepository interface {
	Save(ctx context.Context, u *domainuser.User) error
	FindByID(ctx context.Context, id domainuser.UserID) (*domainuser.User, error)
	ListAll(ctx context.Context) ([]*domainuser.User, error)
	Delete(ctx context.Context, id domainuser.UserID) error
	Count(ctx context.Context) (int64, error)
}

type ReservationRepository interface {
	CountActiveByUserID(ctx context.Context, userID domainuser.UserID) (int64, error)
	CancelAllByUserID(ctx context.Context, userID domainuser.UserID) error
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type CreateUser struct {
	log        *slog.Logger
	ids        IDGenerator
	users      UserRepository
	transactor Transactor
}

func NewCreateUser(
	log *slog.Logger,
	ids IDGenerator,
	users UserRepository,
	transactor Transactor,
) *CreateUser {
	return &CreateUser{
		log:        log,
		ids:        ids,
		users:      users,
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

func (c *CreateUser) Execute(ctx context.Context, in CreateUserInput) (*CreateUserOutput, error) {
	c.log.DebugContext(ctx, "executing create user")

	var id domainuser.UserID
	var role domainuser.Role

	err := c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		now := time.Now()

		count, err := c.users.Count(ctx)
		if err != nil {
			c.log.ErrorContext(ctx, "failed to count users", slog.Any("error", err))
			return fmt.Errorf("count users: %w", err)
		}

		id = c.ids.NewID()

		var u *domainuser.User

		if count == 0 {
			u, err = domainuser.NewRoot(id, now)
			if err != nil {
				c.log.ErrorContext(ctx, "failed to create root user", slog.Any("error", err))
				return fmt.Errorf("create root user: %w", err)
			}
			c.log.InfoContext(ctx, "creating root admin user", slog.String("user_id", id.String()))
		} else {
			role = domainuser.Member
			if in.Role != "" {
				role, err = domainuser.RoleFromString(in.Role)
				if err != nil {
					c.log.WarnContext(ctx, "invalid role provided",
						slog.String("role", in.Role),
						slog.Any("error", err),
					)
					return domain.ErrInvalidArgument(err)
				}
			}

			u, err = domainuser.New(id, role, now)
			if err != nil {
				c.log.ErrorContext(ctx, "failed to create user domain entity", slog.Any("error", err))
				return fmt.Errorf("create user domain entity: %w", err)
			}
		}
		role = u.Role()

		if err = c.users.Save(ctx, u); err != nil {
			c.log.ErrorContext(ctx, "failed to save user",
				slog.String("user_id", u.ID().String()),
				slog.Any("error", err),
			)
			return fmt.Errorf("save user: %w", err)
		}
		return nil
	})

	if err != nil {
		c.log.ErrorContext(ctx, "failed to create user", slog.Any("error", err))
		return nil, err
	}

	c.log.InfoContext(ctx, "user created",
		slog.String("user_id", id.String()),
		slog.String("role", role.String()),
	)

	return &CreateUserOutput{
		ID:   id,
		Role: role.String(),
	}, nil
}
