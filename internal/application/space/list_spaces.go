package space

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/whicu/slotify/internal/application/meta"
	domainspace "github.com/whicu/slotify/internal/domain/space"
)

type ListSpacesLister interface {
	List(
		ctx context.Context,
		onlyActive bool,
	) ([]*domainspace.Space, error)
}

type ListSpaces struct {
	log    *slog.Logger
	spaces ListSpacesLister

	// metrics
	spacesListed metric.Int64Histogram
}

func NewListSpaces(
	log *slog.Logger,
	spaces ListSpacesLister,
) *ListSpaces {
	spacesListed, err := meter.Int64Histogram(
		"slotify.space.list.count",
		metric.WithDescription("Number of spaces returned in space listing"),
		metric.WithUnit("{space}"),
	)
	if err != nil {
		panic(err)
	}

	return &ListSpaces{
		log:          log,
		spaces:       spaces,
		spacesListed: spacesListed,
	}
}

type ListSpacesInput struct {
	OnlyActive bool
}

type ListSpacesItem struct {
	ID        domainspace.SpaceID
	Name      string
	Type      string
	Capacity  int
	Active    bool
	CreatedAt time.Time
}

type ListSpacesOutput struct {
	Spaces []ListSpacesItem
}

func (l *ListSpaces) Execute(
	ctx context.Context,
	in ListSpacesInput,
) (output *ListSpacesOutput, err error) {
	ctx, endObserve := meta.ObserveUseCase(ctx, "space", "list")
	defer func() { endObserve(err) }()

	ctx, span := tracer.Start(ctx, "ListSpaces.Execute",
		trace.WithAttributes(
			attribute.String("domain", "space"),
			attribute.String("usecase", "list"),
			attribute.Bool("only_active", in.OnlyActive),
		),
	)
	defer span.End()
	l.log.DebugContext(
		ctx,
		"executing list spaces",
		slog.Bool("only_active", in.OnlyActive),
	)

	all, err := l.spaces.List(ctx, in.OnlyActive)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list spaces: %w", err)
	}

	items := make([]ListSpacesItem, 0, len(all))

	for _, s := range all {
		items = append(items, ListSpacesItem{
			ID:        s.ID(),
			Name:      s.Name(),
			Type:      s.Type().String(),
			Capacity:  s.Capacity(),
			Active:    s.IsActive(),
			CreatedAt: s.CreatedAt(),
		})
	}

	l.log.InfoContext(
		ctx,
		"spaces listed",
		slog.Int("count", len(items)),
		slog.Bool("only_active", in.OnlyActive),
	)
	span.SetAttributes(attribute.Int("spaces.count", len(items)))
	span.SetStatus(codes.Ok, "spaces successfully listed")

	l.spacesListed.Record(ctx, int64(len(items)))

	return &ListSpacesOutput{
		Spaces: items,
	}, nil
}
