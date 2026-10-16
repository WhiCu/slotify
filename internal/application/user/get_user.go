package user

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
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var ErrUserNotFound = errors.New(
	"application/user: user not found",
)

type GetUserFinder interface {
	FindByID(ctx context.Context, id domainuser.UserID) (*domainuser.User, error)
}

type GetUser struct {
	log   *slog.Logger
	users GetUserFinder
}

func NewGetUser(
	log *slog.Logger,
	users GetUserFinder,
) *GetUser {
	return &GetUser{
		log:   log,
		users: users,
	}
}

type GetUserOutput struct {
	ID        domainuser.UserID
	Role      string
	CreatedAt time.Time
}

func (g *GetUser) Execute(
	ctx context.Context,
	id domainuser.UserID,
) (output *GetUserOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "user", "get")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "GetUser.Execute",
		trace.WithAttributes(
			attribute.String("domain", "user"),
			attribute.String("usecase", "get"),
			attribute.String("user_id", id.String()),
		),
	)
	defer span.End()
	g.log.DebugContext(
		ctx,
		"executing get user",
		slog.String("user_id", id.String()),
	)

	u, err := g.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			span.RecordError(ErrUserNotFound)
			span.SetStatus(codes.Error, ErrUserNotFound.Error())
			return nil, ErrUserNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("find user: %w", err)
	}

	span.SetAttributes(
		attribute.String("user.id", u.ID().String()),
		attribute.String("user.role", u.Role().String()),
	)
	span.SetStatus(codes.Ok, "user successfully retrieved")

	return &GetUserOutput{
		ID:        u.ID(),
		Role:      u.Role().String(),
		CreatedAt: u.CreatedAt(),
	}, nil
}
