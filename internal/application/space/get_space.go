package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type GetSpace struct {
	log    *slog.Logger
	spaces SpaceRepository
}

func NewGetSpace(
	log *slog.Logger,
	spaces SpaceRepository,
) *GetSpace {
	return &GetSpace{
		log:    log,
		spaces: spaces,
	}
}

type GetSpaceOutput struct {
	ID        domainspace.SpaceID
	Name      string
	Type      string
	Capacity  int
	Active    bool
	CreatedAt string
}

func (g *GetSpace) Execute(ctx context.Context, id domainspace.SpaceID) (*GetSpaceOutput, error) {
	g.log.DebugContext(ctx, "executing get space", slog.String("space_id", id.String()))

	s, err := g.spaces.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			g.log.WarnContext(ctx, "space not found", slog.String("space_id", id.String()))
			return nil, ErrSpaceNotFound
		}
		g.log.ErrorContext(ctx, "failed to find space",
			slog.String("space_id", id.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("find space: %w", err)
	}

	return &GetSpaceOutput{
		ID:        s.ID(),
		Name:      s.Name(),
		Type:      s.Type().String(),
		Capacity:  s.Capacity(),
		Active:    s.IsActive(),
		CreatedAt: s.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
