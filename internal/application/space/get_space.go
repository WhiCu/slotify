package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/slotify/internal/domain"
	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type GetSpaceFinder interface {
	FindByID(
		ctx context.Context,
		id domainspace.SpaceID,
	) (*domainspace.Space, error)
}

type GetSpace struct {
	log    *slog.Logger
	spaces GetSpaceFinder
}

func NewGetSpace(
	log *slog.Logger,
	spaces GetSpaceFinder,
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
	CreatedAt time.Time
}

func (g *GetSpace) Execute(
	ctx context.Context,
	id domainspace.SpaceID,
) (*GetSpaceOutput, error) {
	g.log.DebugContext(
		ctx,
		"executing get space",
		slog.String("space_id", id.String()),
	)

	s, err := g.spaces.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrSpaceNotFound
		}

		return nil, fmt.Errorf("find space: %w", err)
	}

	return &GetSpaceOutput{
		ID:        s.ID(),
		Name:      s.Name(),
		Type:      s.Type().String(),
		Capacity:  s.Capacity(),
		Active:    s.IsActive(),
		CreatedAt: s.CreatedAt(),
	}, nil
}
