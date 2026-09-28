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
) (*GetUserOutput, error) {
	g.log.DebugContext(
		ctx,
		"executing get user",
		slog.String("user_id", id.String()),
	)

	u, err := g.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user: %w", err)
	}

	return &GetUserOutput{
		ID:        u.ID(),
		Role:      u.Role().String(),
		CreatedAt: u.CreatedAt(),
	}, nil
}
