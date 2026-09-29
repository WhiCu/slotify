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

type TokenIssuer interface {
	Encode(payload map[string]any, ttl time.Duration) (string, error)
}

type CreateUser struct {
	log         *slog.Logger
	ids         IDGenerator
	counter     CreateUserCounter
	saver       CreateUserSaver
	rootLocker  CreateUserRootLocker
	transactor  Transactor
	tokenIssuer TokenIssuer
	ttl         time.Duration
}

func NewCreateUser(
	log *slog.Logger,
	ids IDGenerator,
	counter CreateUserCounter,
	saver CreateUserSaver,
	rootLocker CreateUserRootLocker,
	transactor Transactor,
	tokenIssuer TokenIssuer,
	ttl time.Duration,
) *CreateUser {
	return &CreateUser{
		log:         log,
		ids:         ids,
		counter:     counter,
		saver:       saver,
		rootLocker:  rootLocker,
		transactor:  transactor,
		tokenIssuer: tokenIssuer,
		ttl:         ttl,
	}
}

type CreateUserInput struct {
	Role string
}

type CreateUserOutput struct {
	ID        domainuser.UserID
	Role      string
	CreatedAt time.Time
	Token     string
}

func (c *CreateUser) Execute(
	ctx context.Context,
	in CreateUserInput,
) (*CreateUserOutput, error) {
	c.log.DebugContext(ctx, "executing create user")

	var (
		id        domainuser.UserID
		role      domainuser.Role
		createdAt time.Time
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
		now := time.Now()
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
		createdAt = u.CreatedAt()

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

	token, err := c.tokenIssuer.Encode(map[string]any{
		"user_id": id.String(),
		"role":    role.String(),
	}, c.ttl)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	c.log.DebugContext(
		ctx,
		"issued token for user",
		slog.String("user_id", id.String()),
		slog.String("role", role.String()),
		slog.Int64("ttl_seconds", c.ttl.Nanoseconds()),
	)

	return &CreateUserOutput{
		ID:        id,
		Role:      role.String(),
		CreatedAt: createdAt,
		Token:     token,
	}, nil
}
