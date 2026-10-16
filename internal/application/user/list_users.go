package user

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
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

var ErrNotAdmin = errors.New(
	"application/user: only admin can perform this action",
)

type ListUsersFinder interface {
	FindByIDForUpdate(
		ctx context.Context,
		id domainuser.UserID,
	) (*domainuser.User, error)
}

type ListUsersLister interface {
	ListAll(ctx context.Context) ([]*domainuser.User, error)
}

type ListUsers struct {
	log        *slog.Logger
	users      ListUsersFinder
	lister     ListUsersLister
	transactor Transactor

	// metrics
	usersListed metric.Int64Histogram
}

func NewListUsers(
	log *slog.Logger,
	users ListUsersFinder,
	lister ListUsersLister,
	transactor Transactor,
) *ListUsers {
	usersListed, err := meter.Int64Histogram(
		"slotify.user.list.count",
		metric.WithDescription("Number of users returned in user list query"),
		metric.WithUnit("{user}"),
	)
	if err != nil {
		panic(err)
	}

	return &ListUsers{
		log:         log,
		users:       users,
		lister:      lister,
		transactor:  transactor,
		usersListed: usersListed,
	}
}

type ListUsersItem struct {
	ID        domainuser.UserID
	Role      string
	CreatedAt time.Time
}

type ListUsersOutput struct {
	Users []ListUsersItem
}

func (l *ListUsers) Execute(
	ctx context.Context,
	actorID domainuser.UserID,
) (output *ListUsersOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "user", "list")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "ListUsers.Execute",
		trace.WithAttributes(
			attribute.String("domain", "user"),
			attribute.String("usecase", "list"),
			attribute.String("actor_id", actorID.String()),
		),
	)
	defer span.End()

	l.log.DebugContext(
		ctx,
		"executing list users",
		slog.String("actor_id", actorID.String()),
	)

	var users []*domainuser.User

	err = l.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		actor, err := l.users.FindByIDForUpdate(ctx, actorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("find actor: %w", err)
		}

		if !actor.IsAdmin() {
			return ErrNotAdmin
		}

		users, err = l.lister.ListAll(ctx)
		if err != nil {
			return fmt.Errorf("list users: %w", err)
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	items := make([]ListUsersItem, 0, len(users))

	for _, u := range users {
		items = append(items, ListUsersItem{
			ID:        u.ID(),
			Role:      u.Role().String(),
			CreatedAt: u.CreatedAt(),
		})
	}

	l.log.InfoContext(
		ctx,
		"users listed",
		slog.String("actor_id", actorID.String()),
		slog.Int("count", len(items)),
	)
	span.SetAttributes(attribute.Int("users.count", len(items)))
	span.SetStatus(codes.Ok, "users successfully listed")

	l.usersListed.Record(ctx, int64(len(items)))

	return &ListUsersOutput{
		Users: items,
	}, nil
}
