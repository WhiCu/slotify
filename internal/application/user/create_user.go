package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var (
	tracer = meta.Tracer
	meter  = meta.Meter
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
) (output *CreateUserOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "user", "create")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "CreateUser.Execute",
		trace.WithAttributes(
			attribute.String("domain", "user"),
			attribute.String("usecase", "create"),
		),
	)
	defer span.End()

	c.log.DebugContext(ctx, "executing create user")

	var (
		id        domainuser.UserID
		role      domainuser.Role
		createdAt time.Time
	)

	err = c.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetAttributes(
		attribute.String("user.id", id.String()),
		attribute.String("user.role", role.String()),
	)

	c.log.InfoContext(
		ctx,
		"user created",
		slog.String("created_user_id", id.String()),
		slog.String("role", role.String()),
	)

	token, err := c.tokenIssuer.Encode(map[string]any{
		"user_id": id.String(),
		"role":    role.String(),
	}, c.ttl)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("issue token: %w", err)
	}

	c.log.DebugContext(
		ctx,
		"issued token for user",
		slog.String("created_user_id", id.String()),
		slog.String("role", role.String()),
		slog.Int64("ttl", c.ttl.Nanoseconds()),
	)
	span.SetStatus(codes.Ok, "user successfully created")

	return &CreateUserOutput{
		ID:        id,
		Role:      role.String(),
		CreatedAt: createdAt,
		Token:     token,
	}, nil
}
