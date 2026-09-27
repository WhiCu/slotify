package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type ListUsers struct {
	log   *slog.Logger
	users UserRepository
}

func NewListUsers(
	log *slog.Logger,
	users UserRepository,
) *ListUsers {
	return &ListUsers{
		log:   log,
		users: users,
	}
}

type ListUsersItem struct {
	ID        domainuser.UserID
	Role      string
	CreatedAt string
}

type ListUsersOutput struct {
	Users []ListUsersItem
}

func (l *ListUsers) Execute(ctx context.Context, actorID domainuser.UserID) (*ListUsersOutput, error) {
	l.log.DebugContext(ctx, "executing list users", slog.String("actor_id", actorID.String()))

	actor, err := l.users.FindByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			l.log.WarnContext(ctx, "actor user not found", slog.String("actor_id", actorID.String()))
			return nil, ErrUserNotFound
		}
		l.log.ErrorContext(ctx, "failed to find actor",
			slog.String("actor_id", actorID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("find actor: %w", err)
	}

	if !actor.IsAdmin() {
		l.log.WarnContext(ctx, "non-admin attempted to list users",
			slog.String("actor_id", actorID.String()),
			slog.String("actor_role", actor.Role().String()),
		)
		return nil, ErrNotAdmin
	}

	all, err := l.users.ListAll(ctx)
	if err != nil {
		l.log.ErrorContext(ctx, "failed to list users", slog.Any("error", err))
		return nil, fmt.Errorf("list users: %w", err)
	}

	items := make([]ListUsersItem, 0, len(all))
	for _, u := range all {
		items = append(items, ListUsersItem{
			ID:        u.ID(),
			Role:      u.Role().String(),
			CreatedAt: u.CreatedAt().Format(time.RFC3339),
		})
	}

	l.log.InfoContext(ctx, "users listed",
		slog.String("actor_id", actorID.String()),
		slog.Int("count", len(items)),
	)

	return &ListUsersOutput{Users: items}, nil
}
