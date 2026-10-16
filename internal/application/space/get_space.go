package space

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
) (output *GetSpaceOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "space", "get")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "GetSpace.Execute",
		trace.WithAttributes(
			attribute.String("domain", "space"),
			attribute.String("usecase", "get"),
			attribute.String("space_id", id.String()),
		),
	)
	defer span.End()
	g.log.DebugContext(
		ctx,
		"executing get space",
		slog.String("space_id", id.String()),
	)

	s, err := g.spaces.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			span.RecordError(ErrSpaceNotFound)
			span.SetStatus(codes.Error, ErrSpaceNotFound.Error())
			return nil, ErrSpaceNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("find space: %w", err)
	}

	span.SetAttributes(
		attribute.String("space.name", s.Name()),
		attribute.String("space.type", s.Type().String()),
		attribute.Int("space.capacity", s.Capacity()),
		attribute.Bool("space.active", s.IsActive()),
	)
	span.SetStatus(codes.Ok, "space successfully retrieved")

	return &GetSpaceOutput{
		ID:        s.ID(),
		Name:      s.Name(),
		Type:      s.Type().String(),
		Capacity:  s.Capacity(),
		Active:    s.IsActive(),
		CreatedAt: s.CreatedAt(),
	}, nil
}
