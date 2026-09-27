package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainuser "github.com/whicu/slotify/internal/domain/user"
)

type GetUser struct {
	log   *slog.Logger
	users UserRepository
}

func NewGetUser(
	log *slog.Logger,
	users UserRepository,
) *GetUser {
	return &GetUser{
		log:   log,
		users: users,
	}
}

type GetUserOutput struct {
	ID        domainuser.UserID
	Role      string
	CreatedAt string
}

func (g *GetUser) Execute(ctx context.Context, id domainuser.UserID) (*GetUserOutput, error) {
	g.log.DebugContext(ctx, "executing get user", slog.String("user_id", id.String()))

	u, err := g.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			g.log.WarnContext(ctx, "user not found", slog.String("user_id", id.String()))
			return nil, ErrUserNotFound
		}
		g.log.ErrorContext(ctx, "failed to find user",
			slog.String("user_id", id.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("find user: %w", err)
	}

	return &GetUserOutput{
		ID:        u.ID(),
		Role:      u.Role().String(),
		CreatedAt: u.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
